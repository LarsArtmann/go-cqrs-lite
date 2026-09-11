package lintutil

import "strings"

// Name-heuristic primitives shared by the rules that classify structs by name
// or file location (C013 payload/view candidates, C035 read-model candidates,
// the F-series adoption rules). Rules compose these into their own
// strong/weak split; the functions below are the single source of the name
// and file-name vocabularies.

// HasEventPayloadNameSuffix reports whether the struct name itself carries an
// event-payload suffix (EVENT, PAYLOAD, EVENTDATA). This is the STRONG C013
// candidate signal: it never needs typed confirmation.
func HasEventPayloadNameSuffix(structName string) bool {
	upper := strings.ToUpper(structName)

	for _, suffix := range []string{"EVENT", "PAYLOAD", "EVENTDATA"} {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}

	return false
}

// IsPayloadFileName reports whether the file base name (events, payloads)
// suggests the struct is an event payload. This is an AMBIENT signal — the
// struct name itself says nothing — so under the typed-confirmation tier it
// needs structural evidence before a rule may fire on it.
func IsPayloadFileName(filePath string) bool {
	base := BaseFileName(filePath)

	return base == "events" || base == "payloads"
}

// HasReadModelNameSuffix reports whether the struct name carries an explicit
// read-model suffix (VIEW, READMODEL, READMODELSTATE, PROJECTION). The strong
// candidate signal for the C013 view branch and C035.
func HasReadModelNameSuffix(structName string) bool {
	upper := strings.ToUpper(structName)

	for _, suffix := range []string{"VIEW", "READMODEL", "READMODELSTATE", "PROJECTION"} {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}

	return false
}

// IsReadModelFileName reports whether the file base name (views, projection,
// readmodel) suggests the struct is a read model. Ambient signal — the weak
// counterpart of HasReadModelNameSuffix.
func IsReadModelFileName(filePath string) bool {
	base := BaseFileName(filePath)

	return base == "views" || base == "projection" || base == "readmodel"
}
