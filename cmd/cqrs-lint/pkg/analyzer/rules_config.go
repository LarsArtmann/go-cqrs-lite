package analyzer

// RulesConfig holds rule-specific configuration that detectors consult to
// suppress domain-specific false positives. Populated from the "rules" key in
// .cqrs-lint.json. Fields are intentionally narrow: each one targets a
// concrete false-positive pattern documented in consumer feedback.
//
// Zero value (the default when no config is present) disables every override,
// so detectors behave exactly as before — this is the contract
// BuildContextFromSource relies on for rule unit tests.
type RulesConfig struct {
	// ExternalAPIStructPrefixes lists struct-name prefixes whose JSON tags
	// mirror an external API (Discord, Stripe, GitHub, ...) and must NOT count
	// toward D002's mixed-casing check. Example: ["Discord", "Stripe"] marks
	// every struct whose name starts with "Discord" or "Stripe" as an external
	// mirror, so its snake_case tags no longer trigger "mixes camelCase and
	// snake_case" on files that also define camelCase consumer types.
	//
	// For one-off cases prefer the in-source marker
	//
	//	//cqrs-lint:external-api
	//
	// on the struct's doc comment; it suppresses the same rule without needing
	// config. Both mechanisms stack: a struct is excluded if either matches.
	ExternalAPIStructPrefixes []string `json:"external-api-struct-prefixes,omitempty"`
}
