package domain

type Adjustment string

const (
	AdjustmentNone            Adjustment = "none"
	AdjustmentBox             Adjustment = "box"
	AdjustmentMultiple        Adjustment = "multiple"
	AdjustmentNearestMultiple Adjustment = "nearestMultiple"
	AdjustmentMinimum         Adjustment = "minimum"
)

type Policy struct {
	Key                     string
	Label                   string
	Variant                 string
	Adjustment              Adjustment
	Multiple                int
	MinQuantity             int
	AdjustmentLabel         string
	AdjustmentComment       string
	PreserveArticleHyphen   bool
	ArticlePrefixAliases    []string
	BlankQuantityHeader     string
	BlankBoxHeader          string
	AllowSmallPositiveOrder bool
	BlankLayout             string
}
