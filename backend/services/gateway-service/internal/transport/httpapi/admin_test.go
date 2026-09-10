package httpapi

import (
	"testing"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func TestCompanyMatchingMode(t *testing.T) {
	t.Parallel()
	if got := companyMatchingMode(nil); got != "standard" {
		t.Fatalf("nil got %q", got)
	}
	if got := companyMatchingMode(&identityv1.Company{}); got != "standard" {
		t.Fatalf("zero got %q", got)
	}
	if got := companyMatchingMode(&identityv1.Company{MatchingMode: commonv1.MatchingMode_MATCHING_MODE_SMART}); got != "smart" {
		t.Fatalf("smart got %q", got)
	}
}

func TestProtoMatchingMode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want commonv1.MatchingMode
	}{
		{"smart", commonv1.MatchingMode_MATCHING_MODE_SMART},
		{"SMART", commonv1.MatchingMode_MATCHING_MODE_SMART},
		{"standard", commonv1.MatchingMode_MATCHING_MODE_STANDARD},
		{"", commonv1.MatchingMode_MATCHING_MODE_UNSPECIFIED},
		{"nope", commonv1.MatchingMode_MATCHING_MODE_UNSPECIFIED},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := protoMatchingMode(tc.in); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestPresentCompanyForHidesMatchingMode(t *testing.T) {
	t.Parallel()
	api := &API{}
	company := &identityv1.Company{Id: "c1", MatchingMode: commonv1.MatchingMode_MATCHING_MODE_SMART}
	admin := api.presentCompanyFor(t.Context(), User{Role: "platform_admin"}, company)
	if admin["matching_mode"] != "smart" {
		t.Fatalf("admin got %#v", admin["matching_mode"])
	}
	purchaser := api.presentCompanyFor(t.Context(), User{Role: "purchaser"}, company)
	if _, ok := purchaser["matching_mode"]; ok {
		t.Fatalf("purchaser saw matching_mode: %#v", purchaser)
	}
}
