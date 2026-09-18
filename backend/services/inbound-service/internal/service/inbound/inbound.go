package inbound

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"strings"
	"time"

	"github.com/google/uuid"

	"order-fill/backend/services/inbound-service/internal/domain"
)

type Store interface {
	GetSettings(ctx context.Context) (domain.Settings, error)
	UpsertSettings(ctx context.Context, enabled bool, receiveAddress string) (domain.Settings, error)
	IncrWebhookCount(ctx context.Context, isError bool)
	GetCompanyInbound(ctx context.Context, companyID string) (domain.CompanyInbound, error)
	GetCompanyByAllowedSender(ctx context.Context, senderEmail string) (domain.CompanyInbound, error)
	UpsertCompanyInbound(ctx context.Context, companyID, receiveAddress, senderEmail string, enabled bool) (domain.CompanyInbound, error)
	MessageExists(ctx context.Context, providerMessageID string) (bool, error)
	SaveMessage(ctx context.Context, msg domain.MessageSummary, attachments []domain.Attachment) error
	ListMessages(ctx context.Context, companyID string, limit int) ([]domain.MessageSummary, error)
	GetMessage(ctx context.Context, messageID string) (domain.MessageSummary, error)
	GetMessageAttachments(ctx context.Context, messageID string) ([]domain.Attachment, error)
	GetAttachment(ctx context.Context, attachmentID string) (domain.Attachment, error)
	GetMessageCompany(ctx context.Context, messageID string) (string, error)
	ListDeliveries(ctx context.Context, limit int) ([]domain.MessageSummary, error)
}

type ObjectStore interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) ([]byte, string, error)
}

type Service struct {
	store      Store
	objects    ObjectStore
	inboundKey string
}

func New(store Store, objects ObjectStore, inboundKey string) *Service {
	return &Service{store: store, objects: objects, inboundKey: inboundKey}
}

func (s *Service) GetSettings(ctx context.Context) (domain.Settings, error) {
	return s.store.GetSettings(ctx)
}

func (s *Service) UpdateSettings(ctx context.Context, enabled bool, receiveAddress string, actorUserID string) (domain.Settings, error) {
	return s.store.UpsertSettings(ctx, enabled, receiveAddress)
}

func (s *Service) GetCompanyInbound(ctx context.Context, companyID string) (domain.CompanyInbound, error) {
	return s.store.GetCompanyInbound(ctx, companyID)
}

func (s *Service) UpdateCompanyInbound(ctx context.Context, companyID, receiveAddress, senderEmail string, enabled bool) (domain.CompanyInbound, error) {
	receiveAddress = strings.TrimSpace(receiveAddress)
	senderEmail = strings.TrimSpace(senderEmail)
	if senderEmail == "" {
		return domain.CompanyInbound{}, domain.ErrInvalid
	}
	return s.store.UpsertCompanyInbound(ctx, companyID, receiveAddress, senderEmail, enabled)
}

func (s *Service) ListMessages(ctx context.Context, companyID string, limit int) ([]domain.MessageSummary, error) {
	return s.store.ListMessages(ctx, companyID, limit)
}

func (s *Service) GetMessage(ctx context.Context, companyID, messageID string) (domain.MessageSummary, error) {
	msg, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return domain.MessageSummary{}, err
	}
	if msg.CompanyID != companyID {
		return domain.MessageSummary{}, domain.ErrUnauthorized
	}
	return msg, nil
}

func (s *Service) GetMessageAttachments(ctx context.Context, messageID string) ([]domain.Attachment, error) {
	return s.store.GetMessageAttachments(ctx, messageID)
}

func (s *Service) GetMessageFile(ctx context.Context, companyID, messageID, attachmentID string) (domain.Attachment, []byte, error) {
	msgCompany, err := s.store.GetMessageCompany(ctx, messageID)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	if companyID != msgCompany {
		return domain.Attachment{}, nil, domain.ErrUnauthorized
	}
	att, err := s.store.GetAttachment(ctx, attachmentID)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	if att.MessageID != messageID {
		return domain.Attachment{}, nil, domain.ErrNotFound
	}
	data, _, err := s.objects.Get(ctx, att.ObjectKey)
	if err != nil {
		return domain.Attachment{}, nil, fmt.Errorf("read inbound attachment object: %w", err)
	}
	return att, data, nil
}

