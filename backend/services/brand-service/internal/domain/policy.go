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
	Key                     string     `json:"brand"`
	Label                   string     `json:"label"`
	Variant                 string     `json:"variant,omitempty"`
	Adjustment              Adjustment `json:"adjustment"`
	Multiple                int        `json:"quantity_multiple"`
	MinQuantity             int        `json:"min_quantity"`
	AdjustmentLabel         string     `json:"adjustment_label"`
	AdjustmentComment       string     `json:"adjustment_comment"`
	PreserveArticleHyphen   bool       `json:"preserve_hyphen"`
	ArticlePrefixAliases    []string   `json:"prefix_aliases"`
	BlankQuantityHeader     string     `json:"blank_quantity_header"`
	BlankBoxHeader          string     `json:"blank_box_header"`
	AllowSmallPositiveOrder bool       `json:"allow_small_positive_order"`
	BlankLayout             string     `json:"blank_layout"`
	RequireUnit             *bool      `json:"require_unit,omitempty"`
}
