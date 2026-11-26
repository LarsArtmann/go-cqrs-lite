package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func listMarkers(w io.Writer, pkgs []pkg) error {
	for _, p := range pkgs {
		for _, m := range p.markers {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				p.dir, m.kind, m.structName, m.wire, m.constName); err != nil {
				return fmt.Errorf("list markers: %w", err)
			}
		}
	}

	return nil
}

func writeGenerated(pkgs []pkg, opts genOptions, output, pkgName string) error {
	if len(pkgs) == 0 {
		fmt.Fprintln(os.Stderr, "no cqrs markers found")

		return nil
	}

	targets, err := targets(pkgs, output, pkgName)
	if err != nil {
		return err
	}

	for _, t := range targets {
		if err := writeIfChanged(t, opts); err != nil {
			return err
		}
	}

	return nil
}

func checkGenerated(pkgs []pkg, opts genOptions, output, pkgName string) error {
	if len(pkgs) == 0 {
		return errors.New("no cqrs markers found under the given paths")
	}

	targets, err := targets(pkgs, output, pkgName)
	if err != nil {
		return err
	}

	var stale []string

	for _, t := range targets {
		fresh, err := isFresh(t, opts)
		if err != nil {
			return err
		}

		if !fresh {
			stale = append(stale, t.path)
		}
	}

	if len(stale) > 0 {
		return fmt.Errorf("generated code is missing or stale, run cqrs-gen: %s",
			strings.Join(stale, ", "))
	}

	fmt.Printf("up to date: %d package(s)\n", len(targets))

	return nil
}

func targets(pkgs []pkg, output, pkgName string) ([]target, error) {
	if output == "" {
		out := make([]target, 0, len(pkgs))

		for _, p := range pkgs {
			out = append(out, target{path: filepath.Join(p.dir, generatedFile), pkg: p})
		}

		return out, nil
	}

	if len(pkgs) > 1 {
		dirs := make([]string, 0, len(pkgs))

		for _, p := range pkgs {
			dirs = append(dirs, p.dir)
		}

		return nil, fmt.Errorf("%d packages declare markers (%s): one --output file cannot "+
			"serve them all, drop --output to write %s into each package",
			len(pkgs), strings.Join(dirs, ", "), generatedFile)
	}

	p := pkgs[0]
	if pkgName != "" {
		p.name = pkgName
	}

	return []target{{path: output, pkg: p}}, nil
}

func writeIfChanged(t target, opts genOptions) error {
	code, err := generate(t.pkg, opts)
	if err != nil {
		return err
	}

	current, err := os.ReadFile(t.path)
	if err == nil && string(current) == code {
		fmt.Printf("unchanged %s (%d markers)\n", t.path, len(t.pkg.markers))

		return nil
	}

	if err := os.WriteFile(t.path, []byte(code), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", t.path, err)
	}

	fmt.Printf("generated %s (%d markers)\n", t.path, len(t.pkg.markers))

	return nil
}

func isFresh(t target, opts genOptions) (bool, error) {
	code, err := generate(t.pkg, opts)
	if err != nil {
		return false, err
	}

	current, err := os.ReadFile(t.path)

	return err == nil && string(current) == code, nil
}
