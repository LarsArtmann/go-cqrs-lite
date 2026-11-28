package search

import (
	"encoding/json"
	"fmt"
	"os"

	errorfamily "github.com/larsartmann/go-error-family"
)

// StaticIndexVersion is the format version written into every static index.
const StaticIndexVersion = 1

// StaticIndex is a search index for the browser: compact JSON, no vectors,
// the same fields and weights the server declares. A page loads or inlines
// it and the client matcher scores it with the in-memory ranking rule.
type StaticIndex struct {
	Version int           `json:"version"`
	Fields  []StaticField `json:"fields"`
	Items   []StaticItem  `json:"items"`
}

// StaticField is one field of a [StaticIndex].
type StaticField struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

// StaticItem is one record of a [StaticIndex]. Fields holds the non-empty
// values by field name.
type StaticItem struct {
	ID     string            `json:"id"`
	Fields map[string]string `json:"fields"`
}

// BuildStaticIndex writes the static index of items. Private fields are left
// out: they never leave the machine. Only non-empty values are emitted.
func BuildStaticIndex[T any](spec Spec[T], items []T, identify func(T) string) (*StaticIndex, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	if identify == nil {
		return nil, errorfamily.NewRejection("search.no_id",
			"a static index needs the function that returns a record's identifier")
	}

	index := &StaticIndex{Version: StaticIndexVersion, Fields: nil, Items: make([]StaticItem, 0, len(items))}

	for _, field := range spec.Fields {
		if field.Private {
			continue
		}

		index.Fields = append(index.Fields, StaticField{Name: field.Name, Weight: field.resolvedWeight()})
	}

	for _, item := range items {
		values := make(map[string]string, len(index.Fields))
		for _, field := range spec.Fields {
			if field.Private {
				continue
			}

			if value := field.Value(item); value != "" {
				values[field.Name] = value
			}
		}

		index.Items = append(index.Items, StaticItem{ID: identify(item), Fields: values})
	}

	return index, nil
}

// WriteFile writes the index as indented JSON to path, replacing the file.
func (ix *StaticIndex) WriteFile(path string) error {
	if ix == nil {
		return errorfamily.NewRejection("search.nil_index", "there is no index to write; build one first")
	}

	encoded, err := json.MarshalIndent(ix, "", "  ")
	if err != nil {
		return errorfamily.Wrap(err, errorfamily.Corruption, "search.static_encode",
			"the static index could not be encoded; nothing was written")
	}

	//nolint:gosec // a static index is a public asset
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		return errorfamily.Wrap(err, errorfamily.Infrastructure, "search.static_write",
			fmt.Sprintf("the static index could not be written to %s; the previous file is unchanged", path))
	}

	return nil
}

// LoadStaticIndex reads a static index written by [StaticIndex.WriteFile].
func LoadStaticIndex(path string) (*StaticIndex, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, errorfamily.Wrap(err, errorfamily.Infrastructure, "search.static_read",
			"the static index could not be read from "+path)
	}

	var index StaticIndex
	if err := json.Unmarshal(encoded, &index); err != nil {
		return nil, errorfamily.Wrap(err, errorfamily.Corruption, "search.static_decode",
			"the static index is not valid JSON; rebuild it")
	}

	if index.Version != StaticIndexVersion {
		return nil, errorfamily.NewRejection("search.static_version",
			fmt.Sprintf("the static index has format version %d; this release reads version %d. Rebuild it",
				index.Version, StaticIndexVersion))
	}

	return &index, nil
}
