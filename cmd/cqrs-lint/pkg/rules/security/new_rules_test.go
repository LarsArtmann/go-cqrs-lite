package security_test

import (
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
)

// --- S001: Hardcoded secrets ---

func TestS001_NoCrashOnEmptyInput(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
	ruletest.AssertRule(t, findings, "S001", 0)
}

// --- S002: Missing encryption for sensitive payloads ---

func TestS002_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, security.NewS002Detector(ctx))
	ruletest.AssertRule(t, findings, "S002", 0)
}

// --- S003: Missing event signing ---

func TestS003_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, security.NewS003Detector(ctx))
	ruletest.AssertRule(t, findings, "S003", 0)
}

// --- S001: Positive test — hardcoded API key ---

func TestS001_DetectsHardcodedKey(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

func init() {
	apiKey := "supersecretvalue123"
	_ = apiKey
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
	ruletest.AssertRule(t, findings, "S001", 1)
}

// --- S002: Positive test — PII event without encryption ---

func TestS002_DetectsPIIWithoutEncryption(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserEmailChanged struct {
	Email string
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS002Detector(ctx))
	ruletest.AssertRule(t, findings, "S002", 1)
}

// --- S003: Positive test — event store without signing ---

func TestS003_DetectsMissingSigning(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	return s, nil
}

func saveEvents(store event.Store, ref event.StreamRef, events []event.Event) error {
	return store.Save(nil, ref, events)
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS003Detector(ctx))
	ruletest.AssertRule(t, findings, "S003", 1)
}

// --- FeatureProfile suppression guards ---

// TestS002_DowngradedForLocalCLI proves the HasServer toggle changes severity:
// a server project gets Error (production PII risk) while a local-only project
// is downgraded to Info (no network exposure). This guards the FeatureProfile
// rewiring of S002 against silent regression.
func TestS002_DowngradedForLocalCLI(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserEmailChanged struct {
	Email string
}
`,
	})

	ctx.FeatureProfile.HasServer = true
	serverFindings := ruletest.RunDetector(t, security.NewS002Detector(ctx))
	ruletest.AssertRule(t, serverFindings, "S002", 1)
	if serverFindings[0].Severity != finding.SeverityError {
		t.Fatalf("server project PII should be ERROR, got %s", serverFindings[0].Severity)
	}

	ctx.FeatureProfile.HasServer = false
	localFindings := ruletest.RunDetector(t, security.NewS002Detector(ctx))
	ruletest.AssertRule(t, localFindings, "S002", 1)
	if localFindings[0].Severity != finding.SeverityInfo {
		t.Errorf("local-only PII should be downgraded to INFO, got %s", localFindings[0].Severity)
	}
}

// TestS003_SuppressedForNoServer proves signing is fully suppressed when there
// is no server: a fixture that would normally fire S003 yields zero findings.
func TestS003_SuppressedForNoServer(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	return s, nil
}

func saveEvents(store event.Store, ref event.StreamRef, events []event.Event) error {
	return store.Save(nil, ref, events)
}
`,
	})

	ctx.FeatureProfile.HasServer = false
	findings := ruletest.RunDetector(t, security.NewS003Detector(ctx))
	ruletest.AssertRule(t, findings, "S003", 0)
}

// --- S007: In-memory session/token store ---

func TestS007_NoCrashOnEmptyInput(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, security.NewS007Detector(ctx))
	ruletest.AssertRule(t, findings, "S007", 0)
}

func TestS007_DetectsInMemorySessionStore(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

func setup() {
	store := NewInMemorySessionStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS007Detector(ctx))
	ruletest.AssertRule(t, findings, "S007", 1)
}

func TestS007_SuppressedWithoutServer(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"cli.go": `package main

func setup() {
	store := NewInMemorySessionStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.HasServer = false
	findings := ruletest.RunDetector(t, security.NewS007Detector(ctx))
	ruletest.AssertRule(t, findings, "S007", 0)
}

func TestS007_DetectsMemoryTokenStoreComposite(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"auth.go": `package main

type MemoryTokenStore struct{}

func setup() {
	s := MemoryTokenStore{}
	_ = s
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS007Detector(ctx))
	ruletest.AssertRule(t, findings, "S007", 1)
}

