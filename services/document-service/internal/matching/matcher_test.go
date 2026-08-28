package matching

import "testing"

func TestSimilarityIgnoresAngiopharmNoise(t *testing.T) {
	if got := Similarity("АН Cream 50 ml", "Angiopharm cream 50 ml"); got != 1 {
		t.Fatalf("expected perfect normalized similarity, got %v", got)
	}
}

func TestChooseCandidatePrefersMatchingVolume(t *testing.T) {
	candidate, ok := ChooseCandidate([]Item{
		{Article: "A1", Name: "Cream 30 ml", Rounded: 5},
		{Article: "A2", Name: "Cream 50 ml", Rounded: 5},
	}, "Cream 50 ml", "")
	if !ok || candidate.Item.Article != "A2" {
		t.Fatalf("expected 50 ml candidate, got %#v ok=%v", candidate, ok)
	}
}
