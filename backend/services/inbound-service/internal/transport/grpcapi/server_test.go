package grpcapi_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"order-fill/backend/pkg/grpcutil"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	inboundv1 "order-fill/backend/proto/gen/go/orderfill/inbound/v1"
	"order-fill/backend/services/inbound-service/internal/domain"
	"order-fill/backend/services/inbound-service/internal/service/inbound"
	"order-fill/backend/services/inbound-service/internal/transport/grpcapi"
)

type stubStore struct {
	companies map[string]domain.CompanyInbound
	saved     []domain.MessageSummary
}

func (s *stubStore) GetSettings(context.Context) (domain.Settings, error) {
	return domain.Settings{Enabled: true}, nil
}
func (s *stubStore) UpsertSettings(context.Context, bool, string) (domain.Settings, error) {
	return domain.Settings{}, nil
}
func (s *stubStore) IncrWebhookCount(context.Context, bool) {}
func (s *stubStore) GetCompanyInbound(_ context.Context, companyID string) (domain.CompanyInbound, error) {
	ci, ok := s.companies[companyID]
	if !ok {
		return domain.CompanyInbound{}, domain.ErrNotFound
	}
	return ci, nil
}
func (s *stubStore) GetCompanyByAllowedSender(_ context.Context, senderEmail string) (domain.CompanyInbound, error) {
	for _, ci := range s.companies {
		if strings.EqualFold(ci.SenderEmail, senderEmail) {
			return ci, nil
		}
	}
	return domain.CompanyInbound{}, domain.ErrNotFound
}
func (s *stubStore) UpsertCompanyInbound(_ context.Context, companyID, receiveAddress, senderEmail string, enabled bool) (domain.CompanyInbound, error) {
	ci := domain.CompanyInbound{CompanyID: companyID, ReceiveAddress: receiveAddress, SenderEmail: senderEmail, Enabled: enabled}
	if s.companies == nil {
		s.companies = map[string]domain.CompanyInbound{}
	}
	s.companies[companyID] = ci
	return ci, nil
}
func (s *stubStore) MessageExists(context.Context, string) (bool, error) { return false, nil }
func (s *stubStore) SaveMessage(_ context.Context, msg domain.MessageSummary, _ []domain.Attachment) error {
	s.saved = append(s.saved, msg)
	return nil
}
func (s *stubStore) ListMessages(context.Context, string, int) ([]domain.MessageSummary, error) {
	return s.saved, nil
}
func (s *stubStore) GetMessage(context.Context, string) (domain.MessageSummary, error) {
	return domain.MessageSummary{}, domain.ErrNotFound
}
func (s *stubStore) GetMessageAttachments(context.Context, string) ([]domain.Attachment, error) {
	return nil, nil
}
func (s *stubStore) GetAttachment(context.Context, string) (domain.Attachment, error) {
	return domain.Attachment{}, domain.ErrNotFound
}
func (s *stubStore) GetMessageCompany(context.Context, string) (string, error) {
	return "", domain.ErrNotFound
}
func (s *stubStore) ListDeliveries(context.Context, int) ([]domain.MessageSummary, error) {
	return s.saved, nil
}

type stubObjects struct{}

func (stubObjects) Put(context.Context, string, io.Reader, int64, string) error { return nil }
func (stubObjects) Get(context.Context, string) ([]byte, string, error) {
	return nil, "", domain.ErrNotFound
}

func testServer(t *testing.T) (*grpcapi.Server, *stubStore) {
	t.Helper()
	store := &stubStore{
		companies: map[string]domain.CompanyInbound{
			"co-1": {CompanyID: "co-1", SenderEmail: "1c@co1.ru", Enabled: true},
			"co-2": {CompanyID: "co-2", SenderEmail: "1c@co2.ru", Enabled: true},
		},
	}
	return grpcapi.NewServer(inbound.New(store, stubObjects{}), "worker-token-16b!"), store
}

func incomingRole(t *testing.T, role string) context.Context {
	t.Helper()
	ctx := grpcutil.WithActorRole(t.Context(), role)
	md, _ := metadata.FromOutgoingContext(ctx)
	return metadata.NewIncomingContext(t.Context(), md)
}

func incomingWorker(t *testing.T, token string) context.Context {
	t.Helper()
	ctx := grpcutil.WithWorkerToken(t.Context(), token)
	md, _ := metadata.FromOutgoingContext(ctx)
	return metadata.NewIncomingContext(t.Context(), md)
}

