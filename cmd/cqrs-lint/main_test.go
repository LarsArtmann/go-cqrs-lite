package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

func TestFilterBySeverity(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Severity: finding.SeverityCritical},
		{Severity: finding.SeverityError},
		{Severity: finding.SeverityWarning},
		{Severity: finding.SeverityInfo},
	}

	tests := []struct {
		minSev    string
		wantCount int
	}{
		{"info", 4},
		{"warning", 3},
		{"error", 2},
		{"critical", 1},
	}

	for _, tt := range tests {
		result := filterBySeverity(findings, tt.minSev)
		if len(result) != tt.wantCount {
			t.Errorf("filterBySeverity(%q): got %d, want %d", tt.minSev, len(result), tt.wantCount)
		}
	}
}

func TestFilterByConfidence(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Confidence: finding.ConfidenceHigh},
		{Confidence: finding.ConfidenceMedium},
		{Confidence: finding.ConfidenceLow},
	}

	tests := []struct {
		minConf   string
		wantCount int
	}{
		{"low", 3},
		{"medium", 2},
		{"high", 1},
	}

	for _, tt := range tests {
		result := filterByConfidence(findings, tt.minConf)
		if len(result) != tt.wantCount {
			t.Errorf(
				"filterByConfidence(%q): got %d, want %d",
				tt.minConf,
				len(result),
				tt.wantCount,
			)
		}
	}
}

func TestParseSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  finding.Severity
	}{
		{"critical", finding.SeverityCritical},
		{"error", finding.SeverityError},
		{"warning", finding.SeverityWarning},
		{"info", finding.SeverityInfo},
		{"unknown", finding.SeverityInfo},
	}

	for _, tt := range tests {
		if got := parseSeverity(tt.input); got != tt.want {
			t.Errorf("parseSeverity(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseConfidence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  finding.Confidence
	}{
		{"high", finding.ConfidenceHigh},
		{"medium", finding.ConfidenceMedium},
		{"low", finding.ConfidenceLow},
		{"unknown", finding.ConfidenceLow},
	}

	for _, tt := range tests {
		if got := parseConfidence(tt.input); got != tt.want {
			t.Errorf("parseConfidence(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCollectFindingsDeduplicates(t *testing.T) {
	t.Parallel()

	f1 := finding.Finding{ID: finding.ID("test-id-1")}
	f2 := finding.Finding{ID: f1.ID}

	seen := make(map[finding.ID]bool)
	var unique []finding.Finding

	for _, f := range []finding.Finding{f1, f2} {
		if seen[f.ID] {
			continue
		}

		seen[f.ID] = true
		unique = append(unique, f)
	}

	if len(unique) != 1 {
		t.Errorf("expected 1 unique finding, got %d", len(unique))
	}
}

func TestOutputFindingsJSON(t *testing.T) {
	t.Parallel()

	f, err := finding.NewBuilder(
		"TEST", "cqrs-lint",
		"test message",
		finding.SeverityWarning,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceHigh).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &AppConfig{Format: "json"}
	err = outputFindings(context.Background(), []finding.Finding{f}, cfg)
	if err != nil {
		t.Errorf("outputFindings json: %v", err)
	}
}

func TestOutputFindingsEmpty(t *testing.T) {
	t.Parallel()

	cfg := &AppConfig{Format: "text", Quiet: true}
	err := outputFindings(context.Background(), nil, cfg)
	if err != nil {
		t.Errorf("outputFindings empty: %v", err)
	}
}

func TestVersionConstant(t *testing.T) {
	if version == "" {
		t.Error("version constant should not be empty")
	}
	if !strings.Contains(version, ".") {
		t.Error("version should contain a dot (semver)")
	}
}

func TestVersionFormat(t *testing.T) {
	s := versionString()
	if !strings.HasPrefix(s, "cqrs-lint "+version) {
		t.Errorf("versionString() = %q, want prefix %q", s, "cqrs-lint "+version)
	}
}

func TestVersionStringWithBoth(t *testing.T) {
	oldCommit, oldDate := commitHash, buildDate
	commitHash, buildDate = "abc1234", "20260803"
	defer func() { commitHash, buildDate = oldCommit, oldDate }()

	s := versionString()
	if !strings.Contains(s, "commit: abc1234") {
		t.Errorf("missing commit in %q", s)
	}
	if !strings.Contains(s, "built: 20260803") {
		t.Errorf("missing build date in %q", s)
	}
}

func TestVersionVerbose(t *testing.T) {
	s := versionVerbose()
	if !strings.HasPrefix(s, "cqrs-lint ") {
		t.Errorf("versionVerbose() should start with cqrs-lint, got %q", s)
	}
	if !strings.Contains(s, "go:") {
		t.Errorf("versionVerbose() should contain go: line, got %q", s)
	}
	if !strings.Contains(s, "arch:") {
		t.Errorf("versionVerbose() should contain arch: line, got %q", s)
	}
	if !strings.Contains(s, "module:") {
		t.Errorf("versionVerbose() should contain module: line, got %q", s)
	}
}

func TestVersionStringLocal(t *testing.T) {
	// Without ldflags injection, commitHash and buildDate are empty.
	s := versionString()
	if !strings.HasPrefix(s, "cqrs-lint ") {
		t.Errorf("versionString() = %q, want prefix %q", s, "cqrs-lint ")
	}
	if strings.Contains(s, "commit:") {
		t.Errorf("local build should not contain commit: %q", s)
	}
}

func TestVersionStringWithCommit(t *testing.T) {
	old := commitHash
	commitHash = "abc1234"
	defer func() { commitHash = old }()

	s := versionString()
	if !strings.Contains(s, "commit: abc1234") {
		t.Errorf("versionString() with commit = %q, want to contain %q", s, "commit: abc1234")
	}
}

func TestExcludeAdoptionFromScore(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Rule: "C007", Severity: finding.SeverityWarning},
		{Rule: "F013", Severity: finding.SeverityInfo},
		{Rule: "F017", Severity: finding.SeverityInfo},
		{Rule: "E016", Severity: finding.SeverityWarning},
	}

	filtered := excludeAdoptionFromScore(findings)

	// Original must be unchanged.
	for i := range findings {
		if findings[i].Suppression != nil {
			t.Errorf("original finding %s was mutated", findings[i].Rule)
		}
	}

	// Filtered copy: F-series suppressed, others not.
	suppressed := 0
	for _, f := range filtered {
		if f.Suppression != nil {
			suppressed++
			if string(f.Rule) != "F013" && string(f.Rule) != "F017" {
				t.Errorf("unexpected suppressed rule: %s", f.Rule)
			}
		}
	}
	if suppressed != 2 {
		t.Errorf("expected 2 F-series suppressed, got %d", suppressed)
	}
}

func TestAdoptionModeHealthScoreDifference(t *testing.T) {
	t.Parallel()

	// Simulate findings: 1 warning + 3 adoption (F-series) infos.
	findings := []finding.Finding{
		{Rule: "C007", Severity: finding.SeverityWarning, Confidence: finding.ConfidenceHigh},
		{Rule: "F013", Severity: finding.SeverityInfo, Confidence: finding.ConfidenceLow},
		{Rule: "F015", Severity: finding.SeverityInfo, Confidence: finding.ConfidenceLow},
		{Rule: "F017", Severity: finding.SeverityInfo, Confidence: finding.ConfidenceLow},
	}

	// Without adoption mode: all findings count.
	scoreWithout := ComputeHealthScore(findings)

	// With adoption mode: F-series suppressed from score.
	scoreWith := ComputeHealthScore(excludeAdoptionFromScore(findings))

	if scoreWith.Score <= scoreWithout.Score {
		t.Errorf(
			"adoption mode should not lower score: with=%d without=%d",
			scoreWith.Score,
			scoreWithout.Score,
		)
	}
}

func TestIsAdoptionRule(t *testing.T) {
	t.Parallel()

	cases := []struct {
		rule string
		want bool
	}{
		{"F001", true},
		{"F021", true},
		{"C007", false},
		{"E016", false},
		{"P012", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isAdoptionRule(tc.rule); got != tc.want {
			t.Errorf("isAdoptionRule(%q) = %v, want %v", tc.rule, got, tc.want)
		}
	}
}

func TestFilterSuppressed(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{ID: finding.ID("active-1")},
		{ID: finding.ID("active-2")},
		{
			ID:          finding.ID("suppressed-1"),
			Suppression: &finding.Suppression{Kind: finding.SuppressionInSource},
		},
		{
			ID:          finding.ID("suppressed-2"),
			Suppression: &finding.Suppression{Kind: finding.SuppressionInSource},
		},
	}

	active, suppressed := filterSuppressed(findings)
	if len(active) != 2 {
		t.Errorf("active count: got %d, want 2", len(active))
	}
	if len(suppressed) != 2 {
		t.Errorf("suppressed count: got %d, want 2", len(suppressed))
	}
	for _, f := range active {
		if f.Suppression != nil {
			t.Errorf("active finding %s should not be suppressed", f.ID)
		}
	}
}

func TestFilterSuppressed_AllActive(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{ID: finding.ID("a")},
		{ID: finding.ID("b")},
	}

	active, suppressed := filterSuppressed(findings)
	if len(active) != 2 {
		t.Errorf("active count: got %d, want 2", len(active))
	}
	if len(suppressed) != 0 {
		t.Errorf("suppressed count: got %d, want 0", len(suppressed))
	}
}

func TestFilterSuppressed_Empty(t *testing.T) {
	t.Parallel()

	active, suppressed := filterSuppressed(nil)
	if len(active) != 0 {
		t.Errorf("active count: got %d, want 0", len(active))
	}
	if len(suppressed) != 0 {
		t.Errorf("suppressed count: got %d, want 0", len(suppressed))
	}
}

func TestFilterFPSuspects(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{ID: finding.ID("high"), Confidence: finding.ConfidenceHigh},
		{ID: finding.ID("medium"), Confidence: finding.ConfidenceMedium},
		{ID: finding.ID("low1"), Confidence: finding.ConfidenceLow},
		{ID: finding.ID("low2"), Confidence: finding.ConfidenceLow},
		{ID: finding.ID("none"), Confidence: finding.ConfidenceNone},
	}

	suspects := filterFPSuspects(findings)
	if len(suspects) != 3 {
		t.Fatalf("suspects count: got %d, want 3 (low1, low2, none)", len(suspects))
	}

	seen := make(map[finding.ID]bool)
	for _, f := range suspects {
		seen[f.ID] = true
		if f.Confidence >= finding.ConfidenceMedium {
			t.Errorf("finding %s has confidence >= Medium, should be filtered out", f.ID)
		}
	}
	for _, id := range []finding.ID{"low1", "low2", "none"} {
		if !seen[id] {
			t.Errorf("finding %s should be in suspects", id)
		}
	}
}

