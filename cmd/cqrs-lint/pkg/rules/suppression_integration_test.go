package rules_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/suppression"
)

// TestNewRulesSuppressedByInlineComment verifies that each of the 12 rules
// added in the 2026-07-30 expansion (C031-C034, P011-P012, D014-D015, A032,
// E016-E017, S010) can be suppressed via an inline //cqrs-lint:ignore(RULE)
// comment. The source is written to real temp files so the suppression filter
// can read the file at the finding's position.
//
// Note: ParseSuppressions only matches standalone comment lines (lines that
// start with //cqrs-lint:ignore), so the suppression directive must be on its
// own line, either on the finding line or the line above.
func TestNewRulesSuppressedByInlineComment(t *testing.T) {
	t.Parallel()

	cases := []struct {
		ruleID  string
		sources map[string]string
	}{
		{
			"C031",
			map[string]string{"handler.go": `package main

import "context"

func setup() {
	command.RegisterTyped(disp, "user.create", func(ctx context.Context, cmd *CreateUser) error {
		_, err := validate(cmd)
		if err != nil {
			//cqrs-lint:ignore(C031) intentionally swallowed
			return nil
		}
		return nil
	})
}
`},
		},
		{
			"C032",
			map[string]string{"handler.go": `package main

import "context"

func handle(ctx context.Context, cmd *Command) error {
	//cqrs-lint:ignore(C032) detached context ok
	bgCtx := context.Background()
	return process(bgCtx, cmd)
}
`},
		},
		{
			"C033",
			map[string]string{"store.go": `package main

import "context"

func save(ctx context.Context) error {
	//cqrs-lint:ignore(C033) unwrapped ok
	if err := store.Save(ctx, evt); err != nil {
		return err
	}
	return nil
}
`},
		},
		{
			"C034",
			map[string]string{"handler.go": `package main

import "context"

func handle(ctx context.Context, cmd *Command) error {
	//cqrs-lint:ignore(C034) fire-and-forget ok
	go func() {
		process(cmd)
	}()
	return nil
}
`},
		},
		{
			"P011",
			map[string]string{"model.go": `package main

type UserReadModel struct {
	//cqrs-lint:ignore(P011) bounded externally
	users map[string]*User
}
`},
		},
		{
			"P012",
			map[string]string{"setup.go": `//cqrs-lint:ignore(P012) wal not needed
package main

import "database/sql"

func setup() {
	backend, _ := storage.NewSQLiteBackend(db)
	_ = backend
}
`},
		},
		{
			"D014",
			map[string]string{"events.go": `package main

type UserCreated struct {
	//cqrs-lint:ignore(D014) intentionally untagged
	Name string
}
`},
		},
		{
			"D015",
			map[string]string{"events.go": `package main

type UserCreated struct {
	//cqrs-lint:ignore(D015) nullable by design
	Name *string
}
`},
		},
		{
			"A032",
			map[string]string{"model.go": `package main

import "github.com/larsartmann/go-cqrs-lite/id/v4"

type User struct {
	//cqrs-lint:ignore(A032) external id type
	UserID string
}
`},
		},
		{
			"E016",
			map[string]string{"server.go": `package main

import "net/http"

func runServer() {
	srv := &http.Server{Addr: ":8080"}
	//cqrs-lint:ignore(E016) no health check needed
	_ = srv.ListenAndServe()
}
`},
		},
		{
			"E017",
			map[string]string{"main.go": `package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ch := make(chan os.Signal, 1)
	//cqrs-lint:ignore(E017) no graceful shutdown needed
	signal.Notify(ch, syscall.SIGTERM)
	<-ch
}
`},
		},
		{
			"S010",
			map[string]string{"setup.go": `package main

func setup() {
	//cqrs-lint:ignore(S010) store encrypted elsewhere
	bus.Use(encryption.EncryptMiddleware(enc))
}
`},
		},
	}

	filter := suppression.NewSuppressionFilter()

	for _, tc := range cases {
		t.Run(tc.ruleID, func(t *testing.T) {
			t.Parallel()

			ctx, _ := analyzer.BuildContextFromTempFiles(t, tc.sources)
			detectors := rules.RegisterAll(ctx)

			// Find the detector for this rule.
			var det finding.Detector
			for _, d := range detectors {
				if len(d.Name()) >= 4 && d.Name()[:4] == tc.ruleID {
					det = d
					break
				}
			}
			if det == nil {
				t.Fatalf("no detector found for rule %s", tc.ruleID)
			}

			findings, err := det.Detect(context.Background())
			if err != nil {
				t.Fatalf("detector %s: %v", det.Name(), err)
			}

			// Must have at least one finding for this rule.
			var ruleFindings []finding.Finding
			for _, f := range findings {
				if string(f.Rule) == tc.ruleID {
					ruleFindings = append(ruleFindings, f)
				}
			}
			if len(ruleFindings) == 0 {
				t.Fatalf("rule %s produced 0 findings — trigger source may need adjustment", tc.ruleID)
			}

			// Run the suppression filter.
			filtered, err := filter.Transform(context.Background(), ruleFindings)
			if err != nil {
				t.Fatalf("suppression filter: %v", err)
			}

			for _, f := range filtered {
				if f.Suppression == nil {
					t.Errorf(
						"rule %s: finding at %s:%d was NOT suppressed — the //cqrs-lint:ignore comment did not match",
						tc.ruleID,
						f.Position.File,
						f.Position.Line,
					)
				}
			}
		})
	}
}
