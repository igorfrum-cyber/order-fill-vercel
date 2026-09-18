package grpcapi

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"order-fill/backend/pkg/grpcutil"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	inboundv1 "order-fill/backend/proto/gen/go/orderfill/inbound/v1"
	"order-fill/backend/services/inbound-service/internal/domain"
	"order-fill/backend/services/inbound-service/internal/service/inbound"
)

type Server struct {
	inboundv1.UnimplementedInboundServiceServer
	svc         *inbound.Service
	workerToken string
}

func NewServer(svc *inbound.Service, workerToken string) *Server {
	return &Server{svc: svc, workerToken: workerToken}
}

func New(handler inboundv1.InboundServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		inboundv1.RegisterInboundServiceServer(s, handler)
	}
	return s
}

func (s *Server) requireWorker(ctx context.Context) error {
	if !grpcutil.WorkerAuthorized(ctx, s.workerToken) {
		return status.Error(codes.PermissionDenied, "worker token required")
	}
	return nil
}

func (s *Server) requireAdmin(ctx context.Context) error {
	if grpcutil.ActorRole(ctx) != "platform_admin" {
		return status.Error(codes.PermissionDenied, "platform admin role required")
	}
	return nil
}

func (s *Server) requireOwner(ctx context.Context, meta *commonv1.RequestMeta, companyID string) error {
	role := grpcutil.ActorRole(ctx)
	if role == "platform_admin" {
		return nil
	}
	if role == "company_owner" && meta != nil && meta.GetCompanyId() == companyID {
		return nil
	}
	return status.Error(codes.PermissionDenied, "not allowed")
}

func (s *Server) requireCompanyRead(ctx context.Context, meta *commonv1.RequestMeta, companyID string) error {
	role := grpcutil.ActorRole(ctx)
	if role == "platform_admin" {
		return status.Error(codes.NotFound, "not found")
	}
	if (role == "company_owner" || role == "company_admin") && meta != nil && meta.GetCompanyId() == companyID {
		return nil
	}
	return status.Error(codes.PermissionDenied, "not allowed")
}

func (s *Server) IngestWebhook(ctx context.Context, req *inboundv1.IngestWebhookRequest) (*inboundv1.IngestWebhookResponse, error) {
	if err := s.requireWorker(ctx); err != nil {
		return nil, err
	}
	if len(req.GetRawPayload()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "raw payload is required")
	}
	msg, err := s.svc.IngestWebhook(ctx, req.GetRawPayload())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPayloadTooLarge):
			return nil, status.Error(codes.InvalidArgument, "payload too large")
		case errors.Is(err, domain.ErrInvalidPayload):
			return nil, status.Error(codes.InvalidArgument, "invalid payload")
		default:
			return nil, status.Error(codes.Internal, "ingest failed")
		}
	}
	return &inboundv1.IngestWebhookResponse{
		CompanyId: msg.CompanyID,
		Status:    protoStatus(msg.Status),
		ErrorCode: msg.ErrorCode,
	}, nil
}

func (s *Server) GetSettings(ctx context.Context, _ *inboundv1.GetSettingsRequest) (*inboundv1.GetSettingsResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	st, err := s.svc.GetSettings(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read settings")
	}
	return &inboundv1.GetSettingsResponse{Settings: protoSettings(st)}, nil
}

func (s *Server) UpdateSettings(ctx context.Context, req *inboundv1.UpdateSettingsRequest) (*inboundv1.UpdateSettingsResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	st, err := s.svc.UpdateSettings(ctx, req.GetEnabled(), req.GetReceiveAddress())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update settings")
	}
	return &inboundv1.UpdateSettingsResponse{Settings: protoSettings(st)}, nil
}

func (s *Server) GetCompanyInbound(ctx context.Context, req *inboundv1.GetCompanyInboundRequest) (*inboundv1.GetCompanyInboundResponse, error) {
	if err := s.requireCompanyRead(ctx, req.GetMeta(), req.GetCompanyId()); err != nil {
		return nil, err
	}
	ci, err := s.svc.GetCompanyInbound(ctx, req.GetCompanyId())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "company inbound not configured")
		}
		return nil, status.Error(codes.Internal, "failed to read company inbound")
	}
	return &inboundv1.GetCompanyInboundResponse{CompanyInbound: protoCompanyInbound(ci)}, nil
}

func (s *Server) UpdateCompanyInbound(ctx context.Context, req *inboundv1.UpdateCompanyInboundRequest) (*inboundv1.UpdateCompanyInboundResponse, error) {
	if err := s.requireOwner(ctx, req.GetMeta(), req.GetCompanyId()); err != nil {
		return nil, err
	}
	if grpcutil.ActorRole(ctx) != "company_owner" && grpcutil.ActorRole(ctx) != "platform_admin" {
		return nil, status.Error(codes.PermissionDenied, "owner role required")
	}
	ci, err := s.svc.UpdateCompanyInbound(ctx, req.GetCompanyId(), req.GetReceiveAddress(), req.GetSenderEmail(), req.GetEnabled())
	if err != nil {
		if errors.Is(err, domain.ErrInvalid) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to update company inbound")
	}
	return &inboundv1.UpdateCompanyInboundResponse{CompanyInbound: protoCompanyInbound(ci)}, nil
}

