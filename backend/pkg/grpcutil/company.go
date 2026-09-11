package grpcutil

import "context"

type companyKey struct{}

func WithCompany(ctx context.Context, companyID string) context.Context {
	if companyID == "" {
		return ctx
	}
	return context.WithValue(ctx, companyKey{}, companyID)
}

func Company(ctx context.Context) string {
	value, _ := ctx.Value(companyKey{}).(string)
	return value
}
