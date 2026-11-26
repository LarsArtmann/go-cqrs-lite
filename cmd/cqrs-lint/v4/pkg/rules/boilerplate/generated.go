package boilerplate

import (
	"go/ast"
	"regexp"
)

var generatedHeader = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// isGenerated: a finding inside a generated file can never be fixed in place,
// it only points away from the generator that owns the code.
func isGenerated(file *ast.File) bool {
	for _, group := range file.Comments {
		if group.Pos() > file.Package {
			break
		}

		for _, comment := range group.List {
			if generatedHeader.MatchString(comment.Text) {
				return true
			}
		}
	}

	return false
}