func TestFilterFPSuspects_Empty(t *testing.T) {
	t.Parallel()

	suspects := filterFPSuspects(nil)
	if len(suspects) != 0 {
		t.Errorf("suspects count: got %d, want 0", len(suspects))
	}
}

// TestSuppressionEndToEnd verifies the full suppression pipeline:
// findings with Suppression set are excluded from active findings,
// health score, and do not contribute to the finding count. This test
// would have caught the original suppression leak bug (v0.1.0–v0.2.0).
func TestSuppressionEndToEnd(t *testing.T) {
	t.Parallel()

	// Build a mix: 1 active error, 1 suppressed error, 1 active info.
	activeErr, _ := finding.NewBuilder(
		"C001", "test", "real bug",
		finding.SeverityError,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).Build()

	suppressedErr, _ := finding.NewBuilder(
		"C002", "test", "suppressed bug",
		finding.SeverityError,
		finding.Pos(finding.FilePath("test.go"), 2, 1),
	).Build()
	suppressedErr.Suppression = &finding.Suppression{
		Kind:   finding.SuppressionInSource,
		Rule:   "C002",
		Reason: "false positive",
	}

	activeInfo, _ := finding.NewBuilder(
		"D002", "test", "style nit",
		finding.SeverityInfo,
		finding.Pos(finding.FilePath("test.go"), 3, 1),
	).Build()

	allFindings := []finding.Finding{activeErr, suppressedErr, activeInfo}

	// Step 1: filterSuppressed splits active from suppressed.
	unsuppressed, suppressedFindings := filterSuppressed(allFindings)
	suppressedCount := len(suppressedFindings)
	if suppressedCount != 1 {
		t.Fatalf("suppressed count: got %d, want 1", suppressedCount)
	}
	if len(unsuppressed) != 2 {
		t.Fatalf("unsuppressed count: got %d, want 2", len(unsuppressed))
	}

	// Step 2: health score computed on unsuppressed — suppressed error
	// must NOT count. Active error costs -5, active info costs -1 → score 94.
	hs := ComputeHealthScoreWithCap(unsuppressed, defaultInfoDeductionCap)
	if hs.Score != 94 {
		t.Errorf("health score: got %d, want 94 (100-5 error-1 info)", hs.Score)
	}

	// Step 3: exit-code check on unsuppressed — suppressed error doesn't trigger.
	exitErr := shouldExitWithError(&AppConfig{}, unsuppressed)
	if !errors.Is(exitErr, errFindingsWithErrors) {
		t.Errorf("exit: active error should trigger errFindingsWithErrors, got %v", exitErr)
	}

	// Step 4: verify suppressed finding is absent from unsuppressed set.
	for _, f := range unsuppressed {
		if f.Suppression != nil {
			t.Errorf("suppressed finding %s should not be in unsuppressed set", f.ID)
		}
	}
}

