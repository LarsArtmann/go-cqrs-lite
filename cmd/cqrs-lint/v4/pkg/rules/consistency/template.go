package consistency

import "github.com/larsartmann/go-finding"

// findingTemplate stamps the tool name shared by every consistency finding;
// category stays per-site because rules in this package vary. Rules build
// findings through findingTemplate.Builder(...) so a future tool change is
// a one-line edit here.
var findingTemplate = finding.NewTemplate(toolName)
