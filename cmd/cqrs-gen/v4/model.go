package main

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

type kind string

const (
	kindCommand kind = "command"
	kindQuery   kind = "query"
	kindEvent   kind = "event"
)

const (
	kindAll          = "all"
	targetSystem     = "system"
	targetDispatcher = "dispatcher"

	generatedFile = "cqrs_gen.go"
	markerPrefix  = "//cqrs:"

	optionSends    = "sends"
	optionReceives = "receives"

	prefixCommand = "Command"
	prefixQuery   = "Query"
	prefixEvent   = "Event"

	catalogFunc = "CatalogMessages"
)

type marker struct {
	kind        kind
	structName  string
	wire        string
	receives    bool
	byValue     bool
	constructor bool
	constName   string
	constExpr   string
	pos         string
}

type pkg struct {
	dir     string
	name    string
	markers []marker
}

type genOptions struct {
	target   string
	catalog  bool
	handlers bool
}

type target struct {
	path string
	pkg  pkg
}

func parseKinds(value string) (map[kind]bool, error) {
	if value == kindAll {
		return map[kind]bool{kindCommand: true, kindQuery: true, kindEvent: true}, nil
	}

	kinds := make(map[kind]bool)

	for name := range strings.SplitSeq(value, ",") {
		k := kind(name)
		if !isKind(k) {
			return nil, fmt.Errorf(
				"invalid --type %q: must be command, query, event, all, or a comma-separated list",
				value,
			)
		}

		kinds[k] = true
	}

	return kinds, nil
}

func parseTarget(value string) (string, error) {
	if value != targetSystem && value != targetDispatcher {
		return "", fmt.Errorf("invalid --target %q: must be system or dispatcher", value)
	}

	return value, nil
}

func isKind(k kind) bool {
	return k == kindCommand || k == kindQuery || k == kindEvent
}

func compareMarkers(a, b marker) int {
	if order := compareKinds(a.kind, b.kind); order != 0 {
		return order
	}

	return strings.Compare(a.structName, b.structName)
}

func compareKinds(a, b kind) int {
	switch {
	case a == b:
		return 0
	case a == kindCommand || (a == kindQuery && b == kindEvent):
		return -1
	default:
		return 1
	}
}

func constantName(k kind, structName string) string {
	return prefixOf(k) + trimKindSuffix(k, structName)
}

func prefixOf(k kind) string {
	switch k {
	case kindCommand:
		return prefixCommand
	case kindQuery:
		return prefixQuery
	case kindEvent:
		return prefixEvent
	}

	return ""
}

func trimKindSuffix(k kind, structName string) string {
	for _, suffix := range kindSuffixes(k) {
		if stem, ok := strings.CutSuffix(structName, suffix); ok && stem != "" {
			return stem
		}
	}

	return structName
}

func kindSuffixes(k kind) []string {
	switch k {
	case kindCommand:
		return []string{"Command", "Cmd"}
	case kindQuery:
		return []string{"Query"}
	case kindEvent:
		return []string{"Payload", "Event"}
	}

	return nil
}

func registerName(m *marker) string {
	return "Register" + m.structName
}

func handlerName(m *marker) string {
	return "Register" + m.structName + "Handler"
}

func eventName(m *marker) string {
	return "New" + m.structName + "Event"
}

func constructorName(m *marker) string {
	return "New" + m.structName
}

func (m *marker) valueType() string {
	if m.byValue {
		return m.structName
	}

	return "*" + m.structName
}

func (m *marker) reservedNames() []string {
	names := []string{registerName(m), handlerName(m), catalogFunc}

	switch {
	case m.kind == kindEvent:
		names = append(names, eventName(m))
	case m.constructor:
		names = append(names, constructorName(m))
	}

	return names
}

func typeName(k kind) string {
	return string(k) + ".Type"
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}

	for i, r := range value {
		isDigit := r >= '0' && r <= '9'

		if r == '_' || unicode.IsLetter(r) || (isDigit && i > 0) {
			continue
		}

		return false
	}

	return true
}

func uniqueSorted(values []string) []string {
	slices.Sort(values)

	return slices.Compact(values)
}