func (s *Service) ListDeliveries(ctx context.Context, limit int) ([]domain.MessageSummary, error) {
	return s.store.ListDeliveries(ctx, limit)
}

type rawAttachment struct {
	FileName    string `json:"file_name"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
	ContentID   string `json:"content_id"`
}

type rawWebhook struct {
	Envelope struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"envelope"`
	Headers            json.RawMessage `json:"headers"`
	Subject            string          `json:"subject"`
	Plain              string          `json:"plain"`
	HTML               string          `json:"html"`
	MessageID          string          `json:"message_id"`
	Attachments        []rawAttachment `json:"attachments"`
	AttachmentQuantity int             `json:"attachment_quantity"`
}

const maxAttachmentBytes = 20 << 20

func (s *Service) parseWebhookPayload(rawPayload []byte) (rawWebhook, error) {
	if len(rawPayload) > 32<<20 {
		return rawWebhook{}, domain.ErrPayloadTooLarge
	}

	var mail rawWebhook
	if err := json.Unmarshal(rawPayload, &mail); err != nil {
		return rawWebhook{}, fmt.Errorf("parse inbound webhook: %w: %v", domain.ErrInvalidPayload, err)
	}
	return mail, nil
}

func (s *Service) resolveProviderMessageID(mail rawWebhook) string {
	if id := strings.TrimSpace(mail.MessageID); id != "" {
		return id
	}
	if id := messageIDFromHeaders(mail.Headers); strings.TrimSpace(id) != "" {
		return id
	}
	return generateFallbackID()
}

func (s *Service) resolveMessageSubject(mail rawWebhook) string {
	if s := subjectFromHeaders(mail.Headers); s != "" {
		return s
	}
	if mail.Subject != "" {
		return mail.Subject
	}
	return ""
}

func (s *Service) IngestWebhook(ctx context.Context, rawPayload []byte) error {
	mail, err := s.parseWebhookPayload(rawPayload)
	if err != nil {
		return err
	}

	providerID := s.resolveProviderMessageID(mail)

	exists, err := s.store.MessageExists(ctx, providerID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	settings, err := s.store.GetSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		s.store.IncrWebhookCount(ctx, true)
		return fmt.Errorf("inbound reception is disabled")
	}

	subject := s.resolveMessageSubject(mail)
	bodyText, bodyHTML := resolveMessageBody(mail)

	inbound, err := s.store.GetCompanyByAllowedSender(ctx, mail.Envelope.From)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return s.saveMinimalMessage(ctx, providerID, subject, bodyText, bodyHTML, mail, domain.StatusErrorUnknownAddress, "unknown_address")
		}
		return err
	}

	msgID := generateID()
	storedAttachments, err := s.storeAttachments(ctx, inbound.CompanyID, msgID, mail.Attachments)
	if err != nil {
		if errors.Is(err, domain.ErrAttachmentTooLarge) {
			return s.saveMinimalMessage(ctx, providerID, subject, bodyText, bodyHTML, mail, domain.StatusErrorTooLarge, "too_large")
		}
		return err
	}

	if len(storedAttachments) == 0 {
		errorCode := "no_attachments"
		if len(filterAttachments(mail.Attachments)) > 0 {
			errorCode = "invalid_attachments"
		}
		return s.saveMinimalMessage(ctx, providerID, subject, bodyText, bodyHTML, mail, domain.StatusErrorNoAttachments, errorCode)
	}

	var totalBytes int64
	for _, att := range storedAttachments {
		totalBytes += att.Size
	}

	msg := domain.MessageSummary{
		ID:                msgID,
		ProviderMessageID: providerID,
		Subject:           subject,
		EnvelopeFrom:      mail.Envelope.From,
		EnvelopeTo:        mail.Envelope.To,
		ReceivedAt:        time.Now().UTC(),
		CompanyID:         inbound.CompanyID,
		Status:            domain.StatusReceived,
		AttachmentCount:   count32(len(storedAttachments)),
		TotalBytes:        totalBytes,
		BodyText:          bodyText,
		BodyHTML:          bodyHTML,
	}
	if err := s.store.SaveMessage(ctx, msg, storedAttachments); err != nil {
		return err
	}
	s.store.IncrWebhookCount(ctx, false)
	return nil
}

