package analyzer

import (
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

// RelPos renders pos as "path:line:column" with the path relative to the
// analyzed ProjectRoot when possible. Findings must be machine-independent:
// an absolute path bakes the analyzing host's directory layout into messages,
// goldens, and diffs (the taskmanager golden only matched on the machine that
// generated it). Falls back to pos.String() when the position escapes the
// root or relativization fails.
func (ctx *AnalysisContext) RelPos(pos token.Position) string {
	if ctx.ProjectRoot == "" || pos.Filename == "" {
		return pos.String()
	}

	rel, err := filepath.Rel(ctx.ProjectRoot, pos.Filename)
	if err != nil || strings.HasPrefix(rel, "..") {
		return pos.String()
	}

	return rel + ":" + strconv.Itoa(pos.Line) + ":" + strconv.Itoa(pos.Column)
}
