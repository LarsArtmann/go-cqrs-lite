package querytest

import "github.com/larsartmann/go-cqrs-lite/query/v2"

// must unwraps a (value, error) pair, panicking on error.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

// MustNew constructs a *query.BasicQuery from queryType, panicking on invalid input.
func MustNew(queryType query.Type) *query.BasicQuery { return must(query.New(queryType)) }
