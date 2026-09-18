package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"order-fill/backend/pkg/grpcutil"
	auditv1 "order-fill/backend/proto/gen/go/orderfill/audit/v1"
	inboundv1 "order-fill/backend/proto/gen/go/orderfill/inbound/v1"
	"order-fill/backend/services/gateway-service/internal/clients"
	"order-fill/backend/services/gateway-service/internal/config"
)

type recordingAudit struct {
	auditv1.AuditServiceClient
	mu       sync.Mutex
	requests []*auditv1.RecordRequest
	err      error
}

func (c *recordingAudit) Record(_ context.Context, req *auditv1.RecordRequest, _ ...grpc.CallOption) (*auditv1.RecordResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = append(c.requests, req)
	if c.err != nil {
		return nil, c.err
	}
	return &auditv1.RecordResponse{Id: "evt-1"}, nil
}

func (c *recordingAudit) lastType() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.requests) == 0 {
		return ""
	}
	return c.requests[len(c.requests)-1].GetType()
}

func (c *recordingAudit) lastCompany() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.requests) == 0 || c.requests[len(c.requests)-1].GetMeta() == nil {
		return ""
	}
	return c.requests[len(c.requests)-1].GetMeta().GetCompanyId()
}

func (c *recordingAudit) called() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.requests) > 0
}

type ingestResultClient struct {
	inboundv1.InboundServiceClient
	status    inboundv1.InboundMessageStatus
	companyID string
	err       error
}

func (c ingestResultClient) IngestWebhook(context.Context, *inboundv1.IngestWebhookRequest, ...grpc.CallOption) (*inboundv1.IngestWebhookResponse, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &inboundv1.IngestWebhookResponse{CompanyId: c.companyID, Status: c.status}, nil
}

type inboundClient struct{ inboundv1.InboundServiceClient }

type inboundSettingsCapture struct {
	inboundv1.InboundServiceClient
	gotCtx context.Context
}

func (c *inboundSettingsCapture) GetSettings(ctx context.Context, _ *inboundv1.GetSettingsRequest, _ ...grpc.CallOption) (*inboundv1.GetSettingsResponse, error) {
	c.gotCtx = ctx
	return &inboundv1.GetSettingsResponse{Settings: &inboundv1.InboundSettings{Enabled: true, ReceiveAddress: "7e1432246b724f3bcd6c@cloudmailin.net", WebhookCount: 3, ErrorCount: 1}}, nil
}

func (inboundClient) IngestWebhook(context.Context, *inboundv1.IngestWebhookRequest, ...grpc.CallOption) (*inboundv1.IngestWebhookResponse, error) {
	return &inboundv1.IngestWebhookResponse{}, nil
}

func (inboundClient) GetSettings(context.Context, *inboundv1.GetSettingsRequest, ...grpc.CallOption) (*inboundv1.GetSettingsResponse, error) {
	return &inboundv1.GetSettingsResponse{Settings: &inboundv1.InboundSettings{Enabled: true, ReceiveAddress: "7e1432246b724f3bcd6c@cloudmailin.net", WebhookCount: 3, ErrorCount: 1}}, nil
}

func (inboundClient) UpdateSettings(_ context.Context, req *inboundv1.UpdateSettingsRequest, _ ...grpc.CallOption) (*inboundv1.UpdateSettingsResponse, error) {
	return &inboundv1.UpdateSettingsResponse{Settings: &inboundv1.InboundSettings{Enabled: req.GetEnabled(), ReceiveAddress: req.GetReceiveAddress()}}, nil
}

func (inboundClient) GetCompanyInbound(context.Context, *inboundv1.GetCompanyInboundRequest, ...grpc.CallOption) (*inboundv1.GetCompanyInboundResponse, error) {
	return &inboundv1.GetCompanyInboundResponse{CompanyInbound: &inboundv1.CompanyInbound{
		CompanyId: "c1", ReceiveAddress: "7e1432246b724f3bcd6c@cloudmailin.net", SenderEmail: "1c@company.ru", Enabled: true,
	}}, nil
}

