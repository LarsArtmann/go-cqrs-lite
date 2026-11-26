package main

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	cmdguard "github.com/larsartmann/cmdguard/v5/pkg/cmdguard/v5"
)

// cmdguardFlags are the flags cmdguard adds to every command; they are not
// part of cqrs-bench's own surface.
var cmdguardFlags = []string{"help", "config", "log-format", "log-level", "no-color"}

// describedFlag is one flag of the `describe` contract.
type describedFlag struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
	Usage  string   `json:"usage"`
}

// describedCommand is one command of the `describe` contract.
type describedCommand struct {
	Name  string          `json:"name"`
	Flags []describedFlag `json:"flags"`
}

// describe runs `cqrs-bench describe` and indexes the contract by command.
func describe(t *testing.T) map[string][]describedFlag {
	t.Helper()

	run := mustRunBench(t, t.TempDir(), "describe")

	var contract struct {
		Commands []describedCommand `json:"commands"`
	}

	if err := json.Unmarshal([]byte(run.stdout), &contract); err != nil {
		t.Fatalf("invalid describe contract: %v\n%s", err, run.stdout)
	}

	byName := make(map[string][]describedFlag, len(contract.Commands))
	for _, command := range contract.Commands {
		byName[command.Name] = command.Flags
	}

	return byName
}

// ownFlags are a command's flags without the cmdguard additions.
func ownFlags(flags []describedFlag) []string {
	names := make([]string, 0, len(flags))
	for _, flag := range flags {
		if !slices.Contains(cmdguardFlags, flag.Name) {
			names = append(names, flag.Name)
		}
	}

	slices.Sort(names)

	return names
}

// flagNames lists the `flag:"..."` tags of a flag struct and its embedded
// benchFlags, in declaration order.
func flagNames(structType reflect.Type) []string {
	names := []string{}

	for _, field := range reflect.VisibleFields(structType) {
		if name := field.Tag.Get("flag"); name != "" {
			names = append(names, name)
		}
	}

	return names
}

// TestCLI_DescribeMatchesTheFlagStructs pins the machine-readable contract
// to the code: every command exposes exactly the flags its struct declares.
func TestCLI_DescribeMatchesTheFlagStructs(t *testing.T) {
	t.Parallel()

	contract := describe(t)

	want := map[string][]string{
		"run":        flagNames(reflect.TypeFor[runFlags]()),
		"compare":    flagNames(reflect.TypeFor[compareFlags]()),
		"sweep":      flagNames(reflect.TypeFor[sweepFlags]()),
		"layout":     flagNames(reflect.TypeFor[layoutFlags]()),
		"list-phases": nil,
	}

	for command, flags := range want {
		slices.Sort(flags)

		if got := ownFlags(contract[command]); !slices.Equal(got, flags) {
			t.Errorf("%s flags = %q, want %q", command, got, flags)
		}
	}
}

// TestCLI_EnumFlagsDescribeTheirValues: a flag with a closed vocabulary
// publishes it, so agents and shells can complete it.
func TestCLI_EnumFlagsDescribeTheirValues(t *testing.T) {
	t.Parallel()

	contract := describe(t)

	cases := []struct {
		command string
		flag    string
		values  []string
	}{
		{"run", "profile", profileNames()},
		{"run", "codec", []string{"json", "cbor"}},
		{"run", "durability", []string{"strict", "normal", "relaxed"}},
		{"layout", "format", layoutFormats()},
	}

	for _, tc := range cases {
		idx := slices.IndexFunc(contract[tc.command], func(flag describedFlag) bool {
			return flag.Name == tc.flag
		})
		if idx < 0 {
			t.Errorf("%s has no --%s flag", tc.command, tc.flag)

			continue
		}

		if got := contract[tc.command][idx].Values; !slices.Equal(got, tc.values) {
			t.Errorf("%s --%s values = %q, want %q", tc.command, tc.flag, got, tc.values)
		}
	}
}

// TestBackendHelpListsEveryBackend: the --backend help text is written by
// hand, so it is pinned to the registry.
func TestBackendHelpListsEveryBackend(t *testing.T) {
	t.Parallel()

	field, ok := reflect.TypeFor[benchFlags]().FieldByName("Backend")
	if !ok {
		t.Fatal("benchFlags has no Backend field")
	}

	help := field.Tag.Get("help")

	for _, name := range backendNames() {
		if !strings.Contains(help, name) {
			t.Errorf("--backend help does not list %q: %s", name, help)
		}
	}

	for _, spec := range backendRegistry() {
		for _, alias := range spec.aliases {
			if !strings.Contains(help, alias) {
				t.Errorf("--backend help does not list the alias %q of %q: %s", alias, spec.name, help)
			}
		}
	}
}

// TestRootConfigKeepsCmdguardConfigFileFlags: the root config only carries
// cmdguard's own config-file flags.
func TestRootConfigKeepsCmdguardConfigFileFlags(t *testing.T) {
	t.Parallel()

	if got := flagNames(reflect.TypeFor[appConfig]()); !slices.Equal(got, flagNames(reflect.TypeFor[cmdguard.Config]())) {
		t.Errorf("appConfig flags = %q, want exactly cmdguard.Config's %q", got, flagNames(reflect.TypeFor[cmdguard.Config]()))
	}
}
