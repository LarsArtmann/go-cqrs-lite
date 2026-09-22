package system

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// buildCRUDQuery generates a CRUD query declaration from named event
// samples. Extra options (FilterOnField, SortOnField, ...) are appended
// after the folds; layout priority is appended last.
func buildCRUDQuery[Q any, R any](
	name, keyField string,
	samples []metaengine.NamedSample,
	opts []any,
	layoutPriority metaengine.Priority,
) (any, []decoderEntry, error) {
	folds, err := metaengine.AutoCRUDByNamedEvents[R](keyField, samples...)
	if err != nil {
		return nil, nil, fmt.Errorf("system: projection %q: %w", name, err)
	}

	args := make([]any, 0, len(folds)+len(opts)+1)
	for _, f := range folds {
		args = append(args, f)
	}

	args = append(args, opts...)

	if layoutPriority.Valid() {
		args = append(args, metaengine.WithLayoutPriority(layoutPriority))
	}

	query := metaengine.Query[Q, R](name, args...)

	return query, samplesToDecoderEntries(samples), nil
}

// buildCounterQuery generates a counter query declaration from count entries.
func buildCounterQuery(
	name string,
	entries []countEntry,
	layoutPriority metaengine.Priority,
) (any, []decoderEntry) {
	folds := make([]metaengine.Fold, 0, len(entries))
	decEntries := make([]decoderEntry, 0, len(entries))

	for _, e := range entries {
		delta := e.delta
		key := e.key

		fold := metaengine.OnRecordTyped(
			e.eventType,
			e.sample,
			func(_ record.Record, _ any) metaengine.Delta {
				return metaengine.Delta{key: delta}
			},
		)
		folds = append(folds, fold)

		decEntries = append(decEntries, decoderEntry{
			eventType: e.eventType,
			sample:    e.sample,
		})
	}

	args := make([]any, 0, len(folds)+1)
	for _, f := range folds {
		args = append(args, f)
	}

	if layoutPriority.Valid() {
		args = append(args, metaengine.WithLayoutPriority(layoutPriority))
	}

	query := metaengine.Query[CountInput, map[string]int64](name, args...)

	return query, decEntries
}

func samplesToDecoderEntries(samples []metaengine.NamedSample) []decoderEntry {
	entries := make([]decoderEntry, len(samples))
	for i, s := range samples {
		entries[i] = decoderEntry{
			eventType: s.EventType(),
			sample:    s.Sample(),
		}
	}

	return entries
}
