package grpcapi

import (
	"errors"

	"google.golang.org/grpc/status"

	"order-fill/backend/pkg/apperr"
	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/password"
)

func toStatus(err error) error {
	if err == nil {
		return nil
	}
	mapped := mapErr(err)
	return status.Error(apperr.GRPCCode(mapped), mapped.Error())
}

func mapErr(err error) error {
	if _, ok := apperr.AsError(err); ok {
		return err
	}
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return apperr.Wrap(apperr.KindUnauthenticated, "unauthorized", err)
	case errors.Is(err, domain.ErrNotFound):
		return apperr.Wrap(apperr.KindNotFound, "not found", err)
	case errors.Is(err, domain.ErrConflict):
		return apperr.Wrap(apperr.KindConflict, "conflict", err)
	case errors.Is(err, domain.ErrInvalid), errors.Is(err, domain.ErrInvalidLoginSlug), errors.Is(err, password.ErrPassword):
		return apperr.Wrap(apperr.KindInvalid, "invalid", err)
	default:
		return apperr.New(apperr.KindInternal, "internal")
	}
}
