// Package adoption implements Feature Adoption Coaching rules (F-series).
//
// These rules proactively suggest go-cqrs-lite features the consumer project
// is NOT yet using. Unlike correctness or API-misuse rules that flag bugs in
// existing code, adoption rules detect ABSENCE of beneficial patterns and
// coach the user toward adopting them.
//
// F-series findings are advisory coaching, not errors: Info by default,
// Warning when the coached-against module is scheduled for removal (F030 on
// the deprecated transport/* modules). Most rules fire once per coaching
// scope (the project, or per module under per-module analysis); F021
// additionally reports each offending query.
package adoption

import "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"

const toolName = lintutil.ToolName
