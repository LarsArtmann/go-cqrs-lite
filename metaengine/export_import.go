package metaengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
)

// Export serializes all collection data to a writer as JSON. Each collection
// is exported as a JSON array of key-value pairs.
func (s *Store) Export(ctx context.Context, w io.Writer) error {
	collections := s.Collections()

	if _, err := fmt.Fprint(w, "{"); err != nil {
		return err //nolint:wrapcheck
	}

	for i, col := range collections {
		if i > 0 {
			if _, err := fmt.Fprint(w, ","); err != nil {
				return err //nolint:wrapcheck
			}
		}

		if _, err := fmt.Fprintf(w, "%q:[", col.Name); err != nil {
			return err //nolint:wrapcheck
		}

		eng, ok := s.collectionEngine(col.Name)
		if !ok {
			if _, err := fmt.Fprint(w, "]"); err != nil {
				return err //nolint:wrapcheck
			}

			continue
		}

		if sb, ok := eng.(ScanBackend); ok {
			result, err := sb.MapScan(ctx, col.Name, nil, nil, nil, 0)
			if err != nil {
				return fmt.Errorf("export %s: %w", col.Name, err)
			}

			for j, row := range result.Items {
				if j > 0 {
					if _, err := fmt.Fprint(w, ","); err != nil {
						return err //nolint:wrapcheck
					}
				}

				data, err := json.Marshal(row)
				if err != nil {
					return fmt.Errorf("export %s row %d: %w", col.Name, j, err)
				}

				if _, err := w.Write(data); err != nil {
					return fmt.Errorf("export %s row %d: %w", col.Name, j, err)
				}
			}
		}

		if _, err := fmt.Fprint(w, "]"); err != nil {
			return err //nolint:wrapcheck
		}
	}

	if _, err := fmt.Fprint(w, "}"); err != nil {
		return err //nolint:wrapcheck
	}

	return nil
}

// Import loads collection data from a JSON reader (previously created by Export).
// Events are replayed into the store via Apply, rebuilding projections.
//
//	err := store.Import(ctx, os.Stdin)
func (s *Store) Import(ctx context.Context, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("metaengine.Import: read: %w", err)
	}

	var collections map[string][]any
	if err := json.Unmarshal(data, &collections); err != nil {
		return fmt.Errorf("metaengine.Import: unmarshal: %w", err)
	}

	for colName, rows := range collections {
		eng, ok := s.collectionEngine(colName)
		if !ok {
			continue // unknown collection, skip
		}

		mb, ok := eng.(MapBackend)
		if !ok {
			continue
		}

		for i, row := range rows {
			rowMap, ok := row.(map[string]any)
			if !ok {
				continue
			}

			key := extractKeyFromMap(rowMap)
			if key == nil {
				continue
			}

			if err := mb.MapSet(ctx, colName, key, row); err != nil {
				return fmt.Errorf("metaengine.Import: set %s[%d]: %w", colName, i, err)
			}
		}
	}

	return nil
}

// extractKeyFromMap tries common key field names.
func extractKeyFromMap(m map[string]any) any {
	for _, key := range []string{"id", "ID", "Id", "key", "Key"} {
		if v, ok := m[key]; ok {
			return v
		}
	}

	return nil
}
