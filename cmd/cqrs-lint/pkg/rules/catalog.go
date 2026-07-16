// Package rules provides centralized rule registration for cqrs-lint.
package rules

type RuleInfo struct {
	ID          string
	Name        string
	Category    string
	Severity    string
	Confidence  string
	Description string
	AutoFix     bool
}

// AllRules returns metadata for all available rules.

func AllRules() []RuleInfo {
	return append(append(coreRules(), extraRules()...), extraRulesBatch2()...)
}

func coreRules() []RuleInfo {
	return []RuleInfo{
		{
			ID:          "C001",
			Name:        "missing-tx-commit",
			Category:    "correctness",
			Severity:    "critical",
			Confidence:  "high",
			Description: "Transaction wrapper returns nil instead of tx.Commit()",
			AutoFix:     true,
		},
		{
			ID:          "C002",
			Name:        "broken-command-id",
			Category:    "correctness",
			Severity:    "critical",
			Confidence:  "high",
			Description: "Command ID() returns zero value — breaks idempotency",
			AutoFix:     false,
		},
		{
			ID:          "C003",
			Name:        "silent-unknown-event-fold",
			Category:    "correctness",
			Severity:    "error",
			Confidence:  "high",
			Description: "Fold function silently ignores unknown event types",
			AutoFix:     true,
		},
		{
			ID:          "C005",
			Name:        "raw-json-unmarshal-payload",
			Category:    "correctness",
			Severity:    "error",
			Confidence:  "high",
			Description: "Raw json.Unmarshal on event payload instead of DecodePayloadAuto",
			AutoFix:     false,
		},
		{
			ID:          "C006",
			Name:        "manual-version-arithmetic",
			Category:    "correctness",
			Severity:    "warning",
			Confidence:  "high",
			Description: "Manual event.Version(x.Int()+1) instead of x.Increment()",
			AutoFix:     true,
		},
		{
			ID:          "C007",
			Name:        "time-now-in-decider",
			Category:    "correctness",
			Severity:    "warning",
			Confidence:  "medium",
			Description: "time.Now() inside decider — non-deterministic",
			AutoFix:     false,
		},
		{
			ID:          "C008",
			Name:        "float64-for-money",
			Category:    "correctness",
			Severity:    "warning",
			Confidence:  "medium",
			Description: "float64 field with monetary name — use decimal or cents",
			AutoFix:     false,
		},
		{
			ID:          "C009",
			Name:        "panic-in-production",
			Category:    "correctness",
			Severity:    "warning",
			Confidence:  "high",
			Description: "panic() in production code — use error returns",
			AutoFix:     false,
		},
		{
			ID:          "C010",
			Name:        "swallowed-error-in-fold",
			Category:    "correctness",
			Severity:    "warning",
			Confidence:  "high",
			Description: "Error from decode/unmarshal discarded in fold",
			AutoFix:     false,
		},
		{
			ID:          "C012",
			Name:        "missing-error-return-in-with-tx",
			Category:    "correctness",
			Severity:    "critical",
			Confidence:  "high",
			Description: "withTx ignores body error — failures silently lost",
			AutoFix:     false,
		},
		{
			ID:          "A001",
			Name:        "manual-command-interface",
			Category:    "api",
			Severity:    "error",
			Confidence:  "high",
			Description: "Manual Type()/ID()/AggregateID() instead of BasicCommand embedding",
			AutoFix:     false,
		},
		{
			ID:          "A002",
			Name:        "newevent-manual-marshal",
			Category:    "api",
			Severity:    "warning",
			Confidence:  "high",
			Description: "event.NewEvent with json.Marshal — use event.New for auto-marshal",
			AutoFix:     false,
		},
		{
			ID:          "A003",
			Name:        "explicit-codec-in-decode",
			Category:    "api",
			Severity:    "info",
			Confidence:  "medium",
			Description: "Explicit codec in DecodePayload — use DecodePayloadAuto",
			AutoFix:     false,
		},
		{
			ID:          "A004",
			Name:        "untyped-dispatch-register",
			Category:    "api",
			Severity:    "warning",
			Confidence:  "medium",
			Description: "Untyped handler with type assertion — use RegisterTyped",
			AutoFix:     false,
		},
		{
			ID:          "A005",
			Name:        "custom-projection-runner",
			Category:    "api",
			Severity:    "warning",
			Confidence:  "medium",
			Description: "Manual bus.SubscribeAll without projectionhost",
			AutoFix:     false,
		},
		{
			ID:          "A006",
			Name:        "adapter-layer-wrapping",
			Category:    "api",
			Severity:    "info",
			Confidence:  "low",
			Description: "WrapEvent/UnwrapEvent adapter methods",
			AutoFix:     false,
		},
		{
			ID:          "A007",
			Name:        "dual-model-oo-functional",
			Category:    "api",
			Severity:    "error",
			Confidence:  "medium",
			Description: "Both OO aggregates and functional deciders",
			AutoFix:     false,
		},
		{
			ID:          "A008",
			Name:        "parallel-type-system",
			Category:    "api",
			Severity:    "error",
			Confidence:  "high",
			Description: "Custom AggregateID/Version types duplicating go-cqrs-lite",
			AutoFix:     false,
		},
	}
}
