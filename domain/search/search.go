// Package search provides fuzzy, case-insensitive search over Recent Folders
// entries, matching the folder name and full path.
package search

import (
	"sort"
	"strings"

	"cwdgo/domain/recentfolders"
)

// Search returns the entries matching query, ranked best first. Matching is
// case-insensitive and fuzzy: the query characters must appear in order in
// the folder name or full path, gaps allowed. Name matches rank above
// path-only matches; within a domain, exact > prefix > substring > fuzzy;
// then earlier start, fewer gaps, shorter target. Ties keep input order.
// An empty query returns all entries in input order.
func Search(entries []recentfolders.Entry, query string) []recentfolders.Entry {
	if query == "" {
		return append([]recentfolders.Entry(nil), entries...)
	}
	q := strings.ToLower(query)

	type scored struct {
		entry recentfolders.Entry
		score score
	}
	matched := make([]scored, 0, len(entries))
	for _, e := range entries {
		if s, ok := bestScore(e, q); ok {
			matched = append(matched, scored{e, s})
		}
	}
	sort.SliceStable(matched, func(i, j int) bool {
		return less(matched[i].score, matched[j].score)
	})
	out := make([]recentfolders.Entry, len(matched))
	for i, m := range matched {
		out[i] = m.entry
	}
	return out
}

// Filter returns the entries matching query, preserving input order (no
// ranking). Matching uses the same case-insensitive fuzzy logic as Search,
// but the relative order of the input is kept — the panel shows Recent
// Folders newest-first and must not reshuffle recorded items when several
// match. An empty query returns all entries in input order.
func Filter(entries []recentfolders.Entry, query string) []recentfolders.Entry {
	if query == "" {
		return append([]recentfolders.Entry(nil), entries...)
	}
	q := strings.ToLower(query)
	out := make([]recentfolders.Entry, 0, len(entries))
	for _, e := range entries {
		if _, ok := bestScore(e, q); ok {
			out = append(out, e)
		}
	}
	return out
}

// matchKind classifies how the query is embedded in a target string.
type matchKind int

const (
	kindExact matchKind = iota
	kindPrefix
	kindSubstring
	kindFuzzy
)

// score orders matches; lower is better.
type score struct {
	domain int // 0 = name, 1 = path
	kind   matchKind
	start  int // rune index where the match starts
	gaps   int // runes skipped inside the match
	length int // target length in runes
}

func less(a, b score) bool {
	if a.domain != b.domain {
		return a.domain < b.domain
	}
	if a.kind != b.kind {
		return a.kind < b.kind
	}
	if a.start != b.start {
		return a.start < b.start
	}
	if a.gaps != b.gaps {
		return a.gaps < b.gaps
	}
	return a.length < b.length
}

// bestScore returns the best match of the folded query q against the entry,
// preferring the folder name over the full path.
func bestScore(e recentfolders.Entry, q string) (score, bool) {
	if kind, start, gaps, length, ok := match(e.Name(), q); ok {
		return score{domain: 0, kind: kind, start: start, gaps: gaps, length: length}, true
	}
	kind, start, gaps, length, ok := match(e.Path, q)
	return score{domain: 1, kind: kind, start: start, gaps: gaps, length: length}, ok
}

// match scores how well the folded query q matches target (case-insensitive).
func match(target, q string) (kind matchKind, start, gaps, length int, ok bool) {
	t := strings.ToLower(target)
	tr := []rune(t)
	qr := []rune(q)
	if len(qr) > len(tr) {
		return 0, 0, 0, 0, false
	}
	if t == q {
		return kindExact, 0, 0, len(tr), true
	}
	start, gaps, ok = bestEmbedding(tr, qr)
	if !ok {
		return 0, 0, 0, 0, false
	}
	kind = kindFuzzy
	if start == 0 && gaps == 0 {
		kind = kindPrefix
	} else if gaps == 0 {
		kind = kindSubstring
	}
	return kind, start, gaps, len(tr), true
}

// bestEmbedding finds the earliest embedding of qr in tr: the smallest start
// index, and for that start the fewest skipped runes (greedy matching is
// optimal for a fixed start).
func bestEmbedding(tr, qr []rune) (start, gaps int, ok bool) {
	for s := 0; s+len(qr) <= len(tr); s++ {
		if tr[s] != qr[0] {
			continue
		}
		pos, g := s, 0
		valid := true
		for i := 1; i < len(qr); i++ {
			next := -1
			for p := pos + 1; p < len(tr); p++ {
				if tr[p] == qr[i] {
					next = p
					break
				}
			}
			if next < 0 {
				valid = false
				break
			}
			g += next - pos - 1
			pos = next
		}
		if valid {
			return s, g, true
		}
	}
	return 0, 0, false
}