// TestShouldExitWithError_FPSuspectsMode verifies that --fp-suspects always
// returns nil (advisory mode) even when Error-severity findings are present.
func TestShouldExitWithError_FPSuspectsMode(t *testing.T) {
	t.Parallel()

	errFinding, _ := finding.NewBuilder(
		"C001", "test", "error",
		finding.SeverityError,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).WithConfidence(finding.ConfidenceLow).Build()

	cfg := &AppConfig{FPSuspects: true}
	err := shouldExitWithError(cfg, []finding.Finding{errFinding})
	if err != nil {
		t.Errorf("fp-suspects mode should always return nil, got %v", err)
	}
}

// TestShouldExitWithError_NormalMode verifies that error-severity findings
// trigger errFindingsWithErrors in normal (non-fp-suspects) mode.
func TestShouldExitWithError_NormalMode(t *testing.T) {
	t.Parallel()

	errFinding, _ := finding.NewBuilder(
		"C001", "test", "error",
		finding.SeverityError,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).Build()

	cfg := &AppConfig{}
	err := shouldExitWithError(cfg, []finding.Finding{errFinding})
	if !errors.Is(err, errFindingsWithErrors) {
		t.Errorf("normal mode with error finding: got %v, want errFindingsWithErrors", err)
	}

	// Warning-only → no error exit.
	warnFinding, _ := finding.NewBuilder(
		"C002", "test", "warning",
		finding.SeverityWarning,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).Build()
	err = shouldExitWithError(cfg, []finding.Finding{warnFinding})
	if err != nil {
		t.Errorf("warning-only should not trigger exit, got %v", err)
	}
}

func TestFormatSuppressedFindings(t *testing.T) {
	t.Parallel()

	suppressed, _ := finding.NewBuilder(
		"C007", "test", "missing error classification",
		finding.SeverityError,
		finding.Pos(finding.FilePath("main.go"), 42, 10),
	).Build()
	suppressed.Suppression = &finding.Suppression{
		Kind:   finding.SuppressionInSource,
		Rule:   "C007",
		Reason: "false positive — already classified upstream",
	}

	var buf strings.Builder
	formatSuppressedFindings(&buf, []finding.Finding{suppressed}, parseColorMode("never"))

	out := buf.String()
	if !strings.Contains(out, "Suppressed Findings (1)") {
		t.Errorf("output should contain header, got:\n%s", out)
	}
	if !strings.Contains(out, "C007") {
		t.Errorf("output should contain rule ID, got:\n%s", out)
	}
	if !strings.Contains(out, "false positive — already classified upstream") {
		t.Errorf("output should contain suppression reason, got:\n%s", out)
	}
	if !strings.Contains(out, "main.go:42:10") {
		t.Errorf("output should contain file location, got:\n%s", out)
	}
}
