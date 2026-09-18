package inbound

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"testing"

	"order-fill/backend/services/inbound-service/internal/domain"
)

type fakeStore struct {
	settings         domain.Settings
	companyBySender  domain.CompanyInbound
	senderErr        error
	exists           map[string]bool
	saved            []domain.MessageSummary
	savedAttachments map[string][]domain.Attachment
	errors           int
	webhooks         int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		settings:         domain.Settings{Enabled: true, ReceiveAddress: "7e1432246b724f3bcd6c@cloudmailin.net"},
		companyBySender:  domain.CompanyInbound{CompanyID: "c1", ReceiveAddress: "7e1432246b724f3bcd6c@cloudmailin.net", SenderEmail: "1c@company.ru", Enabled: true},
		exists:           map[string]bool{},
		savedAttachments: map[string][]domain.Attachment{},
	}
}

func (f *fakeStore) GetSettings(context.Context) (domain.Settings, error) { return f.settings, nil }
func (f *fakeStore) UpsertSettings(_ context.Context, enabled bool, receiveAddress string) (domain.Settings, error) {
	f.settings.Enabled = enabled
	f.settings.ReceiveAddress = receiveAddress
	return f.settings, nil
}
func (f *fakeStore) IncrWebhookCount(_ context.Context, isError bool) {
	if isError {
		f.errors++
	} else {
		f.webhooks++
	}
}
func (f *fakeStore) GetCompanyInbound(_ context.Context, companyID string) (domain.CompanyInbound, error) {
	out := f.companyBySender
	out.CompanyID = companyID
	return out, nil
}
func (f *fakeStore) GetCompanyByAllowedSender(_ context.Context, senderEmail string) (domain.CompanyInbound, error) {
	if f.senderErr != nil {
		return domain.CompanyInbound{}, f.senderErr
	}
	if strings.EqualFold(senderEmail, f.companyBySender.SenderEmail) {
		return f.companyBySender, nil
	}
	return domain.CompanyInbound{}, domain.ErrNotFound
}
func (f *fakeStore) UpsertCompanyInbound(_ context.Context, companyID, receiveAddress, senderEmail string, enabled bool) (domain.CompanyInbound, error) {
	f.companyBySender = domain.CompanyInbound{CompanyID: companyID, ReceiveAddress: receiveAddress, SenderEmail: senderEmail, Enabled: enabled}
	return f.companyBySender, nil
}
func (f *fakeStore) MessageExists(_ context.Context, providerMessageID string) (bool, error) {
	_, ok := f.exists[providerMessageID]
	return ok, nil
}
func (f *fakeStore) SaveMessage(_ context.Context, msg domain.MessageSummary, attachments []domain.Attachment) error {
	f.saved = append(f.saved, msg)
	f.exists[msg.ProviderMessageID] = true
	f.savedAttachments[msg.ID] = attachments
	return nil
}
func (f *fakeStore) ListMessages(context.Context, string, int) ([]domain.MessageSummary, error) {
	return f.saved, nil
}
func (f *fakeStore) GetMessage(_ context.Context, messageID string) (domain.MessageSummary, error) {
	for _, msg := range f.saved {
		if msg.ID == messageID {
			return msg, nil
		}
	}
	return domain.MessageSummary{}, domain.ErrNotFound
}
func (f *fakeStore) GetMessageAttachments(context.Context, string) ([]domain.Attachment, error) {
	return nil, nil
}
func (f *fakeStore) GetAttachment(context.Context, string) (domain.Attachment, error) {
	return domain.Attachment{}, nil
}
func (f *fakeStore) GetMessageCompany(context.Context, string) (string, error) {
	return f.companyBySender.CompanyID, nil
}
func (f *fakeStore) ListDeliveries(context.Context, int) ([]domain.MessageSummary, error) {
	return f.saved, nil
}

type fakeObjectStore struct {
	keys map[string][]byte
}

