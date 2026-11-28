package search

import (
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// ReservedKeys are the URL parameters that carry the request itself. Every
// other parameter is a facet and passes through untouched.
const (
	// ParamQuery is the name of the search text parameter.
	ParamQuery = "q"
	// ParamPage is the name of the page parameter.
	ParamPage = "page"
	// ParamPageSize is the name of the page size parameter.
	ParamPageSize = "page_size"
	// ParamSort is the name of the sort parameter.
	ParamSort = "sort"
	// ParamDir is the name of the sort direction parameter.
	ParamDir = "dir"
)

// Request is one search. The zero value is the unfiltered first page.
type Request struct {
	// Query is the free text, the "q" parameter. Empty means no text search.
	Query string
	// Filters are closed-set facets. They pass through to the [Source]
	// untouched; the search never interprets them.
	Filters url.Values
	// Sort is a key declared in [Spec.Sorts], or empty for relevance.
	Sort string
	// Page selects the page. A zero value means the first page of the
	// default size.
	Page query.Pagination
}

// Normalize trims the query, collapses runs of whitespace to one space, and
// rejects a query that is too long or carries control characters. The limit
// is the spec's; zero means [MaxQueryRunes].
func (r Request) Normalize(limit int) (Request, error) {
	limit, err := limitOrDefault(limit, MaxQueryRunes)
	if err != nil {
		return Request{}, err
	}

	normalized := r

	normalized.Query = collapseWhitespace(r.Query)
	if strings.ContainsFunc(normalized.Query, unicode.IsControl) {
		return Request{}, ErrQueryInvalid
	}

	if length := len([]rune(normalized.Query)); length > limit {
		return Request{}, rejectionWith(ErrQueryTooLong,
			"the search text is "+strconv.Itoa(length)+" characters; the limit is "+strconv.Itoa(limit))
	}

	return normalized, nil
}

// collapseWhitespace trims and collapses runs of whitespace, honouring
// double quotes so a quoted phrase keeps its inner spaces.
func collapseWhitespace(text string) string {
	var (
		out    strings.Builder
		quoted bool
		space  bool
	)

	for _, r := range text {
		switch {
		case r == '"':
			quoted = !quoted

			if space {
				out.WriteByte(' ')

				space = false
			}

			out.WriteRune(r)
		case unicode.IsSpace(r) && !quoted:
			space = out.Len() > 0
		default:
			if space {
				out.WriteByte(' ')

				space = false
			}

			out.WriteRune(r)
		}
	}

	return out.String()
}

// Tokenize splits a query into free terms and operator tokens.
//
// A word of the form key:value, where key is listed in keys, becomes an
// operator token; several values of one key are kept in order. Any other
// word stays a term, including key:value pairs whose key is not listed:
// unknown keys are free text, and what a key means is the product's business.
// Double quotes group words, so status:"in progress" is one token and
// "big invoice" is two terms held together as a phrase.
func Tokenize(query string, keys ...string) ([]string, map[string][]string) {
	var (
		terms  []string
		tokens map[string][]string
	)

	known := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		known[strings.ToLower(key)] = struct{}{}
	}

	for _, word := range splitWords(query) {
		key, value, ok := cutOperator(word)
		if ok {
			if _, isKnown := known[key]; isKnown {
				if tokens == nil {
					tokens = make(map[string][]string)
				}

				tokens[key] = append(tokens[key], value)

				continue
			}
		}

		terms = append(terms, word)
	}

	return terms, tokens
}

// splitWords splits on whitespace outside double quotes and drops the quotes.
func splitWords(query string) []string {
	var (
		words   []string
		current strings.Builder
		quoted  bool
		active  bool
	)

	for _, r := range query {
		switch {
		case r == '"':
			quoted = !quoted
			active = true
		case unicode.IsSpace(r) && !quoted:
			if active {
				words = append(words, current.String())
			}

			current.Reset()

			active = false
		default:
			current.WriteRune(r)

			active = true
		}
	}

	if active {
		words = append(words, current.String())
	}

	return words
}

// cutOperator splits key:value and reports whether the word is a well-formed
// operator. A malformed operator demotes to a term.
func cutOperator(word string) (string, string, bool) {
	key, value, found := strings.Cut(word, ":")
	if !found || value == "" {
		return "", "", false
	}

	key = strings.ToLower(key)
	if key == "" {
		return "", "", false
	}

	for _, r := range key {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return "", "", false
		}
	}

	return key, value, true
}
