package search

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Fold lower-cases text rune by rune, so index i of the folded text is index
// i of the original. A rune never maps to more than one rune, which
// strings.ToLower does not guarantee.
func Fold(text string) string {
	runes := []rune(text)
	for i, r := range runes {
		runes[i] = unicode.ToLower(r)
	}
	return string(runes)
}

// termScores are the multipliers of the in-memory ranking rule: exact beats
// prefix, prefix beats substring.
const (
	scoreExact     = 3
	scorePrefix    = 2
	scoreSubstring = 1
)

// fieldScore returns the best match of term against one folded field value.
func fieldScore(foldedField, foldedTerm string) float64 {
	switch {
	case foldedField == "":
		return 0
	case foldedField == foldedTerm:
		return scoreExact
	case hasWordPrefix(foldedField, foldedTerm):
		return scorePrefix
	case strings.Contains(foldedField, foldedTerm):
		return scoreSubstring
	default:
		return 0
	}
}

// hasWordPrefix reports whether the term starts a word of the field: at the
// start, or right after a character that is not a letter or digit.
// hasWordPrefix reports whether the term starts a word of the field: at the
// start, or right after a character that is not a letter or digit.
func hasWordPrefix(foldedField, foldedTerm string) bool {
	for from := 0; from < len(foldedField); {
		idx := strings.Index(foldedField[from:], foldedTerm)
		if idx < 0 {
			return false
		}
		at := from + idx
		if at == 0 {
			return true
		}
		before, _ := utf8.DecodeLastRuneInString(foldedField[:at])
		if !unicode.IsLetter(before) && !unicode.IsDigit(before) {
			return true
		}
		from = at + len(foldedTerm)
	}
	return false
}

// matchTerm returns the best weighted score of one term over every field, and
// the name of the field that scored best.
func matchTerm[T any](spec Spec[T], item T, foldedTerm string) (float64, string) {
	var (
		best      float64
		bestField string
	)
	for _, field := range spec.Fields {
		weighted := fieldScore(Fold(field.Value(item)), foldedTerm) * field.Weight
		if weighted > best {
			best = weighted
			bestField = field.Name
		}
	}
	return best, bestField
}

// Score applies the in-memory ranking rule to one record.
//
// A term matches when any field contains it case-folded. An exact field
// match scores three times the field weight, a term that starts a word of
// the field two times, and a substring one time. The record's score is the sum over terms of its best
// field score. Every term must match; otherwise Score returns false and 0.
//
// The terms come from [Tokens]; operator tokens are not matched against the
// text.
func Score[T any](spec Spec[T], req Request, item T) (float64, bool) {
	terms, _ := spec.Tokens(req.Query)
	return ScoreTerms(spec, item, terms)
}

// ScoreTerms is [Score] over terms the caller already split.
func ScoreTerms[T any](spec Spec[T], item T, terms []string) (float64, bool) {
	if len(terms) == 0 {
		return 0, false
	}
	total := 0.0
	for _, term := range terms {
		score, _ := matchTerm(spec, item, Fold(term))
		if score == 0 {
			return 0, false
		}
		total += score
	}
	return total, true
}

// IsIdentifierLike reports whether text is an identifier rather than prose:
// at least half of its characters are not letters and at least one is a
// digit, the shape of a phone number, a deed number or an order id.
// Tokenizers split such queries, so they are served lexically (Auto rule 2).
func IsIdentifierLike(text string) bool {
	var (
		letters int
		digits  int
		total   int
	)
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		total++
		switch {
		case unicode.IsDigit(r):
			digits++
		case unicode.IsLetter(r):
			letters++
		}
	}
	return total > 0 && digits > 0 && letters*2 < total
}

// RankHits sorts hits in place by score, highest first, and breaks ties by
// identifier so the order is stable across runs. It returns the same slice.
func RankHits[T any](hits []Hit[T]) []Hit[T] {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].ID < hits[j].ID
	})
	return hits
}

// isProseWorthEmbedding reports whether a query is prose that may mean more
// than its words: several terms, or one long token that is not an
// identifier. It is the predicate behind Auto rules 2 to 4.
func isProseWorthEmbedding(terms []string) bool {
	switch len(terms) {
	case 0:
		return false
	case 1:
		runes := []rune(terms[0])
		return len(runes) > shortTermRunes && !IsIdentifierLike(terms[0])
	default:
		return true
	}
}

// shortTermRunes is the length under which a single token is a name or an id,
// where exact matching beats meaning (Auto rule 3).
const shortTermRunes = 6
