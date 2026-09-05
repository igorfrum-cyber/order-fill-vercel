package clients

import (
	"context"

	"order-fill/backend/pkg/grpcutil"
	auditv1 "order-fill/backend/proto/gen/go/orderfill/audit/v1"
	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	jobsv1 "order-fill/backend/proto/gen/go/orderfill/jobs/v1"
	passkeyv1 "order-fill/backend/proto/gen/go/orderfill/passkey/v1"
	twofav1 "order-fill/backend/proto/gen/go/orderfill/twofa/v1"
	"order-fill/backend/services/gateway-service/internal/config"
)

type Clients struct {
	Identity identityv1.IdentityServiceClient
	TwoFA    twofav1.TwoFAServiceClient
	Passkey  passkeyv1.PasskeyServiceClient
	Jobs     jobsv1.JobServiceClient
	Files    filesv1.FileServiceClient
	Audit    auditv1.AuditServiceClient
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
	return Clients{
		Identity: identityv1.NewIdentityServiceClient(identityConn),
		TwoFA:    twofav1.NewTwoFAServiceClient(twoFAConn),
		Passkey:  passkeyv1.NewPasskeyServiceClient(passkeyConn),
		Jobs:     jobsv1.NewJobServiceClient(jobConn),
		Files:    filesv1.NewFileServiceClient(fileConn),
		Audit:    auditv1.NewAuditServiceClient(auditConn),
	}, nil
}