func newFakeObjectStore() *fakeObjectStore { return &fakeObjectStore{keys: map[string][]byte{}} }
func (f *fakeObjectStore) Put(_ context.Context, key string, body io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	f.keys[key] = data
	return nil
}
func (f *fakeObjectStore) Get(_ context.Context, key string) ([]byte, string, error) {
	data, ok := f.keys[key]
	if !ok {
		return nil, "", domain.ErrNotFound
	}
	return data, "application/octet-stream", nil
}

func webhookEnvelope(from, to, messageID string) string {
	return `{"message_id":"` + messageID + `","envelope":{"from":"` + from + `","to":"` + to + `"},"attachments":[]}`
}

func webhookWithAttachments(messageID string, attachments []rawAttachment) string {
	return `{"message_id":"` + messageID + `","envelope":{"from":"1c@company.ru","to":"7e1432246b724f3bcd6c@cloudmailin.net"},"attachments":` + encodeRaw(attachments) + `}`
}

func webhookWithSubject(messageID, subject string) string {
	return `{"message_id":"` + messageID + `","envelope":{"from":"1c@company.ru","to":"7e1432246b724f3bcd6c@cloudmailin.net"},"subject":"` + subject + `","attachments":[]}`
}

func encodeRaw(in []rawAttachment) string {
	var b strings.Builder
	b.WriteString("[")
	for i, a := range in {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"file_name":`)
		b.WriteString(strconvQuote(a.FileName))
		b.WriteString(`,"content_type":`)
		b.WriteString(strconvQuote(a.ContentType))
		b.WriteString(`,"content":`)
		b.WriteString(strconvQuote(a.Content))
		b.WriteString(`}`)
	}
	b.WriteString("]")
	return b.String()
}

func strconvQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func base64Of(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func TestIngestSenderMatch(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookWithAttachments("msg-sender-1", []rawAttachment{
		{FileName: "report.xlsx", ContentType: "application/vnd.ms-excel", Content: base64Of("xlsx-bytes")},
	})
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d", len(store.saved))
	}
	msg := store.saved[0]
	if msg.Status != domain.StatusReceived {
		t.Fatalf("status=%s, want received", msg.Status)
	}
	if msg.CompanyID != "c1" {
		t.Fatalf("company_id=%s, want c1", msg.CompanyID)
	}
}

func TestIngestInvalidJSONIsInvalidPayload(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	err := svc.IngestWebhook(t.Context(), []byte("{"))
	if !errors.Is(err, domain.ErrInvalidPayload) {
		t.Fatalf("got %v, want ErrInvalidPayload", err)
	}
	if len(store.saved) != 0 {
		t.Fatalf("poison payload must not be stored, saved=%d", len(store.saved))
	}
}

func TestIngestPayloadTooLarge(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	err := svc.IngestWebhook(t.Context(), make([]byte, 32<<20+1))
	if !errors.Is(err, domain.ErrPayloadTooLarge) {
		t.Fatalf("got %v, want ErrPayloadTooLarge", err)
	}
	if len(store.saved) != 0 {
		t.Fatalf("oversized payload must not be stored, saved=%d", len(store.saved))
	}
}

func TestIngestUnknownSenderSavesMinimal(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookEnvelope("unknown@x.io", "7e1432246b724f3bcd6c@cloudmailin.net", "msg-sender-2")
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d", len(store.saved))
	}
	if store.saved[0].Status != domain.StatusErrorUnknownAddress {
		t.Fatalf("status=%s, want error:unknown_address", store.saved[0].Status)
	}
	if store.saved[0].ErrorCode != "unknown_address" {
		t.Fatalf("error_code=%s", store.saved[0].ErrorCode)
	}
	if store.errors != 1 {
		t.Fatalf("errors=%d", store.errors)
	}
}

func TestIngestSubjectSaved(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookWithSubject("msg-subject-1", "Заказ №12345")
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d", len(store.saved))
	}
	if store.saved[0].Subject != "Заказ №12345" {
		t.Fatalf("subject=%q, want %q", store.saved[0].Subject, "Заказ №12345")
	}
}

func TestIngestStoresFormattedMessageBody(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := `{"message_id":"msg-body-1","envelope":{"from":"1c@company.ru","to":"7e1432246b724f3bcd6c@cloudmailin.net"},"subject":"Заказ","plain":"Текст письма","html":"<p><strong>Текст</strong> письма</p>","attachments":[{"file_name":"report.xlsx","content":"` + base64Of("data") + `"}]}`
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d", len(store.saved))
	}
	if got := store.saved[0].BodyText; got != "Текст письма" {
		t.Fatalf("body_text=%q", got)
	}
	if got := store.saved[0].BodyHTML; got != "<p><strong>Текст</strong> письма</p>" {
		t.Fatalf("body_html=%q", got)
	}
}

func TestIngestSubjectFromHeaders(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := `{"message_id":"msg-subject-2","envelope":{"from":"1c@company.ru","to":"7e1432246b724f3bcd6c@cloudmailin.net"},"headers":{"Subject":["Тема из заголовков"]},"attachments":[]}`
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d", len(store.saved))
	}
	if store.saved[0].Subject != "Тема из заголовков" {
		t.Fatalf("subject=%q, want %q", store.saved[0].Subject, "Тема из заголовков")
	}
}

func TestIngestSubjectSavedOnMinimalMessage(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookWithSubject("msg-subject-3", "Ошибка доставки")
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d", len(store.saved))
	}
	if store.saved[0].Subject != "Ошибка доставки" {
		t.Fatalf("subject=%q on minimal message", store.saved[0].Subject)
	}
}

func TestIngestUUIDv7Format(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookWithAttachments("msg-uuid-1", []rawAttachment{
		{FileName: "data.xlsx", Content: base64Of("bytes")},
	})
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d", len(store.saved))
	}
	id := store.saved[0].ID
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Fatalf("id=%q, want UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)", id)
	}
}

func TestIngestDedupesByProviderMessageID(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookWithAttachments("dup-sender-1", []rawAttachment{
		{FileName: "a.xlsx", Content: base64Of("data")},
	})
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	firstCount := len(store.saved)
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if len(store.saved) != firstCount {
		t.Fatalf("duplicate was processed: %d", len(store.saved))
	}
}

func TestIngestDisabledRejects(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	store.settings.Enabled = false
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookEnvelope("1c@company.ru", "7e1432246b724f3bcd6c@cloudmailin.net", "msg-disabled-1")
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err == nil {
		t.Fatal("expected error when disabled")
	}
	if store.errors != 1 {
		t.Fatalf("errors=%d", store.errors)
	}
}

func TestIngestNoAttachments(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookEnvelope("1c@company.ru", "7e1432246b724f3bcd6c@cloudmailin.net", "msg-noatt-1")
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 || store.saved[0].Status != domain.StatusErrorNoAttachments {
		t.Fatalf("status=%v", store.saved[0].Status)
	}
}

func TestIngestStoresAttachmentAndObject(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	objects := newFakeObjectStore()
	svc := New(store, objects, "bucket")
	body := webhookWithAttachments("msg-store-1", []rawAttachment{
		{FileName: "report.xlsx", ContentType: "application/vnd.ms-excel", Content: base64Of("xlsx-bytes")},
	})
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d", len(store.saved))
	}
	msg := store.saved[0]
	if msg.Status != domain.StatusReceived || msg.AttachmentCount != 1 || msg.TotalBytes != int64(len("xlsx-bytes")) {
		t.Fatalf("msg=%+v", msg)
	}
	if store.webhooks != 1 {
		t.Fatalf("webhooks=%d", store.webhooks)
	}
	attachments := store.savedAttachments[msg.ID]
	if len(attachments) != 1 {
		t.Fatalf("attachments=%d", len(attachments))
	}
	if _, ok := objects.keys[attachments[0].ObjectKey]; !ok {
		t.Fatalf("object not stored, keys=%v", objects.keys)
	}
}

func TestIngestSkipsInvalidBase64(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookWithAttachments("msg-b64-1", []rawAttachment{
		{FileName: "bad.bin", Content: "not-base64!!"},
		{FileName: "ok.txt", Content: base64Of("hello")},
	})
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 || store.saved[0].AttachmentCount != 1 {
		t.Fatalf("saved=%+v", store.saved[0])
	}
}

func TestIngestTooLargeAttachmentRejected(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookWithAttachments("msg-large-1", []rawAttachment{
		{FileName: "big.bin", Content: base64Of(strings.Repeat("a", 21<<20))},
	})
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 || store.saved[0].Status != domain.StatusErrorTooLarge {
		t.Fatalf("saved=%+v", store.saved[0])
	}
}

func TestFilterAttachmentsIgnoresURLOnly(t *testing.T) {
	t.Parallel()
	out := filterAttachments([]rawAttachment{
		{FileName: "a.xlsx", Content: "AA=="},
		{FileName: "u.xlsx", Content: "", URL: "https://attachments.example.com/1"},
	})
	if len(out) != 1 || out[0].FileName != "a.xlsx" {
		t.Fatalf("out=%+v", out)
	}
}

func TestContentTypeFallback(t *testing.T) {
	t.Parallel()
	if got := contentType("report.xlsx", ""); got == "application/octet-stream" || got == "" {
		t.Fatalf("xlsx content type not derived: %q", got)
	}
	if got := contentType("x.bin", "custom/type"); got != "custom/type" {
		t.Fatalf("declared type must be used: %q", got)
	}
}

func TestIngestSenderMatchOnStoreError(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	store.senderErr = errors.New("db down")
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookEnvelope("1c@company.ru", "7e1432246b724f3bcd6c@cloudmailin.net", "msg-dberr-1")
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err == nil {
		t.Fatal("expected error when store fails")
	}
}

func TestUpdateCompanyInboundEmptySenderRejected(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	_, err := svc.UpdateCompanyInbound(t.Context(), "c1", "addr@test.com", "", true)
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for empty senderEmail, got: %v", err)
	}
}

func TestUpdateSettingsPassesReceiveAddress(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	got, err := svc.UpdateSettings(t.Context(), true, "new@cloudmailin.net", "admin-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ReceiveAddress != "new@cloudmailin.net" {
		t.Fatalf("receive_address=%q, want new@cloudmailin.net", got.ReceiveAddress)
	}
	if store.settings.ReceiveAddress != "new@cloudmailin.net" {
		t.Fatalf("store not updated: receive_address=%q", store.settings.ReceiveAddress)
	}
}

func TestIngestSubjectHeadersBeatTopLevel(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := `{"message_id":"msg-hdr-1","envelope":{"from":"1c@company.ru","to":"7e1432246b724f3bcd6c@cloudmailin.net"},"subject":"top-level","headers":{"Subject":["from-header"]},"attachments":[{"file_name":"f.xlsx","content":"` + base64Of("data") + `"}]}`
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d", len(store.saved))
	}
	if store.saved[0].Subject != "from-header" {
		t.Fatalf("subject=%q, want headers Subject to win over top-level", store.saved[0].Subject)
	}
}

func TestIngestUsesCloudMailinHeaderMessageIDForDeduplication(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := `{"envelope":{"from":"1c@company.ru","to":"7e1432246b724f3bcd6c@cloudmailin.net"},"headers":{"message_id":"<cloudmailin-123@example.com>"},"attachments":[{"file_name":"f.xlsx","content":"` + base64Of("data") + `"}]}`
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("duplicate ingest: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d, want one message for duplicate provider ID", len(store.saved))
	}
	if store.saved[0].ProviderMessageID != "<cloudmailin-123@example.com>" {
		t.Fatalf("provider_message_id=%q", store.saved[0].ProviderMessageID)
	}
}

func TestIngestSubjectOnTooLargeError(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := `{"message_id":"msg-lg-subj","envelope":{"from":"1c@company.ru","to":"7e1432246b724f3bcd6c@cloudmailin.net"},"subject":"Огромный файл","attachments":[{"file_name":"big.bin","content":"` + base64Of(strings.Repeat("x", 21<<20)) + `"}]}`
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d", len(store.saved))
	}
	if store.saved[0].Subject != "Огромный файл" {
		t.Fatalf("subject=%q on too_large message", store.saved[0].Subject)
	}
	if store.saved[0].Status != domain.StatusErrorTooLarge {
		t.Fatalf("status=%s, want too_large", store.saved[0].Status)
	}
}
