// Package resilience implements cqrs-lint rules that detect missing resilience
// patterns: absent retry middleware, missing circuit breakers, and missing
// dead-letter configuration on projection hosts.
//
// All rules in this package are consumer-coaching: they are skipped when
// linting the go-cqrs-lite library itself (IsLibrarySelfLint).
package resilience

import "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"

const toolName = lintutil.ToolName
