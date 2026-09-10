package brand

import (
	"context"

	"order-fill/backend/pkg/grpcutil"
	brandv1 "order-fill/backend/proto/gen/go/orderfill/brand/v1"
	"order-fill/backend/services/document-service/internal/domain/brand"
)

type Client interface {
	Detect(ctx context.Context, group, fileName string) (brandKey, variant string, err error)
	Policy(ctx context.Context, brandKey, variant string) (brand.RuleConfig, error)
}

type GRPC struct {
	client brandv1.BrandServiceClient
}

func Dial(ctx context.Context, addr string) (*GRPC, error) {
	conn, err := grpcutil.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &GRPC{client: brandv1.NewBrandServiceClient(conn)}, nil
}

func (c *GRPC) Detect(ctx context.Context, group, fileName string) (string, string, error) {
	resp, err := c.client.DetectBrand(ctx, &brandv1.DetectBrandRequest{NomenclatureGroup: group, FileName: fileName})
	if err != nil {
		return "", "", err
	}
	return resp.GetBrand(), resp.GetVariant(), nil
}

func (c *GRPC) Policy(ctx context.Context, brandKey, variant string) (brand.RuleConfig, error) {
	resp, err := c.client.GetBrandPolicy(ctx, &brandv1.GetBrandPolicyRequest{Brand: brandKey, Variant: variant})
	if err != nil {
		return brand.RuleConfig{}, err
	}
	return RuleFromProto(resp.GetPolicy()), nil
}

func RuleFromProto(p *brandv1.BrandPolicy) brand.RuleConfig {
	if p == nil {
		return brand.RuleConfig{}
	}
	return brand.RuleConfig{
		Key:                     p.GetBrand(),
		Label:                   p.GetLabel(),
		Adjustment:              brand.Adjustment(p.GetAdjustment()),
		Multiple:                int(p.GetQuantityMultiple()),
		AdjustmentLabel:         p.GetAdjustmentLabel(),
		AdjustmentComment:       p.GetAdjustmentComment(),
		PreserveArticleHyphen:   p.GetPreserveHyphen(),
		ArticlePrefixAliases:    p.GetPrefixAliases(),
		BlankQuantityHeader:     p.GetBlankQuantityHeader(),
		BlankBoxHeader:          p.GetBlankBoxHeader(),
		RequireUnit:             p.RequireUnit,
		AllowSmallPositiveOrder: p.GetAllowSmallPositiveOrder(),
		BlankLayout:             p.GetBlankLayout(),
	}
}
