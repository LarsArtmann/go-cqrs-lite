package search

import (
	"strconv"
	"strings"
	"unicode"
)

// EscapeLike escapes the LIKE wildcards of a literal so that, inside a
// "... LIKE ? ESCAPE '\'" clause, the value matches only itself.
func EscapeLike(literal string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(literal)
}

// LikePattern wraps a term for a contains match. Use it only in a query with
// the ESCAPE '\' clause, and with LOWER on both sides; LIKE folds case for
// ASCII only.
func LikePattern(term string) string {
	return "%" + EscapeLike(term) + "%"
}

// fts5MaxQueryRunes is the longest query FTS5Query quotes, the cap
// pbx-lens and InboxClean settled on.
const fts5MaxQueryRunes = 256

// FTS5Query turns free text into a safe FTS5 MATCH expression: every word
// is quoted with embedded quotes doubled, words are joined with AND, and
// with phrase true they form one quoted phrase instead.
//
// Operator tokens are not understood here; pass the free terms. Control
// characters and over-long input are rejected, never quoted into the query.
func FTS5Query(query string, phrase bool) (string, error) {
	query = strings.TrimSpace(collapseWhitespace(query))
	if query == "" {
		return "", nil
	}

	if strings.ContainsFunc(query, unicode.IsControl) {
		return "", ErrQueryInvalid
	}

	if length := len([]rune(query)); length > fts5MaxQueryRunes {
		return "", rejectionWith(ErrQueryTooLong,
			"the search text is "+strconv.Itoa(length)+" characters; the limit is "+strconv.Itoa(fts5MaxQueryRunes))
	}

	if phrase {
		return fts5Quote(query), nil
	}

	words := strings.Fields(query)
	for i, word := range words {
		words[i] = fts5Quote(word)
	}

	return strings.Join(words, " AND "), nil
}

// fts5Quote doubles embedded quotes and wraps the text in double quotes.
func fts5Quote(text string) string {
	return `"` + strings.ReplaceAll(text, `"`, `""`) + `"`
}
