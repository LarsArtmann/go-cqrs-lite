package metaengine

import (
	"encoding/json/v2"
	"fmt"
	"strings"
)

// Inspect returns a human-readable summary of all collections, their ADTs,
// read patterns, assigned engines, and complexity scores. Useful for CLI
// tools, debug endpoints, and startup diagnostics.
func (s *Store) Inspect() string {
	collections := s.Collections()

	if len(collections) == 0 {
		return "metaengine: no collections registered"
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "metaengine: %d collection(s)\n", len(collections))

	for _, c := range collections {
		fmt.Fprintf(
			&sb,
			"  %-20s  ADT=%-10s  pattern=%-20s  engine=%-15s  complexity=%s\n",
			c.Name, c.ADT, c.ReadPattern, c.EngineName, c.Complexity,
		)
	}

	return sb.String()
}

// InspectJSON returns a machine-readable JSON summary of all collections,
// suitable for API endpoints, monitoring tools, and structured logging.
func (s *Store) InspectJSON() ([]byte, error) {
	return json.Marshal(s.Collections()) //nolint:wrapcheck
}