func (inboundClient) UpdateCompanyInbound(_ context.Context, req *inboundv1.UpdateCompanyInboundRequest, _ ...grpc.CallOption) (*inboundv1.UpdateCompanyInboundResponse, error) {
	return &inboundv1.UpdateCompanyInboundResponse{CompanyInbound: &inboundv1.CompanyInbound{
		CompanyId: req.GetCompanyId(), ReceiveAddress: req.GetReceiveAddress(), SenderEmail: req.GetSenderEmail(), Enabled: req.GetEnabled(),
	}}, nil
}

func (inboundClient) ListMessages(context.Context, *inboundv1.ListMessagesRequest, ...grpc.CallOption) (*inboundv1.ListMessagesResponse, error) {
	return &inboundv1.ListMessagesResponse{Messages: []*inboundv1.InboundMessageSummary{
		{Id: "m1", ProviderMessageId: "abc", Subject: "Заказ №123", EnvelopeFrom: "1c@company.ru", EnvelopeTo: "7e1432246b724f3bcd6c@cloudmailin.net",
			ReceivedAt: "2026-09-15T10:00:00Z", CompanyId: "c1", Status: inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_RECEIVED,
			AttachmentCount: 1, TotalBytes: 1024,
			Attachments: []*inboundv1.InboundAttachment{
				{Id: "a1", Name: "report.xlsx", ContentType: "application/vnd.ms-excel", Size: 1024},
			},
		},
	}}, nil
}

func (inboundClient) GetMessage(context.Context, *inboundv1.GetMessageRequest, ...grpc.CallOption) (*inboundv1.GetMessageResponse, error) {
	return &inboundv1.GetMessageResponse{Message: &inboundv1.InboundMessageSummary{
		Id: "m1", Subject: "Заказ №123", EnvelopeFrom: "1c@company.ru", ReceivedAt: "2026-09-15T10:00:00Z",
		CompanyId: "c1", BodyText: "plain body", BodyHtml: "<p>formatted body</p>",
	}}, nil
}

func (inboundClient) GetMessageFile(context.Context, *inboundv1.GetMessageFileRequest, ...grpc.CallOption) (*inboundv1.GetMessageFileResponse, error) {
	return &inboundv1.GetMessageFileResponse{Attachment: &inboundv1.InboundAttachment{
		Id: "a1", Name: "report.xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Size: 1024,
	}, Body: []byte("xlsx"), CompanyId: "c1"}, nil
}

func (inboundClient) ListDeliveries(context.Context, *inboundv1.ListDeliveriesRequest, ...grpc.CallOption) (*inboundv1.ListDeliveriesResponse, error) {
	return &inboundv1.ListDeliveriesResponse{Deliveries: []*inboundv1.InboundMessageSummary{
		{Id: "d1", ProviderMessageId: "abc", Subject: "Заказ №456", EnvelopeFrom: "1c@company.ru", EnvelopeTo: "7e1432246b724f3bcd6c@cloudmailin.net",
			ReceivedAt: "2026-09-15T10:00:00Z", CompanyId: "c1", Status: inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_RECEIVED,
			AttachmentCount: 1, TotalBytes: 1024},
	}}, nil
}

func TestInboundWebhookWrongTokenRejected(t *testing.T) {
	t.Parallel()
	api := &API{InboundWebhookToken: "secret-token"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/webhook", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer wrong")
	if api.inboundWebhookAuthorized(req) {
		t.Fatal("wrong token must be rejected")
	}
}

func TestInboundWebhookAuthorizedWithToken(t *testing.T) {
	t.Parallel()
	api := &API{InboundWebhookToken: "secret-token"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/webhook", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer secret-token")
	if !api.inboundWebhookAuthorized(req) {
		t.Fatal("correct token must pass")
	}
}

