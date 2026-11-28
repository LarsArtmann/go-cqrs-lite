package search

import "strconv"

// Mode selects the kind of search. The zero value is [ModeAuto].
//
// Explicit modes are strict: asking for a kind the deployment cannot serve is
// [ErrModeUnavailable], never a silent step down, so a test cannot pass on
// lexical results when it meant to exercise another path.
type Mode uint8

const (
	// ModeAuto climbs the ladder as far as the deployment and the query
	// allow. It never fails for a missing capability.
	ModeAuto Mode = iota
	// ModeLexical serves full text when the engine offers it and scans
	// otherwise. It needs no capability.
	ModeLexical
	// ModeFullText requires a full-text index on the engine.
	ModeFullText
	// ModeVector requires a vector index and an embedder.
	ModeVector
	// ModeHybrid requires both a full-text index and an embedder.
	ModeHybrid
)

// String returns the lowercase name of the mode.
func (m Mode) String() string {
	if m > ModeHybrid {
		return "mode(" + strconv.Itoa(int(m)) + ")"
	}

	switch m {
	case ModeAuto:
		return "auto"
	case ModeLexical:
		return "lexical"
	case ModeFullText:
		return "fulltext"
	case ModeVector:
		return "vector"
	case ModeHybrid:
		return "hybrid"
	default:
		return "mode(" + strconv.Itoa(int(m)) + ")"
	}
}

// IsValid reports whether m is one of the declared modes.
func (m Mode) IsValid() bool {
	return m <= ModeHybrid
}

// Via names the tier that produced a [Result]. Client-side search (tier 0)
// never appears here: when it ran, no request reached the server.
type Via uint8

const (
	// ViaNone means no text search ran: the query was empty, too short, or
	// made of operator tokens only, and facets alone narrowed the list.
	ViaNone Via = iota
	// ViaScan is tier 1: the lexical scan, scored in Go.
	ViaScan
	// ViaFullText is tier 2: a full-text index.
	ViaFullText
	// ViaVector is tier 3: a vector index.
	ViaVector
	// ViaHybrid is tier 4: full-text and vector, fused by rank.
	ViaHybrid
)

// String returns the lowercase name of the tier.
func (v Via) String() string {
	if v > ViaHybrid {
		return "via(" + strconv.Itoa(int(v)) + ")"
	}

	switch v {
	case ViaNone:
		return "none"
	case ViaScan:
		return "scan"
	case ViaFullText:
		return "fulltext"
	case ViaVector:
		return "vector"
	case ViaHybrid:
		return "hybrid"
	default:
		return "via(" + strconv.Itoa(int(v)) + ")"
	}
}

// IsValid reports whether v is one of the declared tiers.
func (v Via) IsValid() bool {
	return v <= ViaHybrid
}

// IsText reports whether a text search ran at all.
func (v Via) IsText() bool {
	return v >= ViaScan && v <= ViaHybrid
}

// IsSemantic reports whether an embedder took part in the search.
func (v Via) IsSemantic() bool {
	return v == ViaVector || v == ViaHybrid
}
