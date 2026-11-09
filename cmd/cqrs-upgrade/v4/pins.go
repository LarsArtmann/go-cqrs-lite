// Package main implements cqrs-upgrade: bump every go-cqrs-lite module pin in
// a consumer's go.mod to the latest published tag, verify the module still
// builds, and report usage of APIs removed at v5.
package main

import (
	"fmt"
	"strings"

	"golang.org/x/mod/modfile"
)

const modulePrefix = "github.com/larsartmann/go-cqrs-lite/"

// pin is one direct go-cqrs-lite requirement in the consumer's go.mod.
type pin struct {
	Module  string
	Current string
}

// collectPins parses the go.mod at path and returns its direct go-cqrs-lite
// requirements. Indirect pins are skipped: bumping them would promote them to
// direct entries — go mod tidy re-resolves them through the direct modules.
func collectPins(path string) ([]pin, error) {
	data, err := osReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	mod, err := modfile.Parse(path, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var pins []pin

	for _, req := range mod.Require {
		if !isCQRSPin(req) {
			continue
		}

		pins = append(pins, pin{Module: req.Mod.Path, Current: req.Mod.Version})
	}

	return pins, nil
}

// isCQRSPin reports whether the requirement is a direct go-cqrs-lite module.
func isCQRSPin(req *modfile.Require) bool {
	if req.Indirect {
		return false
	}

	return strings.HasPrefix(req.Mod.Path, modulePrefix)
}
