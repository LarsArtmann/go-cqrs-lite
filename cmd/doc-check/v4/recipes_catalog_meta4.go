package main

// Recipe catalog, part B4: recipes.md §2.48–§2.50 (the tail of section 2,
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
	"### 2.50 Per-Group Caps — keep the newest N rows PER owner (`WithCapPerGroup`, ADR-0164) #1": {
		imports: []string{
			`"context"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/record/v4"`,
		},
		preamble: "type TranscriptRecorded struct {\n" +
			"\tID, Owner string\n" +
			"\tStartedAt int64\n" +
			"}\n" +
			"type TranscriptView struct {\n" +
			"\tID, Owner string\n" +
			"\tStartedAt int64\n" +
			"}\n" +
			"var eng metaengine.Engine\n" +
			"var ctx context.Context\n",
		errFunc:  true,
		trailers: "_ = removed\n_ = err",
	},
	"### 2.51 The Deployment Ladder — same preset from Go, CUE, or NixOS (`system.Standard`, ADR-0165) #1": {
		imports: []string{
			`"context"`,
			`"fmt"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
		},
		preamble: "var ctx = context.Background()\n" +
			"var domain system.DomainConfig\n",
		trailers: "_ = sys\n_ = err",
	},
}
