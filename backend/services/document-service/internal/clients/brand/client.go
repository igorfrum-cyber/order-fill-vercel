package brand

import "context"

type Client interface {
	Detect(ctx context.Context, group, fileName string) (brand, variant string, err error)
	PolicyMultiple(ctx context.Context, brand string) (int, error)
}
