package correctness

import "github.com/larsartmann/go-finding"

// findingTemplate stamps the fields every correctness finding shares: the
// tool name and the correctness category. Rules build findings through
// findingTemplate.Builder(...) so per-site boilerplate stays minimal and a
// future tool/category change is a one-line edit here.
var findingTemplate = finding.NewTemplate(toolName).
	WithCategory(finding.CategoryCorrectness)
