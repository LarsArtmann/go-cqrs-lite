package architecture

// HideSection names a part of the reference a redacted export can suppress
// (ADR-0160). The vocabulary follows the model: a section with no catalog
// surface (contacts — no contact fields exist on any resource) is not a
// HideSection.
type HideSection string

const (
	// HideOwners suppresses every "Owned by …" meta item and owners block.
	HideOwners HideSection = "owners"
	// HideRepository suppresses the source-repository link blocks.
	HideRepository HideSection = "repository"
	// HideExamples suppresses message example payloads (the synthesised
	// fallback hides with them: an example is an example).
	HideExamples HideSection = "examples"
	// HideSpecifications suppresses service specification tables.
	HideSpecifications HideSection = "specifications"
	// HideDocs suppresses the custom documentation pages.
	HideDocs HideSection = "docs"
)

// visibility is the resolved hide policy. Every flag false is the default:
// the whole-catalog reference of ADR-0159.
type visibility struct {
	owners         bool
	repository     bool
	examples       bool
	specifications bool
	docs           bool
}

// visibilityFrom resolves the option list; unknown values are ignored so a
// renamed constant cannot silently change the export.
func visibilityFrom(sections []HideSection) visibility {
	out := visibility{
		owners:         false,
		repository:     false,
		examples:       false,
		specifications: false,
		docs:           false,
	}

	for _, section := range sections {
		switch section {
		case HideOwners:
			out.owners = true
		case HideRepository:
			out.repository = true
		case HideExamples:
			out.examples = true
		case HideSpecifications:
			out.specifications = true
		case HideDocs:
			out.docs = true
		case HideSection(""):
			// The zero value hides nothing.
		}
	}

	return out
}
