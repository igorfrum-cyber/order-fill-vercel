package inbound

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"strings"
	"time"

	"order-fill/backend/services/inbound-service/internal/domain"
)

type Store interface {
	GetSettings(ctx context.Context) (domain.Settings, error)
	UpsertSettings(ctx context.Context, enabled bool) (domain.Settings, error)
	IncrWebhookCount(ctx context.Context, isError bool)
	GetCompanyInbound(ctx context.Context, companyID string) (domain.CompanyInbound, error)
	GetCompanyInboundByAddress(ctx context.Context, address string) (domain.CompanyInbound, error)
	UpsertCompanyInbound(ctx context.Context, companyID, address string, allowedFrom []string, enabled bool) (domain.CompanyInbound, error)
	MessageExists(ctx context.Context, providerMessageID string) (bool, error)
	SaveMessage(ctx context.Context, msg domain.MessageSummary, attachments []domain.Attachment) error
	ListMessages(ctx context.Context, companyID string, limit int) ([]domain.MessageSummary, error)
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

func (s *Service) UpdateSettings(ctx context.Context, enabled bool, actorUserID string) (domain.Settings, error) {
	return s.store.UpsertSettings(ctx, enabled)
}

func (s *Service) GetCompanyInbound(ctx context.Context, companyID string) (domain.CompanyInbound, error) {
	return s.store.GetCompanyInbound(ctx, companyID)
}

func (s *Service) UpdateCompanyInbound(ctx context.Context, companyID, address string, allowedFrom []string, enabled bool) (domain.CompanyInbound, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return domain.CompanyInbound{}, domain.ErrInvalid
	}
	return s.store.UpsertCompanyInbound(ctx, companyID, address, allowedFrom, enabled)
}

func (s *Service) ListMessages(ctx context.Context, companyID string, limit int) ([]domain.MessageSummary, error) {
	return s.store.ListMessages(ctx, companyID, limit)
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
	MessageID          string          `json:"message_id"`
	Attachments        []rawAttachment `json:"attachments"`
	AttachmentQuantity int             `json:"attachment_quantity"`
}

const maxAttachmentBytes = 20 << 20

func (s *Service) IngestWebhook(ctx context.Context, rawPayload []byte) error {
	if len(rawPayload) > 32<<20 {
		return domain.ErrPayloadTooLarge
	}

	var mail rawWebhook
	if err := json.Unmarshal(rawPayload, &mail); err != nil {
		return fmt.Errorf("parse inbound webhook: %w", err)
	}

	if strings.TrimSpace(mail.MessageID) == "" {
		if h := subjectFromHeaders(mail.Headers); h != "" {
			mail.MessageID = generateFallbackID()
		} else {
			mail.MessageID = generateFallbackID()
		}
	}
	providerID := strings.TrimSpace(mail.MessageID)

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

	inbound, err := s.store.GetCompanyInboundByAddress(ctx, mail.Envelope.To)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.saveMinimalMessage(ctx, providerID, mail, domain.StatusErrorUnknownAddress, "unknown_address")
			return nil
		}
		return err
	}

	if len(inbound.AllowedFrom) > 0 && !fromMatchesWhitelist(mail.Envelope.From, inbound.AllowedFrom) {
		s.saveMinimalMessage(ctx, providerID, mail, domain.StatusErrorMismatchFrom, "mismatch_from")
		return nil
	}

	attachments := filterAttachments(mail.Attachments)
	if len(attachments) == 0 {
		s.saveMinimalMessage(ctx, providerID, mail, domain.StatusErrorNoAttachments, "no_attachments")
		return nil
	}

	msgID := generateID()
	var storedAttachments []domain.Attachment
	var totalBytes int64

	for _, att := range attachments {
		data, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			continue
		}
		if int64(len(data)) > maxAttachmentBytes {
			s.saveMinimalMessage(ctx, providerID, mail, domain.StatusErrorTooLarge, "too_large")
			return nil
		}
		attID := generateID()
		objectKey := fmt.Sprintf("inbound/%s/%s/%s", inbound.CompanyID, msgID, attID)
		if err := s.objects.Put(ctx, objectKey, bytes.NewReader(data), int64(len(data)), att.ContentType); err != nil {
			return fmt.Errorf("store inbound attachment object: %w", err)
		}
		storedAttachments = append(storedAttachments, domain.Attachment{
			ID:          attID,
			MessageID:   msgID,
			Name:        att.FileName,
			ContentType: contentType(att.FileName, att.ContentType),
			Size:        int64(len(data)),
			ObjectKey:   objectKey,
		})
		totalBytes += int64(len(data))
	}

	msg := domain.MessageSummary{
		ID:                msgID,
		ProviderMessageID: providerID,
		EnvelopeFrom:      mail.Envelope.From,
		EnvelopeTo:        mail.Envelope.To,
		ReceivedAt:        time.Now().UTC(),
		CompanyID:         inbound.CompanyID,
		Status:            domain.StatusReceived,
		AttachmentCount:   count32(len(storedAttachments)),
		TotalBytes:        totalBytes,
	}
	if err := s.store.SaveMessage(ctx, msg, storedAttachments); err != nil {
		return err
	}
	s.store.IncrWebhookCount(ctx, false)
	return nil
}

func (s *Service) saveMinimalMessage(ctx context.Context, providerID string, mail rawWebhook, status domain.InboundStatus, errorCode string) {
	msg := domain.MessageSummary{
		ID:                generateID(),
		ProviderMessageID: providerID,
		EnvelopeFrom:      mail.Envelope.From,
		EnvelopeTo:        mail.Envelope.To,
		ReceivedAt:        time.Now().UTC(),
		Status:            status,
		ErrorCode:         errorCode,
	}
	_ = s.store.SaveMessage(ctx, msg, nil)
	s.store.IncrWebhookCount(ctx, true)
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

func fromMatchesWhitelist(from string, allowed []string) bool {
	from = strings.TrimSpace(strings.ToLower(from))
	for _, a := range allowed {
		a = strings.TrimSpace(strings.ToLower(a))
		if a == "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(from), []byte(a)) == 1 {
			return true
		}
		if strings.HasSuffix(from, "@"+a) || strings.HasSuffix(from, "."+a) {
			return true
		}
	}
	return false
}

func generateID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func generateFallbackID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "msg-" + hex.EncodeToString(b[:])
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
