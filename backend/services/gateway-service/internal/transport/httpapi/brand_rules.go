package httpapi

import (
	"net/http"

	brandv1 "order-fill/backend/proto/gen/go/orderfill/brand/v1"
)

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
	rules := make([]map[string]any, 0, len(listed.GetBrands()))
	for _, key := range listed.GetBrands() {
		response, err := a.Clients.Brand.GetBrandPolicy(r.Context(), &brandv1.GetBrandPolicyRequest{RequestId: a.meta(user).GetRequestId(), Brand: key})
		if err != nil {
			writeGRPCError(w, "get_brand_rule_failed", err)
			return
		}
		policy := response.GetPolicy()
		rule := map[string]any{
			"brand": policy.GetBrand(), "label": policy.GetLabel(), "variant": policy.GetVariant(),
			"adjustment": policy.GetAdjustment(), "adjustment_label": policy.GetAdjustmentLabel(),
			"adjustment_comment": policy.GetAdjustmentComment(), "quantity_multiple": policy.GetQuantityMultiple(),
			"min_quantity": policy.GetMinQuantity(), "preserve_hyphen": policy.GetPreserveHyphen(),
			"prefix_aliases": policy.GetPrefixAliases(), "blank_quantity_header": policy.GetBlankQuantityHeader(),
			"blank_box_header": policy.GetBlankBoxHeader(), "blank_layout": policy.GetBlankLayout(),
			"allow_small_positive_order": policy.GetAllowSmallPositiveOrder(),
		}
		if policy.RequireUnit != nil {
			rule["require_unit"] = policy.GetRequireUnit()
		}
		rules = append(rules, rule)
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rules})
}
