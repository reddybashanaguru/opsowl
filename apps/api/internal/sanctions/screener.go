// Package sanctions — Arabic/English name screener using Jaro-Winkler similarity.
// Handles Arabic diacritics (tashkeel), transliteration variants, and mixed-script names.
// Used for CBUAE TFS screening (UAE) and SAMA watchlist screening (Saudi Arabia).
package sanctions

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// MatchResult is the outcome of screening a single name against a watchlist entry.
type MatchResult struct {
	ListName    string      // "UAE_TFS", "UN_SC", "OFAC_SDN", "SAMA_WATCHLIST"
	ListVersion string      // date of the list snapshot used
	EntryID     string      // identifier in the source list
	MatchedName string      // the name as it appears in the list
	Score       float64     // 0–1; 1 = exact
	Status      MatchStatus
}

// MatchStatus classifies a screening result.
type MatchStatus string

const (
	MatchClean     MatchStatus = "clean"
	MatchPotential MatchStatus = "potential_hit" // score 0.75–0.98 — manual review required
	MatchConfirmed MatchStatus = "confirmed_hit"  // exact or score ≥ 0.99
)

// WatchlistEntry is a single name (with aliases) on any regional watchlist.
type WatchlistEntry struct {
	ListName    string
	ListVersion string
	EntryID     string
	Names       []string // primary name + all known aliases
}

// Screener holds loaded watchlist entries and performs name matching.
// In production, entries are refreshed daily via the sanctions loader.
type Screener struct {
	entries []WatchlistEntry
}

// NewScreener builds a Screener from a set of watchlist entries.
func NewScreener(entries []WatchlistEntry) *Screener {
	return &Screener{entries: entries}
}

// Screen checks the given name against all loaded watchlists.
// Returns all potential and confirmed matches; empty slice means clean.
func (s *Screener) Screen(name string) []MatchResult {
	query := normalizeName(name)
	var results []MatchResult

	for _, entry := range s.entries {
		best := 0.0
		bestName := ""
		for _, candidate := range entry.Names {
			score := jaroWinkler(query, normalizeName(candidate))
			if score > best {
				best = score
				bestName = candidate
			}
		}
		if best < 0.75 {
			continue
		}
		status := MatchPotential
		if best >= 0.99 {
			status = MatchConfirmed
		}
		results = append(results, MatchResult{
			ListName:    entry.ListName,
			ListVersion: entry.ListVersion,
			EntryID:     entry.EntryID,
			MatchedName: bestName,
			Score:       best,
			Status:      status,
		})
	}
	return results
}

// normalizeName converts a name to canonical form for comparison:
//   - NFD decompose then strip non-spacing marks (removes Arabic tashkeel / diacritics)
//   - recompose to NFC
//   - lowercase, strip punctuation, collapse whitespace
func normalizeName(s string) string {
	// Decompose and strip non-spacing marks (Unicode Mn category = diacritics, harakat, etc.)
	decomposed := norm.NFD.String(s)
	var stripped strings.Builder
	for _, r := range decomposed {
		if !unicode.Is(unicode.Mn, r) {
			stripped.WriteRune(r)
		}
	}
	s = norm.NFC.String(stripped.String())

	var b strings.Builder
	prevSpace := true
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSpace = false
		} else if !prevSpace {
			b.WriteRune(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// jaroWinkler computes the Jaro-Winkler similarity between two strings.
// Returns [0, 1] where 1 = identical. Prefix bonus (p=0.1) is applied for
// shared leading characters — important for Arabic name prefix variants (Al-, El-, etc.).
func jaroWinkler(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	jaro := jaroSimilarity(s1, s2)
	prefix := 0
	for i := 0; i < len(s1) && i < len(s2) && i < 4; i++ {
		if s1[i] == s2[i] {
			prefix++
		} else {
			break
		}
	}
	return jaro + float64(prefix)*0.1*(1.0-jaro)
}

func jaroSimilarity(s1, s2 string) float64 {
	if len(s1) == 0 && len(s2) == 0 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}
	matchDist := maxInt(len(s1), len(s2))/2 - 1
	if matchDist < 0 {
		matchDist = 0
	}
	s1m := make([]bool, len(s1))
	s2m := make([]bool, len(s2))
	matches := 0
	transpositions := 0

	for i, c1 := range s1 {
		lo := i - matchDist
		if lo < 0 {
			lo = 0
		}
		hi := i + matchDist + 1
		if hi > len(s2) {
			hi = len(s2)
		}
		for j := lo; j < hi; j++ {
			if s2m[j] || rune(s2[j]) != c1 {
				continue
			}
			s1m[i] = true
			s2m[j] = true
			matches++
			break
		}
	}
	if matches == 0 {
		return 0.0
	}
	k := 0
	for i, m := range s1m {
		if !m {
			continue
		}
		for !s2m[k] {
			k++
		}
		if s1[i] != s2[k] {
			transpositions++
		}
		k++
	}
	return (float64(matches)/float64(len(s1)) +
		float64(matches)/float64(len(s2)) +
		float64(matches-transpositions/2)/float64(matches)) / 3.0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