func TestGetCompanyInboundAllowsCompanyAdmin(t *testing.T) {
	t.Parallel()
	srv, _ := testServer(t)
	ctx := incomingRole(t, "company_admin")
	req := &inboundv1.GetCompanyInboundRequest{
		Meta:      &commonv1.RequestMeta{CompanyId: "co-1"},
		CompanyId: "co-1",
	}
	if _, err := srv.GetCompanyInbound(ctx, req); err != nil {
		t.Fatalf("company_admin of co-1 must read inbound settings: %v", err)
	}
}

func TestGetCompanyInboundRejectsForeignCompany(t *testing.T) {
	t.Parallel()
	srv, _ := testServer(t)
	ctx := incomingRole(t, "company_admin")
	req := &inboundv1.GetCompanyInboundRequest{
		Meta:      &commonv1.RequestMeta{CompanyId: "co-1"},
		CompanyId: "co-2",
	}
	_, err := srv.GetCompanyInbound(ctx, req)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("foreign company_id must be denied, got %v", err)
	}
}

func TestIngestWebhookInvalidJSONIsInvalidArgument(t *testing.T) {
	t.Parallel()
	srv, _ := testServer(t)
	ctx := incomingWorker(t, "worker-token-16b!")
	_, err := srv.IngestWebhook(ctx, &inboundv1.IngestWebhookRequest{RawPayload: []byte("{")})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("got %v, want InvalidArgument so CloudMailin stops retrying poison payloads", err)
	}
}

func TestIngestWebhookPayloadTooLargeIsInvalidArgument(t *testing.T) {
	t.Parallel()
	srv, _ := testServer(t)
	ctx := incomingWorker(t, "worker-token-16b!")
	_, err := srv.IngestWebhook(ctx, &inboundv1.IngestWebhookRequest{RawPayload: make([]byte, 32<<20+1)})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("got %v, want InvalidArgument for oversized payload", err)
	}
}

func TestIngestWebhookUnknownSenderReturnsRejectedStatus(t *testing.T) {
	t.Parallel()
	srv, _ := testServer(t)
	ctx := incomingWorker(t, "worker-token-16b!")
	resp, err := srv.IngestWebhook(ctx, &inboundv1.IngestWebhookRequest{
		RawPayload: []byte(`{"message_id":"m-unknown","envelope":{"from":"nobody@x.io","to":"in@x.io"}}`),
	})
	if err != nil {
		t.Fatalf("unknown sender must still ingest: %v", err)
	}
	if resp.GetStatus() != inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_ERROR_UNKNOWN_ADDRESS {
		t.Fatalf("status=%v", resp.GetStatus())
	}
}

func TestIngestWebhookReceivedReturnsCompany(t *testing.T) {
	t.Parallel()
	srv, _ := testServer(t)
	ctx := incomingWorker(t, "worker-token-16b!")
	resp, err := srv.IngestWebhook(ctx, &inboundv1.IngestWebhookRequest{
		RawPayload: []byte(`{"message_id":"m-ok","envelope":{"from":"1c@co1.ru","to":"in@x.io"},"attachments":[{"file_name":"sales.xlsx","content":"ZGF0YQ=="}]}`),
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if resp.GetStatus() != inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_RECEIVED {
		t.Fatalf("status=%v", resp.GetStatus())
	}
	if resp.GetCompanyId() != "co-1" {
		t.Fatalf("company_id=%q", resp.GetCompanyId())
	}
}

func TestListDeliveriesOmitsSubjectAndSender(t *testing.T) {
	t.Parallel()
	srv, store := testServer(t)
	store.saved = []domain.MessageSummary{{
		ID: "d1", ProviderMessageID: "abc", Subject: "Заказ №456", EnvelopeFrom: "1c@company.ru",
		Status: domain.StatusReceived, BodyText: "secret", BodyHTML: "<p>secret</p>",
	}}
	ctx := incomingRole(t, "platform_admin")
	resp, err := srv.ListDeliveries(ctx, &inboundv1.ListDeliveriesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetDeliveries()) != 1 {
		t.Fatalf("deliveries=%d", len(resp.GetDeliveries()))
	}
	got := resp.GetDeliveries()[0]
	if got.GetSubject() != "" || got.GetEnvelopeFrom() != "" {
		t.Fatalf("platform feed must hide subject/from, got subject=%q from=%q", got.GetSubject(), got.GetEnvelopeFrom())
	}
	if got.GetBodyText() != "" || got.GetBodyHtml() != "" {
		t.Fatalf("platform feed must hide bodies")
	}
	if got.GetStatus() != inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_RECEIVED {
		t.Fatalf("status=%v", got.GetStatus())
	}
}
