package apperr

import (
	"errors"
	"net/http"

	"google.golang.org/grpc/codes"
)

type Kind string

const (
	KindInvalid            Kind = "invalid"
	KindUnauthenticated    Kind = "unauthenticated"
	KindPermissionDenied   Kind = "permission_denied"
	KindNotFound           Kind = "not_found"
	KindConflict           Kind = "conflict"
	KindFailedPrecondition Kind = "failed_precondition"
	KindInternal           Kind = "internal"
	KindUnavailable        Kind = "unavailable"
)

type Error struct {
	Kind Kind
	Msg  string
	err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.err != nil {
		return e.err.Error()
	}
	return string(e.Kind)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func New(kind Kind, msg string) *Error {
	return &Error{Kind: kind, Msg: msg}
}

func Wrap(kind Kind, msg string, err error) *Error {
	return &Error{Kind: kind, Msg: msg, err: err}
}

func AsError(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}

func KindOf(err error) Kind {
	if e, ok := AsError(err); ok {
		return e.Kind
	}
	return KindInternal
}

func HTTPStatus(err error) int {
	switch KindOf(err) {
	case KindInvalid:
		return http.StatusBadRequest
	case KindUnauthenticated:
		return http.StatusUnauthorized
	case KindPermissionDenied:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict, KindFailedPrecondition:
		return http.StatusConflict
	case KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func GRPCCode(err error) codes.Code {
	switch KindOf(err) {
	case KindInvalid:
		return codes.InvalidArgument
	case KindUnauthenticated:
		return codes.Unauthenticated
	case KindPermissionDenied:
		return codes.PermissionDenied
	case KindNotFound:
		return codes.NotFound
	case KindConflict:
		return codes.AlreadyExists
	case KindFailedPrecondition:
		return codes.FailedPrecondition
	case KindUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}
