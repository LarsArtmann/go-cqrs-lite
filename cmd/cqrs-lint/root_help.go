package main

// rootLongHelp is the root command's long help text: the hand-written
// usage block listing every subcommand plus the suppression syntax guide.
// It lives here (not inline in main) so TestHelpListsEveryRegisteredCommand
// can diff the documented command list against the registered commands —
// the `changelog` omission class becomes a mechanical test failure instead
// of a human-checked hope.
//
//nolint:gochecknoglobals // read-only help text
var rootLongHelp = "cqrs-lint — Domain-aware linter for go-cqrs-lite consumers\n\n" +
	"Analyzes Go projects for CQRS anti-patterns, correctness bugs, and API misuse.\n\n" +
	"Usage:\n" +
	"  cqrs-lint [path] [flags]     Lint Go project for CQRS anti-patterns (default)\n" +
	"  cqrs-lint rules              List all available rules\n" +
	"  cqrs-lint explain            Explain the .cqrs-lint.json config format in detail\n" +
	"  cqrs-lint doctor             Show resolved config + detected feature profile\n" +
	"  cqrs-lint scorecard          Show module adoption scorecard (used/missing/coverage)\n" +
	"  cqrs-lint init               Create a .cqrs-lint.json with defaults\n" +
	"  cqrs-lint version            Print version\n" +
	"  cqrs-lint changelog          Print commits since the last release tag\n\n" +
	"SUPPRESSIONS:\n\n" +
	"  Inline (single rule):\n" +
	"    //cqrs-lint:ignore(C007) reason text\n\n" +
	"  Inline (multiple rules):\n" +
	"    //cqrs-lint:ignore(C007,A001) reason text\n\n" +
	"  Block:\n" +
	"    //cqrs-lint:ignore-start\n" +
	"    ...code...\n" +
	"    //cqrs-lint:ignore-end\n\n" +
	"  Block (specific rules):\n" +
	"    //cqrs-lint:ignore-start(C007,A001)\n" +
	"    ...code...\n" +
	"    //cqrs-lint:ignore-end\n\n" +
	"  Both //cqrs-lint: and // cqrs-lint: (any spacing) are accepted.\n" +
	"  Place inline suppressions on the line above the code or at end of line.\n" +
	"  Several directives may share one line; each contributes its rules.\n" +
	"  Directives as text inside /* */ blocks or raw strings are inert.\n" +
	"  --fail-on-stale-suppressions also flags stray ignore-end lines and\n" +
	"  ignore-start blocks never closed.\n" +
	"  Struct-field-level: place the comment directly above the field.\n\n" +
	"  Disable rules project-wide via config: {\"rules\": {\"disable\": [\"P012\"]}}\n" +
	"  or the --exclude-rules flag.\n" +
	"  Rule-specific config: {\"rules\": {\"external-api-struct-prefixes\": [\"Discord\"]}}.\n"