func TestInboundWebhookGateRejectsMissingToken(t *testing.T) {
	t.Parallel()
	h := New(configForTest(), clients.Clients{Inbound: inboundClient{}})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/inbound/webhook", strings.NewReader(`{}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestInboundWebhookGateAcceptsToken(t *testing.T) {
	t.Parallel()
	h := New(configForTest(), clients.Clients{Inbound: inboundClient{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/webhook", strings.NewReader(`{"message_id":"abc"}`))
	req.Header.Set("Authorization", "Bearer webhook-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInboundWebhookRateLimitReturns429(t *testing.T) {
	t.Parallel()
	cfg := configForTest()
	cfg.InboundWebhookRPS = 1
	cfg.InboundWebhookBurst = 1
	h := New(cfg, clients.Clients{Inbound: inboundClient{}})
	post := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/webhook", strings.NewReader(`{"message_id":"abc"}`))
		req.Header.Set("Authorization", "Bearer webhook-token")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	first := post()
	if first.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}
	second := post()
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status=%d want 429 body=%s", second.Code, second.Body.String())
	}
}

func TestInboundSettingsRequiresAdmin(t *testing.T) {
	t.Parallel()
	api := &API{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/settings", nil)
	req = req.WithContext(withUser(t.Context(), User{Role: "company_owner"}))
	rec := httptest.NewRecorder()
	api.inboundSettings(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestInboundSettingsReturnsStats(t *testing.T) {
	t.Parallel()
	capture := &inboundSettingsCapture{}
	api := &API{Clients: clients.Clients{Inbound: capture}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/settings", nil)
	req = req.WithContext(withUser(t.Context(), User{ID: "admin", Role: "platform_admin"}))
	rec := httptest.NewRecorder()
	api.inboundSettings(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `"webhook_count":3`) || !strings.Contains(body, `"receive_address":"7e1432246b724f3bcd6c@cloudmailin.net"`) {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
	md, _ := metadata.FromOutgoingContext(capture.gotCtx)
	roles := md.Get(grpcutil.ActorRoleMetadataKey)
	if len(roles) != 1 || roles[0] != "platform_admin" {
		t.Fatalf("inbound settings RPC must carry x-actor-role, got %v", roles)
	}
}

func TestInboundUpdateSettingsAdmin(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Inbound: inboundClient{}}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/settings", strings.NewReader(`{"enabled":false,"receive_address":"new@cloudmailin.net"}`))
	req = req.WithContext(withUser(t.Context(), User{ID: "admin", Role: "platform_admin"}))
	rec := httptest.NewRecorder()
	api.updateInboundSettings(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInboundCompanyRejectsPlatformAdmin(t *testing.T) {
	t.Parallel()
	api := &API{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/companies/c1", nil)
	req.SetPathValue("company_id", "c1")
	req = req.WithContext(withUser(t.Context(), User{ID: "admin", Role: "platform_admin"}))
	rec := httptest.NewRecorder()
	api.inboundCompany(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestInboundCompanyRejectsWrongCompany(t *testing.T) {
	t.Parallel()
	api := &API{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/companies/c2", nil)
	req.SetPathValue("company_id", "c2")
	req = req.WithContext(withUser(t.Context(), User{ID: "u1", Role: "company_owner", CompanyID: "c1"}))
	rec := httptest.NewRecorder()
	api.inboundCompany(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestInboundCompanyOwnerReadsOwn(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Inbound: inboundClient{}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/companies/c1", nil)
	req.SetPathValue("company_id", "c1")
	req = req.WithContext(withUser(t.Context(), User{ID: "u1", Role: "company_owner", CompanyID: "c1"}))
	rec := httptest.NewRecorder()
	api.inboundCompany(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `"sender_email":"1c@company.ru"`) {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
}

func TestInboundCompanyReadAllowsCompanyStaff(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Inbound: inboundClient{}}}
	cases := []struct {
		name string
		user User
		want int
	}{
		{"owner", User{ID: "u1", Role: "company_owner", CompanyID: "c1"}, http.StatusOK},
		{"admin", User{ID: "u2", Role: "company_admin", CompanyID: "c1"}, http.StatusOK},
		{"purchaser", User{ID: "u3", Role: "purchaser", CompanyID: "c1"}, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/companies/c1", nil)
			req.SetPathValue("company_id", "c1")
			req = req.WithContext(withUser(t.Context(), tc.user))
			rec := httptest.NewRecorder()
			api.inboundCompany(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tc.want, rec.Body.String())
			}
			if tc.want == http.StatusOK && !strings.Contains(rec.Body.String(), `"sender_email":"1c@company.ru"`) {
				t.Fatalf("body=%s", rec.Body.String())
			}
		})
	}
}

func TestInboundCompanyUpdateRequiresOwner(t *testing.T) {
	t.Parallel()
	api := &API{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/companies/c1", strings.NewReader(`{"sender_email":"s@x.io"}`))
	req.SetPathValue("company_id", "c1")
	req = req.WithContext(withUser(t.Context(), User{ID: "u1", Role: "company_admin", CompanyID: "c1"}))
	rec := httptest.NewRecorder()
	api.updateInboundCompany(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestInboundCompanyUpdateOwner(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Inbound: inboundClient{}}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/companies/c1", strings.NewReader(`{"receive_address":"7e1432246b724f3bcd6c@cloudmailin.net","sender_email":"1c@x.ru","enabled":true}`))
	req.SetPathValue("company_id", "c1")
	req = req.WithContext(withUser(t.Context(), User{ID: "u1", Role: "company_owner", CompanyID: "c1"}))
	rec := httptest.NewRecorder()
	api.updateInboundCompany(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `"sender_email":"1c@x.ru"`) {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
}

func TestInboundCompanyUpdateRecordsAudit(t *testing.T) {
	t.Parallel()
	audit := &recordingAudit{}
	api := &API{Clients: clients.Clients{Inbound: inboundClient{}, Audit: audit}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/companies/c1", strings.NewReader(`{"sender_email":"1c@x.ru","enabled":true}`))
	req.SetPathValue("company_id", "c1")
	req = req.WithContext(withUser(t.Context(), User{ID: "u1", Login: "owner", Role: "company_owner", CompanyID: "c1"}))
	rec := httptest.NewRecorder()
	api.updateInboundCompany(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := audit.lastType(); got != "inbound_address_updated" {
		t.Fatalf("audit type=%q", got)
	}
	if got := audit.lastCompany(); got != "c1" {
		t.Fatalf("audit company=%q", got)
	}
}

func TestInboundMessagesCompanyAdmin(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Inbound: inboundClient{}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/companies/c1/messages", nil)
	req.SetPathValue("company_id", "c1")
	req = req.WithContext(withUser(t.Context(), User{ID: "u2", Role: "company_admin", CompanyID: "c1"}))
	rec := httptest.NewRecorder()
	api.inboundMessages(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `"subject":"Заказ №123"`) {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
}

func TestInboundMessagesOwner(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Inbound: inboundClient{}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/companies/c1/messages", nil)
	req.SetPathValue("company_id", "c1")
	req = req.WithContext(withUser(t.Context(), User{ID: "u1", Role: "company_owner", CompanyID: "c1"}))
	rec := httptest.NewRecorder()
	api.inboundMessages(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `"status":"received"`) || !strings.Contains(body, `"subject":"Заказ №123"`) || !strings.Contains(body, `"name":"report.xlsx"`) {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
}

func TestInboundMessageFileOwnerReturnsBody(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Inbound: inboundClient{}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/companies/c1/messages/m1/files/a1", nil)
	req.SetPathValue("company_id", "c1")
	req.SetPathValue("message_id", "m1")
	req.SetPathValue("attachment_id", "a1")
	req = req.WithContext(withUser(t.Context(), User{ID: "u1", Role: "company_owner", CompanyID: "c1"}))
	rec := httptest.NewRecorder()
	api.inboundMessageFile(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "xlsx" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" {
		t.Fatal("content type header missing")
	}
	if cd := rec.Header().Get("Content-Disposition"); cd == "" {
		t.Fatal("content disposition header missing")
	}
}

func TestInboundDeliveriesAdmin(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Inbound: inboundClient{}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/deliveries", nil)
	req = req.WithContext(withUser(t.Context(), User{ID: "admin", Role: "platform_admin"}))
	rec := httptest.NewRecorder()
	api.inboundDeliveries(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"provider_message_id":"abc"`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInboundDeliveriesRequiresAdmin(t *testing.T) {
	t.Parallel()
	api := &API{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/deliveries", nil)
	req = req.WithContext(withUser(t.Context(), User{Role: "company_owner"}))
	rec := httptest.NewRecorder()
	api.inboundDeliveries(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestPresentInboundSettingsNil(t *testing.T) {
	t.Parallel()
	got := presentInboundSettings(nil)
	if got.Enabled || got.ReceiveAddress != "" || got.WebhookCount != 0 {
		t.Fatalf("expected zero value for nil input, got %+v", got)
	}
}

func TestPresentInboundCompanyNil(t *testing.T) {
	t.Parallel()
	got := presentInboundCompany(nil)
	if got.CompanyID != "" || got.ReceiveAddress != "" || got.SenderEmail != "" {
		t.Fatalf("expected zero value for nil input, got %+v", got)
	}
}

func TestPresentInboundMessageNil(t *testing.T) {
	t.Parallel()
	got := presentInboundMessage(nil)
	if got.ID != "" || got.Subject != "" || got.Attachments != nil {
		t.Fatalf("expected zero value for nil input, got %+v", got)
	}
}

func TestInboundDeliveriesOmitsSubjectAndSender(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Inbound: inboundClient{}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbound/deliveries", nil)
	req = req.WithContext(withUser(t.Context(), User{ID: "admin", Role: "platform_admin"}))
	rec := httptest.NewRecorder()
	api.inboundDeliveries(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
	if strings.Contains(body, `"subject":"Заказ №456"`) || strings.Contains(body, `"envelope_from":"1c@company.ru"`) {
		t.Fatalf("platform deliveries must omit subject/from, body=%s", body)
	}
	if !strings.Contains(body, `"status":"received"`) || !strings.Contains(body, `"provider_message_id":"abc"`) {
		t.Fatalf("status/id still required, body=%s", body)
	}
}

func TestInboundUpdateCompanyForwardsSenderEmail(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Inbound: inboundClient{}}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/companies/c1", strings.NewReader(`{"sender_email":"new@x.ru","enabled":true}`))
	req.SetPathValue("company_id", "c1")
	req = req.WithContext(withUser(t.Context(), User{ID: "u1", Role: "company_owner", CompanyID: "c1"}))
	rec := httptest.NewRecorder()
	api.updateInboundCompany(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `"sender_email":"new@x.ru"`) {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
}

func configForTest() config.Config {
	return config.Config{AllowedOrigins: "*", InboundWebhook: "webhook-token"}
}

func webhookRequest() *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/webhook", strings.NewReader(`{"message_id":"abc"}`))
	req.Header.Set("Authorization", "Bearer webhook-token")
	return req
}

func TestInboundWebhookRecordsReceivedAudit(t *testing.T) {
	t.Parallel()
	audit := &recordingAudit{err: errors.New("audit down")}
	h := New(configForTest(), clients.Clients{
		Inbound: ingestResultClient{
			status:    inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_RECEIVED,
			companyID: "c1",
		},
		Audit: audit,
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, webhookRequest())
	if rec.Code != http.StatusOK {
		t.Fatalf("ingest must succeed when audit fails, status=%d body=%s", rec.Code, rec.Body.String())
	}
	if audit.lastType() != "inbound_webhook_received" {
		t.Fatalf("audit type=%q", audit.lastType())
	}
	if audit.lastCompany() != "c1" {
		t.Fatalf("audit company=%q", audit.lastCompany())
	}
}

func TestInboundWebhookUnknownSenderRecordsRejectedAudit(t *testing.T) {
	t.Parallel()
	audit := &recordingAudit{err: errors.New("audit down")}
	h := New(configForTest(), clients.Clients{
		Inbound: ingestResultClient{
			status: inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_ERROR_UNKNOWN_ADDRESS,
		},
		Audit: audit,
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, webhookRequest())
	if rec.Code != http.StatusOK {
		t.Fatalf("unknown sender is still 200, status=%d body=%s", rec.Code, rec.Body.String())
	}
	if audit.lastType() != "inbound_rejected" {
		t.Fatalf("audit type=%q", audit.lastType())
	}
}

func TestInboundWebhookInvalidPayloadRecordsRejectedAudit(t *testing.T) {
	t.Parallel()
	audit := &recordingAudit{err: errors.New("audit down")}
	h := New(configForTest(), clients.Clients{
		Inbound: ingestResultClient{err: status.Error(codes.InvalidArgument, "invalid payload")},
		Audit:   audit,
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, webhookRequest())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
	if rec.Code >= 500 {
		t.Fatalf("audit error must not 5xx poison, status=%d", rec.Code)
	}
	if !audit.called() || audit.lastType() != "inbound_rejected" {
		t.Fatalf("audit type=%q called=%v", audit.lastType(), audit.called())
	}
}