func TestS007_IgnoresCQRSEventMemoryStore(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store.go": `package main

func setup() {
	s := memory.NewStore()
	_ = s
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS007Detector(ctx))
	ruletest.AssertRule(t, findings, "S007", 0)
}

func TestS007_IgnoresInMemoryTokenBucket(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"ratelimit.go": `package main

func setup() {
	b := NewInMemoryTokenBucket()
	_ = b
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS007Detector(ctx))
	ruletest.AssertRule(t, findings, "S007", 0)
}

func TestS007_IgnoresTestFiles(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"session_test.go": `package main

func setup() {
	store := NewInMemorySessionStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS007Detector(ctx))
	ruletest.AssertRule(t, findings, "S007", 0)
}

// --- S006: Financial data without encryption ---

func TestS006_NoCrashOnEmptyInput(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 0)
}

func TestS006_DetectsStrongFinancialField(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"payment.go": `package main

type PaymentMethod struct {
	CardNumber string ` + "`json:\"card_number\"`" + `
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 1)
}

func TestS006_StrongSeverityIsError(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"bank.go": `package main

type BankAccount struct {
	IBAN string ` + "`json:\"iban\"`" + `
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 1)
	if findings[0].Severity != finding.SeverityError {
		t.Fatalf("strong financial indicator should be ERROR, got %s", findings[0].Severity)
	}
}

func TestS006_DetectsMediumFinancialField(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"payroll.go": `package main

type EmployeePayroll struct {
	Salary float64 ` + "`json:\"salary\"`" + `
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 1)
}

func TestS006_DetectsWeakCompound(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"order.go": `package main

type OrderTotal struct {
	Amount  float64 ` + "`json:\"amount\"`" + `
	Balance float64 ` + "`json:\"balance\"`" + `
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 1)
}

func TestS006_SuppressesSingleWeakField(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"product.go": `package main

type Product struct {
	Price float64 ` + "`json:\"price\"`" + `
	Name  string ` + "`json:\"name\"`" + `
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 0)
}

func TestS006_RequiresSerializationTags(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"calc.go": `package main

type TaxCalculator struct {
	CardNumber string
	Amount     float64
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 0)
}

func TestS006_SuppressedWithEncryptionImport(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"payroll.go": `package main

import "github.com/larsartmann/go-cqrs-lite/encryption/v4"

type EmployeePayroll struct {
	Salary float64 ` + "`json:\"salary\"`" + `
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 0)
}

func TestS006_DowngradedForLocalCLI(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"payroll.go": `package main

type EmployeePayroll struct {
	Salary float64 ` + "`json:\"salary\"`" + `
}
`,
	})

	ctx.FeatureProfile.HasServer = true
	serverFindings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, serverFindings, "S006", 1)

	ctx.FeatureProfile.HasServer = false
	localFindings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, localFindings, "S006", 1)
	if localFindings[0].Severity != finding.SeverityInfo {
		t.Errorf("local-only financial data should be downgraded to INFO, got %s",
			localFindings[0].Severity)
	}
}

func TestS006_IgnoresTestFiles(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"payment_test.go": `package main

type PaymentMethod struct {
	CardNumber string ` + "`json:\"card_number\"`" + `
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 0)
}

func TestS006_DetectsFinancialTypeName(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"billing.go": `package main

type Invoice struct {
	ID    string ` + "`json:\"id\"`" + `
	Title string ` + "`json:\"title\"`" + `
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 1)
}

func TestS006_IgnoresNonFinancialStruct(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"user.go": `package main

type User struct {
	Name  string ` + "`json:\"name\"`" + `
	Email string ` + "`json:\"email\"`" + `
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 0)
}

func TestS006_IgnoresPanelSubstring(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"ui.go": `package main

type DetailsPanelConfig struct {
	Sections []string ` + "`json:\"sections,omitempty\"`" + `
}`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 0)
}

func TestS006_IgnoresDatabaseSubstring(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"stats.go": `package main

type DiskStats struct {
	DatabaseBytes int64 ` + "`json:\"databaseBytes\"`" + `
	EventBytes    int64 ` + "`json:\"eventBytes\"`" + `
}`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 0)
}

func TestS006_DetectsPrimaryAccountNumber(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"card.go": `package main

type PaymentCard struct {
	PrimaryAccountNumber string ` + "`json:\"pan\"`" + `
}`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 1)
}
