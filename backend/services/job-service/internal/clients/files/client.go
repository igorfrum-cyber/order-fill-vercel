package files

import (
	"context"
	"strings"

	"order-fill/backend/pkg/grpcutil"
	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	"order-fill/backend/services/job-service/internal/domain"
)

type Client struct {
	api filesv1.FileServiceClient
}

func Dial(ctx context.Context, addr string) (*Client, error) {
	conn, err := grpcutil.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &Client{api: filesv1.NewFileServiceClient(conn)}, nil
}

func (c *Client) Describe(ctx context.Context, ids []string) ([]domain.FileRef, error) {
	out := make([]domain.FileRef, 0, len(ids))
	for _, id := range ids {
		resp, err := c.api.GetObject(ctx, &filesv1.GetObjectRequest{Id: id})
		if err != nil {
			return nil, err
		}
		obj := resp.GetObject()
		kind, _, _ := strings.Cut(obj.GetKey(), "/")
		out = append(out, domain.FileRef{
			ID: obj.GetId(), Kind: kind, ObjectKey: obj.GetKey(),
			Name: obj.GetName(), ContentType: obj.GetContentType(),
		})
	}
	return out, nil
}
