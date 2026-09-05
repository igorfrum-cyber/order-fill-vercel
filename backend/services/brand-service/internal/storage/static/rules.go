package static

import "order-fill/backend/services/brand-service/internal/domain"

var policies = map[string]domain.Policy{
	"angiopharm": {
		Key:               "angiopharm",
		Label:             "ANGIOPHARM",
		Adjustment:        domain.AdjustmentBox,
		AdjustmentLabel:   "Шт. в коробке",
		AdjustmentComment: "до коробки",
	},
	"christina": {
		Key:               "christina",
		Label:             "CHRISTINA",
		Adjustment:        domain.AdjustmentMultiple,
		Multiple:          3,
		AdjustmentLabel:   "Кратность",
		AdjustmentComment: "до кратности 3",
	},
	"levissime": {
		Key:                  "levissime",
		Label:                "LeviSsime",
		Adjustment:           domain.AdjustmentBox,
		AdjustmentLabel:      "Кол-во в уп.",
		AdjustmentComment:    "до коробки",
		ArticlePrefixAliases: []string{"MT"},
		BlankQuantityHeader:  "order",
		BlankBoxHeader:       "packageQuantity",
	},
	"sothys": {
		Key:                   "sothys",
		Label:                 "SOTHYS",
		Adjustment:            domain.AdjustmentNone,
		AdjustmentLabel:       "Без округления",
		PreserveArticleHyphen: true,
		BlankLayout:           "splitVariants",
	},
	"novacutan": {
		Key:             "novacutan",
		Label:           "NOVACUTAN",
		Adjustment:      domain.AdjustmentNone,
		AdjustmentLabel: "Мин. заказ",
		BlankLayout:     "novacutan",
	},
	"skin_synergy": {
		Key:                 "skin_synergy",
		Label:               "Skin Synergy",
		Adjustment:          domain.AdjustmentNone,
		AdjustmentLabel:     "Без округления",
		BlankQuantityHeader: "exactQuantity",
	},
	"klapp": {
		Key:                 "klapp",
		Label:               "KLAPP",
		Adjustment:          domain.AdjustmentNearestMultiple,
		Multiple:            3,
		AdjustmentLabel:     "Кратность",
		AdjustmentComment:   "до кратности 3",
		BlankQuantityHeader: "order",
	},
}

func Policy(brand string) domain.Policy {
	if p, ok := policies[brand]; ok {
		return p
	}
	return policies["angiopharm"]
}

func List() []string {
	return []string{"angiopharm", "christina", "klapp", "levissime", "novacutan", "skin_synergy", "sothys"}
}
