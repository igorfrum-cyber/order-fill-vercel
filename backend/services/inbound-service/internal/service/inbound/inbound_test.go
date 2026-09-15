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
	companyByAddress domain.CompanyInbound
	addressErr       error
	exists           map[string]bool
	saved            []domain.MessageSummary
	savedAttachments map[string][]domain.Attachment
	errors           int
	webhooks         int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		settings:         domain.Settings{Enabled: true},
		companyByAddress: domain.CompanyInbound{CompanyID: "c1", ReceiveAddress: "sales@orderfill.local", AllowedFrom: []string{"1c@company.ru"}},
		exists:           map[string]bool{},
		savedAttachments: map[string][]domain.Attachment{},
	}
}

func (f *fakeStore) GetSettings(context.Context) (domain.Settings, error) { return f.settings, nil }
func (f *fakeStore) UpsertSettings(_ context.Context, enabled bool) (domain.Settings, error) {
	f.settings.Enabled = enabled
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
	out := f.companyByAddress
	out.CompanyID = companyID
	return out, nil
}
func (f *fakeStore) GetCompanyInboundByAddress(_ context.Context, address string) (domain.CompanyInbound, error) {
	if f.addressErr != nil {
		return domain.CompanyInbound{}, f.addressErr
	}
	if strings.EqualFold(address, f.companyByAddress.ReceiveAddress) {
		return f.companyByAddress, nil
	}
	return domain.CompanyInbound{}, domain.ErrNotFound
}
func (f *fakeStore) UpsertCompanyInbound(_ context.Context, companyID, address string, allowed []string, enabled bool) (domain.CompanyInbound, error) {
	f.companyByAddress = domain.CompanyInbound{CompanyID: companyID, ReceiveAddress: address, AllowedFrom: allowed, Enabled: enabled}
	return f.companyByAddress, nil
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
func (f *fakeStore) GetMessageAttachments(context.Context, string) ([]domain.Attachment, error) {
	return nil, nil
}
func (f *fakeStore) GetAttachment(context.Context, string) (domain.Attachment, error) {
	return domain.Attachment{}, nil
}
func (f *fakeStore) GetMessageCompany(context.Context, string) (string, error) {
	return f.companyByAddress.CompanyID, nil
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
	return `{"message_id":"` + messageID + `","envelope":{"from":"1c@company.ru","to":"sales@orderfill.local"},"attachments":` + encodeRaw(attachments) + `}`
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

func TestIngestUnknownAddressSavesMinimal(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "order-fill-inbound")
	body := webhookEnvelope("any@x.io", "none@other.io", "msg-1")
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved=%d", len(store.saved))
	}
	if store.saved[0].Status != domain.StatusErrorUnknownAddress {
		t.Fatalf("status=%s", store.saved[0].Status)
	}
	if store.saved[0].ErrorCode != "unknown_address" {
		t.Fatalf("error_code=%s", store.saved[0].ErrorCode)
	}
	if store.errors != 1 {
		t.Fatalf("errors=%d", store.errors)
	}
}

func TestIngestDedupesByProviderMessageID(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookEnvelope("1c@company.ru", "sales@orderfill.local", "dup-1")
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
	body := webhookEnvelope("1c@company.ru", "sales@orderfill.local", "msg-2")
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err == nil {
		t.Fatal("expected error when disabled")
	}
	if store.errors != 1 {
		t.Fatalf("errors=%d", store.errors)
	}
}

func TestIngestMismatchFrom(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookEnvelope("evil@x.io", "sales@orderfill.local", "msg-3")
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.saved) != 1 || store.saved[0].Status != domain.StatusErrorMismatchFrom {
		t.Fatalf("status=%v", store.saved[0].Status)
	}
}

func TestIngestNoAttachments(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookEnvelope("1c@company.ru", "sales@orderfill.local", "msg-4")
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
	body := webhookWithAttachments("msg-5", []rawAttachment{
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
	body := webhookWithAttachments("msg-6", []rawAttachment{
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
	body := webhookWithAttachments("msg-7", []rawAttachment{
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

func TestFromMatchesWhitelist(t *testing.T) {
	t.Parallel()
	allowed := []string{"1c@company.ru", "support"}
	if !fromMatchesWhitelist("1c@company.ru", allowed) {
		t.Fatal("exact match must pass")
	}
	if !fromMatchesWhitelist("1C@COMPANY.RU", allowed) {
		t.Fatal("case-insensitive match must pass")
	}
	if fromMatchesWhitelist("evil@x.io", allowed) {
		t.Fatal("unlisted sender must be rejected")
	}
	if fromMatchesWhitelist("sales@company.ru", allowed) {
		t.Fatal("suffix against bare word whitelist must fail")
	}
	if fromMatchesWhitelist("", allowed) {
		t.Fatal("empty from must be rejected")
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

func TestIngestUnknownAddressOnStoreError(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	store.addressErr = errors.New("db down")
	svc := New(store, newFakeObjectStore(), "bucket")
	body := webhookEnvelope("any@x.io", "unknown@x.io", "msg-8")
	if err := svc.IngestWebhook(t.Context(), []byte(body)); err == nil {
		t.Fatal("expected error when store fails")
	}
}
