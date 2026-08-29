package matching

import (
	"math"
	"testing"
)

func TestSimilarityMatchesTheBrowserEngineForAngiopharmPrefix(t *testing.T) {
	got := Similarity("Косметичка непромокаемая", "АН Косметичка непромокаемая")
	if math.Abs(got-0.9411764705882353) > 1e-9 {
		t.Fatalf("expected 94%% LCS against an unstripped АН prefix, got %v", got)
	}
}

func TestSimilarityKeepsTesterLabelScore(t *testing.T) {
	got := Similarity(`Этикетка "ТЕСТЕР"`, `АН Этикетка "ТЕСТЕР", 500 шт/рул`)
	if math.Abs(got-0.6818181818181818) > 1e-9 {
		t.Fatalf("expected 68%% LCS for the tester label, got %v", got)
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
