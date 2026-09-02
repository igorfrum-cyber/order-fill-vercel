package identity

import "testing"

func TestLoginSlugFromNameLatin(t *testing.T) {
	got := LoginSlugFromName("Acme Corp!", "id-ignored")
	if got != "acme-corp" {
		t.Fatalf("got %q", got)
	}
}

func TestLoginSlugFromNameCyrillic(t *testing.T) {
	got := LoginSlugFromName("ООО Ромашка", "id-ignored")
	if got != "ooo-romashka" {
		t.Fatalf("got %q", got)
	}
}

func TestLoginSlugFromNameFallbackToCompanyID(t *testing.T) {
	got := LoginSlugFromName("!!!", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if got != "company-aaaaaaaa" {
		t.Fatalf("got %q", got)
	}
}

func TestUniqueLoginSlugAppendsIDWhenTaken(t *testing.T) {
	taken := map[string]struct{}{"acme": {}}
	got := UniqueLoginSlug("Acme", "22222222-2222-2222-2222-222222222222", func(slug string) bool {
		_, ok := taken[slug]
		return ok
	})
	if got != "acme-22222222" {
		t.Fatalf("got %q", got)
	}
}
