package grpcapi_test

import (
	"context"
	"io"
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
func (s *stubStore) GetCompanyByAllowedSender(context.Context, string) (domain.CompanyInbound, error) {
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
	return grpcapi.NewServer(inbound.New(store, stubObjects{}, "bucket"), "worker-token-16b!"), store
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
