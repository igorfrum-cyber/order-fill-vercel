package brand

import "testing"

func TestRuleUnknownFallsBackToAngiopharm(t *testing.T) {
	t.Parallel()
	if got := Rule("unknown").Key; got != "angiopharm" {
		t.Fatalf("Rule(unknown).Key = %q, want angiopharm", got)
	}
}

func TestArticleNormalizeOptionsFollowsHyphenFlag(t *testing.T) {
	t.Parallel()
	if got := ArticleNormalizeOptions(Rule("sothys")); !got.PreserveHyphen {
		t.Fatal("sothys must preserve article hyphens")
	}
	if got := ArticleNormalizeOptions(Rule("levissime")); got.PreserveHyphen {
		t.Fatal("levissime must not preserve article hyphens")
	}
}

func TestCategoryCoefficientUsesBrandSpecificNovacutanRules(t *testing.T) {
	if got := CategoryCoefficient("C", Rule("novacutan")); got != 1.5 {
		t.Fatalf("expected novacutan C coefficient 1.5, got %v", got)
	}
	if got := CategoryCoefficient("C", Rule("angiopharm")); got != 1 {
		t.Fatalf("expected default C coefficient 1, got %v", got)
	}
}

func TestAdjustQuantityForBrandAppliesBrandRounding(t *testing.T) {
	christina := AdjustQuantityForBrand(11, "christina", "")
	if christina.Inserted == nil || *christina.Inserted != 12 || !christina.BoxAdjusted {
		t.Fatalf("expected Christina 11 to adjust to 12, got %#v", christina)
	}

	klapp := AdjustQuantityForBrand(10, "klapp", "")
	if klapp.Inserted == nil || *klapp.Inserted != 9 || !klapp.BoxAdjusted {
		t.Fatalf("expected KLAPP 10 to nearest multiple 9, got %#v", klapp)
	}
}

func TestCalculateAdjustedQuantity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		recommended  float64
		rule         RuleConfig
		box          string
		wantInsert   float64
		wantInsertOK bool
		wantComment  string
	}{
		{name: "small recommendation stays empty", recommended: 1.2, rule: Rule("angiopharm"), box: "3"},
		{name: "no adjustment inserts rounded", recommended: 7, rule: Rule("sothys"), wantInsert: 7, wantInsertOK: true},
		{name: "box without size inserts rounded", recommended: 10, rule: Rule("angiopharm"), wantInsert: 10, wantInsertOK: true},
		{name: "minimum lifts short order", recommended: 2, rule: RuleConfig{Adjustment: AdjustmentMinimum, AdjustmentComment: "мин"}, box: "6", wantInsert: 6, wantInsertOK: true, wantComment: "мин"},
		{name: "minimum already met", recommended: 8, rule: RuleConfig{Adjustment: AdjustmentMinimum}, box: "6", wantInsert: 8, wantInsertOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CalculateAdjustedQuantity(tt.recommended, tt.rule, tt.box)
			if tt.wantInsertOK {
				if got.Inserted == nil || *got.Inserted != tt.wantInsert {
					t.Fatalf("inserted = %v, want %v", got.Inserted, tt.wantInsert)
				}
			} else if got.Inserted != nil {
				t.Fatalf("inserted = %v, want nil", *got.Inserted)
			}
			if got.AutoComment != tt.wantComment {
				t.Fatalf("comment = %q, want %q", got.AutoComment, tt.wantComment)
			}
		})
	}
}

func TestCategoryCoefficientTable(t *testing.T) {
	t.Parallel()
	rule := Rule("angiopharm")
	tests := []struct {
		value string
		want  float64
	}{
		{value: "A+", want: 2},
		{value: "A", want: 1.75},
		{value: "B", want: 1.5},
		{value: "C", want: 1},
		{value: "Z", want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()
			if got := CategoryCoefficient(tt.value, rule); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyFromNomenclatureGroupMaps1CFilterNames(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Ангиофарм ":    "angiopharm",
		"ANGIOPHARM":    "angiopharm",
		"Кристина":      "christina",
		"CHRISTINA":     "christina",
		"KLAPP":         "klapp",
		"клапп":         "klapp",
		"SKIN SYNERGY":  "skin_synergy",
		"Skin Synergy":  "skin_synergy",
		"скин синерджи": "skin_synergy",
		"LeviSsime":     "levissime",
		"левисим":       "levissime",
		"SOTHYS":        "sothys",
		"сотис":         "sothys",
		"NOVACUTAN":     "novacutan",
		"новакутан":     "novacutan",
	}
	for group, want := range cases {
		got, ok := KeyFromNomenclatureGroup(group)
		if !ok || got != want {
			t.Fatalf("group %q: got (%q, %v), want %q", group, got, ok, want)
		}
	}
	if _, ok := KeyFromNomenclatureGroup("Неизвестный бренд"); ok {
		t.Fatal("unknown group must not map to a brand")
	}
}
