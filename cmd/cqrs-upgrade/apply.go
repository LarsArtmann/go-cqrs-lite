package main

import (
	"fmt"
	"strings"

	"golang.org/x/mod/modfile"
)

// bump describes one module version change.
type bump struct {
	Module     string
	From, To   string
	upToDate   bool
	resolveErr error
}

// planUpgrades resolves the latest version for each pin and records the
// change (or the reason nothing changes).
func planUpgrades(pins []pin) []bump {
	bumps := make([]bump, 0, len(pins))

	for _, p := range pins {
		b := bump{Module: p.Module, From: p.Current}

		latest, err := versionResolver(p.Module)
		if err != nil {
			b.resolveErr = err
		} else {
			b.To = latest
			b.upToDate = latest == "" || latest == p.Current
		}

		bumps = append(bumps, b)
	}

	return bumps
}

// editGoMod rewrites the go.mod require directives to the target versions.
// Offline-safe (pure file edit); network verification happens in verify.
func editGoMod(path string, bumps []bump) error {
	data, err := osReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	mod, err := modfile.Parse(path, data, nil)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	target := map[string]string{}

	for _, b := range bumps {
		if b.To != "" && b.To != b.From {
			target[b.Module] = b.To
		}
	}

	if len(target) == 0 {
		return nil
	}

	for _, req := range mod.Require {
		if v, ok := target[req.Mod.Path]; ok {
			req.Mod.Version = v
		}
	}

	mod.Cleanup()

	out, err := mod.Format()
	if err != nil {
		return fmt.Errorf("format %s: %w", path, err)
	}

	if err := osWriteFile(path, out); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// verify runs go mod tidy + build + vet with the workspace isolated, so the
// consumer is validated against PUBLISHED versions exactly like CI.
func verify(dir string) error {
	if out, err := goOut(dir, "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w\n%s", err, out)
	}

	if _, err := goOut(dir, "build", "./..."); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	if _, err := goOut(dir, "vet", "./..."); err != nil {
		return fmt.Errorf("go vet: %w", err)
	}

	return nil
}

// formatBumps renders the dry-run/apply table.
func formatBumps(bumps []bump) string {
	var sb strings.Builder

	sb.WriteString("module\tcurrent\tlatest\tstatus\n")

	for _, b := range bumps {
		status := "bump"
		switch {
		case b.resolveErr != nil:
			status = "resolve-error"
		case b.upToDate:
			status = "up-to-date"
		}

		to := b.To
		if to == "" {
			to = "-"
		}

		sb.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\n", b.Module, b.From, to, status))
	}

	return sb.String()
}
