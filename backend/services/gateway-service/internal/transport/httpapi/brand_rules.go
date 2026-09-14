package httpapi

import (
	"net/http"

	brandv1 "order-fill/backend/proto/gen/go/orderfill/brand/v1"
)

type brandRuleJSON struct {
	Brand                   string   `json:"brand"`
	Variant                 string   `json:"variant"`
	Label                   string   `json:"label"`
	Adjustment              string   `json:"adjustment"`
	AdjustmentLabel         string   `json:"adjustment_label"`
	AdjustmentComment       string   `json:"adjustment_comment"`
	QuantityMultiple        int32    `json:"quantity_multiple"`
	MinQuantity             int32    `json:"min_quantity"`
	PreserveHyphen          bool     `json:"preserve_hyphen"`
	PrefixAliases           []string `json:"prefix_aliases"`
	BlankQuantityHeader     string   `json:"blank_quantity_header"`
	BlankBoxHeader          string   `json:"blank_box_header"`
	BlankLayout             string   `json:"blank_layout"`
	AllowSmallPositiveOrder bool     `json:"allow_small_positive_order"`
	RequireUnit             *bool    `json:"require_unit"`
}

func (a *API) listBrandRules(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	if user.Role != "platform_admin" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	listed, err := a.Clients.Brand.ListBrands(r.Context(), &brandv1.ListBrandsRequest{})
	if err != nil {
		writeGRPCError(w, "list_brand_rules_failed", err)
		return
	}
	rules := make([]brandRuleJSON, 0, len(listed.GetBrands()))
	for _, key := range listed.GetBrands() {
		response, err := a.Clients.Brand.GetBrandPolicy(r.Context(), &brandv1.GetBrandPolicyRequest{RequestId: a.meta(user).GetRequestId(), Brand: key})
		if err != nil {
			writeGRPCError(w, "get_brand_rule_failed", err)
			return
		}
		policy := response.GetPolicy()
		rules = append(rules, presentBrandRule(policy))
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rules})
}

func (a *API) updateBrandRule(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	if user.Role != "platform_admin" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	var payload brandRuleJSON
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	payload.Brand = r.PathValue("brand")
	response, err := a.Clients.Brand.UpdateBrandPolicy(a.jobCtx(r, user), &brandv1.UpdateBrandPolicyRequest{
		Meta: a.meta(user),
		Policy: &brandv1.BrandPolicy{
			Brand: payload.Brand, Label: payload.Label, Adjustment: payload.Adjustment,
			AdjustmentLabel: payload.AdjustmentLabel, AdjustmentComment: payload.AdjustmentComment,
			QuantityMultiple: payload.QuantityMultiple, MinQuantity: payload.MinQuantity,
			PreserveHyphen: payload.PreserveHyphen, PrefixAliases: payload.PrefixAliases,
			BlankQuantityHeader: payload.BlankQuantityHeader, BlankBoxHeader: payload.BlankBoxHeader,
			BlankLayout: payload.BlankLayout, AllowSmallPositiveOrder: payload.AllowSmallPositiveOrder,
			RequireUnit: payload.RequireUnit,
		},
	})
	if err != nil {
		writeGRPCError(w, "update_brand_rule_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentBrandRule(response.GetPolicy()))
}

func presentBrandRule(policy *brandv1.BrandPolicy) brandRuleJSON {
	return brandRuleJSON{
		Brand: policy.GetBrand(), Variant: policy.GetVariant(), Label: policy.GetLabel(), Adjustment: policy.GetAdjustment(),
		AdjustmentLabel: policy.GetAdjustmentLabel(), AdjustmentComment: policy.GetAdjustmentComment(),
		QuantityMultiple: policy.GetQuantityMultiple(), MinQuantity: policy.GetMinQuantity(),
		PreserveHyphen: policy.GetPreserveHyphen(), PrefixAliases: policy.GetPrefixAliases(),
		BlankQuantityHeader: policy.GetBlankQuantityHeader(), BlankBoxHeader: policy.GetBlankBoxHeader(),
		BlankLayout: policy.GetBlankLayout(), AllowSmallPositiveOrder: policy.GetAllowSmallPositiveOrder(),
		RequireUnit: policy.RequireUnit,
	}
}
