package grpcapi

import (
	"context"
	"time"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) ListSessions(ctx context.Context, req *identityv1.ListSessionsRequest) (*identityv1.ListSessionsResponse, error) {
	actor, err := s.actor(ctx, req.GetActorUserId())
	if err != nil {
		return nil, toStatus(err)
	}
	views, err := s.auth.ListSessions(ctx, actor, req.GetSessionToken())
	if err != nil {
		return nil, toStatus(err)
	}
	out := make([]*identityv1.Session, 0, len(views))
	for _, view := range views {
		out = append(out, &identityv1.Session{
			Id:        view.ID,
			UserId:    actor.ID,
			Device:    view.Device,
			Current:   view.Current,
			CreatedAt: view.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return &identityv1.ListSessionsResponse{Sessions: out}, nil
}
