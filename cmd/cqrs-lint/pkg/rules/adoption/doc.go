// Package adoption implements Feature Adoption Coaching rules (F-series).
//
// These rules proactively suggest go-cqrs-lite features the consumer project
// is NOT yet using. Unlike correctness or API-misuse rules that flag bugs in
// existing code, adoption rules detect ABSENCE of beneficial patterns and
// coach the user toward adopting them.
//
// All F-series rules emit SeverityInfo findings — they are suggestions, not
// errors. Each fires at most once per project (project-level, not per-file).
package adoption

import "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"

const toolName = lintutil.ToolName
