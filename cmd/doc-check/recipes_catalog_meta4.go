package main

// Recipe catalog, part B4: recipes.md §2.48–§2.49 (the tail of section 2,
// split out of part B3 to keep files under the 350-line file cap).

var recipeCatalogB4 = map[string]recipeSpec{
	"### 2.48 Publish a Self-Contained Architecture Reference (catalog/architecture) #1": {
		imports: []string{
			`"log"`,
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"`,
		},
		preamble: "type OrderPlaced struct{ OrderID string }\n",
	},
	"### 2.49 Pin Your Engine's Stored Bytes (sqliteengine / pebbleengine / bboltengine / badgerengine) #1": {
		imports: []string{
			`"context"`,
			`"database/sql"`,
			`"strings"`,
			`"testing"`,
			`_ "modernc.org/sqlite"`,
			`sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"`,
			`metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
		},
	},
}
