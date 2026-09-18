package clients

import (
	"context"

	"order-fill/backend/pkg/grpcutil"
	auditv1 "order-fill/backend/proto/gen/go/orderfill/audit/v1"
	brandv1 "order-fill/backend/proto/gen/go/orderfill/brand/v1"
	calculationv1 "order-fill/backend/proto/gen/go/orderfill/calculation/v1"
	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	inboundv1 "order-fill/backend/proto/gen/go/orderfill/inbound/v1"
	jobsv1 "order-fill/backend/proto/gen/go/orderfill/jobs/v1"
	passkeyv1 "order-fill/backend/proto/gen/go/orderfill/passkey/v1"
	twofav1 "order-fill/backend/proto/gen/go/orderfill/twofa/v1"
	"order-fill/backend/services/gateway-service/internal/config"
)

type Clients struct {
	Identity    identityv1.IdentityServiceClient
	TwoFA       twofav1.TwoFAServiceClient
	Passkey     passkeyv1.PasskeyServiceClient
	Jobs        jobsv1.JobServiceClient
	Files       filesv1.FileServiceClient
	Audit       auditv1.AuditServiceClient
	Brand       brandv1.BrandServiceClient
	Inbound     inboundv1.InboundServiceClient
	Calculation calculationv1.CalculationServiceClient
}

func Dial(ctx context.Context, cfg config.Config) (Clients, error) {
	identityConn, err := grpcutil.Dial(ctx, cfg.IdentityGRPC)
	if err != nil {
		return Clients{}, err
	}
	twoFAConn, err := grpcutil.Dial(ctx, cfg.TwoFAGRPC)
	if err != nil {
		return Clients{}, err
	}
	passkeyConn, err := grpcutil.Dial(ctx, cfg.PasskeyGRPC)
	if err != nil {
		return Clients{}, err
	}
	jobConn, err := grpcutil.Dial(ctx, cfg.JobGRPC)
	if err != nil {
		return Clients{}, err
	}
	fileConn, err := grpcutil.Dial(ctx, cfg.FileGRPC)
	if err != nil {
		return Clients{}, err
	}
	auditConn, err := grpcutil.Dial(ctx, cfg.AuditGRPC)
	if err != nil {
		return Clients{}, err
	}
	brandConn, err := grpcutil.Dial(ctx, cfg.BrandGRPC)
	if err != nil {
		return Clients{}, err
	}
	inboundConn, err := grpcutil.Dial(ctx, cfg.InboundGRPC)
	if err != nil {
		return Clients{}, err
	}
	calculationConn, err := grpcutil.Dial(ctx, cfg.CalculationGRPC)
	if err != nil {
		return Clients{}, err
	}
	return Clients{
		Identity:    identityv1.NewIdentityServiceClient(identityConn),
		TwoFA:       twofav1.NewTwoFAServiceClient(twoFAConn),
		Passkey:     passkeyv1.NewPasskeyServiceClient(passkeyConn),
		Jobs:        jobsv1.NewJobServiceClient(jobConn),
		Files:       filesv1.NewFileServiceClient(fileConn),
		Audit:       auditv1.NewAuditServiceClient(auditConn),
		Brand:       brandv1.NewBrandServiceClient(brandConn),
		Inbound:     inboundv1.NewInboundServiceClient(inboundConn),
		Calculation: calculationv1.NewCalculationServiceClient(calculationConn),
	}, nil
}