func (s *Server) ListMessages(ctx context.Context, req *inboundv1.ListMessagesRequest) (*inboundv1.ListMessagesResponse, error) {
	if err := s.requireCompanyRead(ctx, req.GetMeta(), req.GetCompanyId()); err != nil {
		return nil, err
	}
	msgs, err := s.svc.ListMessages(ctx, req.GetCompanyId(), int(req.GetLimit()))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list messages")
	}
	out := make([]*inboundv1.InboundMessageSummary, len(msgs))
	for i, m := range msgs {
		pm := protoMessageSummary(m)
		pm.BodyText = ""
		pm.BodyHtml = ""
		atts, err := s.svc.GetMessageAttachments(ctx, m.ID)
		if err == nil && len(atts) > 0 {
			pm.Attachments = make([]*inboundv1.InboundAttachment, len(atts))
			for j, a := range atts {
				pm.Attachments[j] = protoAttachment(a)
			}
		}
		out[i] = pm
	}
	return &inboundv1.ListMessagesResponse{Messages: out}, nil
}

func (s *Server) GetMessage(ctx context.Context, req *inboundv1.GetMessageRequest) (*inboundv1.GetMessageResponse, error) {
	if err := s.requireCompanyRead(ctx, req.GetMeta(), req.GetCompanyId()); err != nil {
		return nil, err
	}
	msg, err := s.svc.GetMessage(ctx, req.GetCompanyId(), req.GetMessageId())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "message not found")
		}
		if errors.Is(err, domain.ErrUnauthorized) {
			return nil, status.Error(codes.PermissionDenied, "not allowed")
		}
		return nil, status.Error(codes.Internal, "failed to read message")
	}
	return &inboundv1.GetMessageResponse{Message: protoMessageSummary(msg)}, nil
}

func (s *Server) GetMessageFile(ctx context.Context, req *inboundv1.GetMessageFileRequest) (*inboundv1.GetMessageFileResponse, error) {
	companyID := req.GetMeta().GetCompanyId()
	if err := s.requireCompanyRead(ctx, req.GetMeta(), companyID); err != nil {
		return nil, err
	}
	att, data, err := s.svc.GetMessageFile(ctx, companyID, req.GetMessageId(), req.GetAttachmentId())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "attachment not found")
		}
		if errors.Is(err, domain.ErrUnauthorized) {
			return nil, status.Error(codes.PermissionDenied, "not allowed")
		}
		return nil, status.Error(codes.Internal, "failed to read attachment")
	}
	return &inboundv1.GetMessageFileResponse{
		Attachment: protoAttachment(att),
		CompanyId:  companyID,
		Body:       data,
	}, nil
}

func (s *Server) ListDeliveries(ctx context.Context, req *inboundv1.ListDeliveriesRequest) (*inboundv1.ListDeliveriesResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	msgs, err := s.svc.ListDeliveries(ctx, int(req.GetLimit()))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list deliveries")
	}
	out := make([]*inboundv1.InboundMessageSummary, len(msgs))
	for i, m := range msgs {
		out[i] = protoMessageSummary(m)
		out[i].BodyText = ""
		out[i].BodyHtml = ""
		out[i].Subject = ""
		out[i].EnvelopeFrom = ""
	}
	return &inboundv1.ListDeliveriesResponse{Deliveries: out}, nil
}

func protoSettings(st domain.Settings) *inboundv1.InboundSettings {
	return &inboundv1.InboundSettings{
		Enabled:        st.Enabled,
		ReceiveAddress: st.ReceiveAddress,
		LastWebhookAt:  st.LastWebhookAt.Format(time.RFC3339),
		WebhookCount:   st.WebhookCount,
		ErrorCount:     st.ErrorCount,
	}
}

func protoCompanyInbound(ci domain.CompanyInbound) *inboundv1.CompanyInbound {
	return &inboundv1.CompanyInbound{
		CompanyId:      ci.CompanyID,
		ReceiveAddress: ci.ReceiveAddress,
		AllowedFrom:    ci.AllowedFrom,
		SenderEmail:    ci.SenderEmail,
		Enabled:        ci.Enabled,
	}
}

func protoMessageSummary(m domain.MessageSummary) *inboundv1.InboundMessageSummary {
	return &inboundv1.InboundMessageSummary{
		Id:                m.ID,
		ProviderMessageId: m.ProviderMessageID,
		Subject:           m.Subject,
		EnvelopeFrom:      m.EnvelopeFrom,
		EnvelopeTo:        m.EnvelopeTo,
		ReceivedAt:        m.ReceivedAt.Format(time.RFC3339),
		CompanyId:         m.CompanyID,
		Status:            protoStatus(m.Status),
		ErrorCode:         m.ErrorCode,
		AttachmentCount:   int32(m.AttachmentCount),
		TotalBytes:        m.TotalBytes,
		BodyText:          m.BodyText,
		BodyHtml:          m.BodyHTML,
	}
}

func protoStatus(st domain.InboundStatus) inboundv1.InboundMessageStatus {
	switch st {
	case domain.StatusReceived:
		return inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_RECEIVED
	case domain.StatusProcessed:
		return inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_PROCESSED
	case domain.StatusErrorUnknownAddress:
		return inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_ERROR_UNKNOWN_ADDRESS
	case domain.StatusErrorMismatchFrom:
		return inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_ERROR_MISMATCH_FROM
	case domain.StatusErrorNoAttachments:
		return inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_ERROR_NO_ATTACHMENTS
	case domain.StatusErrorTooLarge:
		return inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_ERROR_TOO_LARGE
	default:
		return inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_UNSPECIFIED
	}
}

func protoAttachment(a domain.Attachment) *inboundv1.InboundAttachment {
	return &inboundv1.InboundAttachment{
		Id:          a.ID,
		Name:        a.Name,
		ContentType: a.ContentType,
		Size:        a.Size,
	}
}
