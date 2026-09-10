package files

import "context"

type Client interface {
	Get(ctx context.Context, id string) (name string, body []byte, err error)
	Put(ctx context.Context, name string, body []byte) (id string, err error)
}
