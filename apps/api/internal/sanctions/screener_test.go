package sanctions

import (
	"testing"
)

func uaeTestEntries() []WatchlistEntry {
	return []WatchlistEntry{
		{ListName: "UAE_TFS", ListVersion: "2026-06-01", EntryID: "UAE-001",
			Names: []string{"Mohammed Al Rashidi", "محمد الراشدي", "Mohamed Al Rashidy"}},
		{ListName: "UN_SC", ListVersion: "2026-06-01", EntryID: "UN-002",
			Names: []string{"Abdullah bin Laden", "Osama bin Laden"}},
		{ListName: "OFAC_SDN", ListVersion: "2026-06-01", EntryID: "OFAC-003",
			Names: []string{"Iranian National Bank", "Bank Melli Iran"}},
	}
}

func saTestEntries() []WatchlistEntry {
	return []WatchlistEntry{
		{ListName: "SAMA_WATCHLIST", ListVersion: "2026-06-01", EntryID: "SA-001",
			Names: []string{"Khalid Al Ghamdi", "خالد الغامدي"}},
		{ListName: "UN_SC", ListVersion: "2026-06-01", EntryID: "UN-003",
			Names: []string{"Al Qaeda Finance Committee"}},
	}
}

func TestScreener_ExactMatch(t *testing.T) {
	s := NewScreener(uaeTestEntries())
	results := s.Screen("Mohammed Al Rashidi")
	if len(results) == 0 {
		t.Fatal("expected at least one match for exact name")
	}
	if results[0].Status != MatchConfirmed {
		t.Errorf("expected MatchConfirmed for exact name, got %s", results[0].Status)
	}
	if results[0].Score < 0.99 {
		t.Errorf("expected score >= 0.99 for exact match, got %.3f", results[0].Score)
	}
}

func TestScreener_FuzzyMatch_ArabicTransliteration(t *testing.T) {
	s := NewScreener(uaeTestEntries())
	// Common transliteration variant
	results := s.Screen("Mohamed Al Rashidy")
	if len(results) == 0 {
		t.Fatal("expected fuzzy match for Arabic name transliteration variant")
	}
	if results[0].Score < 0.75 {
		t.Errorf("expected score >= 0.75 for transliteration variant, got %.3f", results[0].Score)
	}
}

func TestScreener_NoMatch_UnrelatedName(t *testing.T) {
	s := NewScreener(uaeTestEntries())
	results := s.Screen("John Smith")
	for _, r := range results {
		if r.Status == MatchConfirmed {
			t.Errorf("unexpected confirmed hit for unrelated name: %s score %.3f", r.MatchedName, r.Score)
		}
	}
}

func TestScreener_PotentialHit_SimilarName(t *testing.T) {
	s := NewScreener(uaeTestEntries())
	results := s.Screen("Mohd Al Rashidi")
	found := false
	for _, r := range results {
		if r.Score >= 0.75 {
			found = true
		}
	}
	if !found {
		t.Error("expected potential hit for similar Arabic name")
	}
}

func TestScreener_SaudiSAMAWatchlist(t *testing.T) {
	s := NewScreener(saTestEntries())
	results := s.Screen("Khalid Al Ghamdi")
	if len(results) == 0 {
		t.Fatal("expected match on SAMA_WATCHLIST")
	}
	if results[0].ListName != "SAMA_WATCHLIST" {
		t.Errorf("expected SAMA_WATCHLIST list, got %s", results[0].ListName)
	}
}

func TestScreener_EmptyName(t *testing.T) {
	s := NewScreener(uaeTestEntries())
	results := s.Screen("")
	// empty name should return no confirmed hits
	for _, r := range results {
		if r.Status == MatchConfirmed {
			t.Errorf("empty name should not produce confirmed hit")
		}
	}
}

func TestScreener_NormalizeName_ArabicDiacritics(t *testing.T) {
	// Arabic name with tashkeel (diacritics) should normalize to same as without
	withDiacritics := "مُحَمَّد"
	withoutDiacritics := "محمد"
	n1 := normalizeName(withDiacritics)
	n2 := normalizeName(withoutDiacritics)
	if n1 != n2 {
		t.Errorf("diacritics normalization failed: %q != %q", n1, n2)
	}
}

func TestScreener_NormalizeName_CaseAndPunctuation(t *testing.T) {
	n := normalizeName("AL-RASHIDI, Mohammed.")
	expected := "al rashidi mohammed"
	if n != expected {
		t.Errorf("expected %q, got %q", expected, n)
	}
}

func TestJaroWinkler_IdenticalStrings(t *testing.T) {
	score := jaroWinkler("mohammed", "mohammed")
	if score != 1.0 {
		t.Errorf("expected 1.0 for identical strings, got %.3f", score)
	}
}

func TestJaroWinkler_EmptyStrings(t *testing.T) {
	score := jaroWinkler("", "")
	if score != 1.0 {
		t.Errorf("expected 1.0 for two empty strings, got %.3f", score)
	}
	score2 := jaroWinkler("", "name")
	if score2 != 0.0 {
		t.Errorf("expected 0.0 for empty vs non-empty, got %.3f", score2)
	}
}

func TestJaroWinkler_PrefixBonus(t *testing.T) {
	// Pair with 4-char shared prefix gets full prefix bonus
	withPrefix := jaroWinkler("abcd rashidi", "abcd rasheedi")
	// Pair with zero shared prefix gets no bonus
	noPrefix := jaroWinkler("xbcd rashidi", "abcd rasheedi")
	if withPrefix <= noPrefix {
		t.Errorf("expected shared-prefix pair to score higher: %.4f <= %.4f", withPrefix, noPrefix)
	}
}