func (s *Service) storeAttachments(ctx context.Context, companyID string, msgID string, attachments []rawAttachment) ([]domain.Attachment, error) {
	filtered := filterAttachments(attachments)
	if len(filtered) == 0 {
		return nil, nil
	}

	var stored []domain.Attachment
	for _, att := range filtered {
		data, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			continue
		}
		if int64(len(data)) > maxAttachmentBytes {
			return nil, domain.ErrAttachmentTooLarge
		}
		attID := generateID()
		objectKey := fmt.Sprintf("inbound/%s/%s/%s", companyID, msgID, attID)
		if err := s.objects.Put(ctx, objectKey, bytes.NewReader(data), int64(len(data)), att.ContentType); err != nil {
			return nil, fmt.Errorf("store inbound attachment object: %w", err)
		}
		stored = append(stored, domain.Attachment{
			ID:          attID,
			MessageID:   msgID,
			Name:        att.FileName,
			ContentType: contentType(att.FileName, att.ContentType),
			Size:        int64(len(data)),
			ObjectKey:   objectKey,
		})
	}
	if len(stored) == 0 {
		return nil, nil
	}
	return stored, nil
}

func (s *Service) saveMinimalMessage(ctx context.Context, providerID, subject, bodyText, bodyHTML string, mail rawWebhook, status domain.InboundStatus, errorCode string) error {
	msg := domain.MessageSummary{
		ID:                generateID(),
		ProviderMessageID: providerID,
		Subject:           subject,
		EnvelopeFrom:      mail.Envelope.From,
		EnvelopeTo:        mail.Envelope.To,
		ReceivedAt:        time.Now().UTC(),
		Status:            status,
		ErrorCode:         errorCode,
		BodyText:          bodyText,
		BodyHTML:          bodyHTML,
	}

	if err := s.store.SaveMessage(ctx, msg, nil); err != nil {
		return err
	}
	s.store.IncrWebhookCount(ctx, true)
	return nil
}

func resolveMessageBody(mail rawWebhook) (string, string) {
	plain, html := mail.Plain, mail.HTML
	var fields map[string]any
	if err := json.Unmarshal(mail.Headers, &fields); err == nil {
		if plain == "" {
			plain = headerValue(fields, "plain", "text", "text_plain")
		}
		if html == "" {
			html = headerValue(fields, "html", "text_html")
		}
	}
	return plain, html
}

func headerValue(fields map[string]any, names ...string) string {
	for key, value := range fields {
		for _, name := range names {
			if strings.EqualFold(key, name) {
				if text, ok := value.(string); ok {
					return text
				}
			}
		}
	}
	return ""
}

func messageIDFromHeaders(headers json.RawMessage) string {
	var h map[string]any
	if err := json.Unmarshal(headers, &h); err != nil {
		return ""
	}
	for k, v := range h {
		if !strings.EqualFold(k, "message_id") && !strings.EqualFold(k, "message-id") {
			continue
		}
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
		if values, ok := v.([]any); ok && len(values) > 0 {
			if s, ok := values[0].(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func subjectFromHeaders(headers json.RawMessage) string {
	var h map[string]any
	if err := json.Unmarshal(headers, &h); err != nil {
		return ""
	}
	for k, v := range h {
		if !strings.EqualFold(k, "subject") {
			continue
		}
		switch t := v.(type) {
		case string:
			return t
		case []any:
			if len(t) > 0 {
				if s, ok := t[0].(string); ok {
					return s
				}
			}
		}
	}
	return ""
}

func filterAttachments(raw []rawAttachment) []rawAttachment {
	var out []rawAttachment
	for _, a := range raw {
		name := strings.TrimSpace(a.FileName)
		if name == "" || a.Content == "" {
			continue
		}
		out = append(out, a)
	}
	return out
}

func contentType(name, declared string) string {
	if declared != "" {
		return declared
	}
	if i := strings.LastIndex(name, "."); i >= 0 {
		ct := mime.TypeByExtension(strings.ToLower(name[i:]))
		if ct != "" {
			if idx := strings.Index(ct, ";"); idx >= 0 {
				ct = strings.TrimSpace(ct[:idx])
			}
			return ct
		}
	}
	return "application/octet-stream"
}

func generateID() string {
	return uuid.New().String()
}

func generateFallbackID() string {
	return "msg-" + uuid.New().String()
}

func count32(n int) int32 {
	if n > math.MaxInt32 {
		return math.MaxInt32
	}
	if n < math.MinInt32 {
		return math.MinInt32
	}
	return int32(n)
}
