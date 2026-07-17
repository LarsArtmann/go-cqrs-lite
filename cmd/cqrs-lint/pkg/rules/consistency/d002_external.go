package consistency

import (
	"go/ast"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// externalAPIMarker is the in-source opt-out comment for D002. Place it on a
// struct's doc comment to tell cqrs-lint "this struct's JSON tags mirror an
// external API; don't count them toward the mixed-casing check."
//
//	//cqrs-lint:external-api
//	type DiscordMessage struct {
//	    Content string `json:"content"`
//	    GuildID string `json:"guild_id"`
//	}
const externalAPIMarker = "cqrs-lint:external-api"

// collectExternalAPIStructs returns the set of StructType nodes the consumer
// has marked as mirroring an external API (Discord/Stripe/GitHub). Such
// structs are excluded from D002's mixed-casing check because their snake_case
// JSON tags are dictated by the upstream API and are not a local style choice.
//
// A struct is "external" when EITHER:
//   - its name starts with one of the configured
//     RulesConfig.ExternalAPIStructPrefixes (e.g. "Discord", "Stripe"), OR
//   - it carries the //cqrs-lint:external-api marker in its doc comment — on
//     either the enclosing GenDecl (single-spec `type Foo struct{}`) or the
//     TypeSpec itself (grouped `type ( ... )` blocks).
//
// The two mechanisms stack: a struct is excluded if either matches.
func collectExternalAPIStructs(
	fileAST *ast.File,
	cfg analyzer.RulesConfig,
) map[*ast.StructType]bool {
	external := make(map[*ast.StructType]bool)
	if len(cfg.ExternalAPIStructPrefixes) == 0 && !fileContainsMarker(fileAST) {
		return external
	}

	ast.Inspect(fileAST, func(n ast.Node) bool {
		gd, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		genDeclDoc := ""
		if gd.Doc != nil {
			genDeclDoc = gd.Doc.Text()
		}

		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			specDoc := ""
			if ts.Doc != nil {
				specDoc = ts.Doc.Text()
			}

			if isExternalAPIStruct(ts.Name.Name, genDeclDoc, specDoc, cfg) {
				external[st] = true
			}
		}

		return true
	})

	return external
}

// fileContainsMarker is a cheap pre-filter: if no prefixes are configured AND
// the file's text carries no //cqrs-lint:external-api comment, there is nothing
// to collect and we skip the full GenDecl walk.
func fileContainsMarker(fileAST *ast.File) bool {
	for _, group := range fileAST.Comments {
		for _, c := range group.List {
			if strings.Contains(c.Text, externalAPIMarker) {
				return true
			}
		}
	}

	return false
}

// isExternalAPIStruct reports whether a struct should be excluded from D002
// based on its name, its doc comment text, and the configured prefix list.
func isExternalAPIStruct(name, genDeclDoc, specDoc string, cfg analyzer.RulesConfig) bool {
	for _, prefix := range cfg.ExternalAPIStructPrefixes {
		if prefix == "" {
			continue
		}

		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return strings.Contains(genDeclDoc, externalAPIMarker) ||
		strings.Contains(specDoc, externalAPIMarker)
}
