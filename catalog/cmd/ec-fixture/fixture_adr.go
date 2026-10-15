package main

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// fixtureADRIDs are the two decision records of the fixture: a superseded
// hand-written-docs decision and its replacement, so the supersedes/
// supersededBy pointer pair renders with a real counterpart on both ends.
const (
	fixtureADRSupersededID = "document-architecture-by-hand"
	fixtureADRCurrentID    = "emit-catalog-from-code"
)

// fixtureADRs builds the demo decision records. Every ADR frontmatter field
// class EventCatalog renders appears at least once: status, date, owners,
// decisionMakers, appliesTo (typed + version-pinned), supersedes,
// supersededBy, related, badges, and a Context/Decision/Consequences body.
func fixtureADRs() []catalog.ADR {
	return []catalog.ADR{
		{
			ID:      fixtureADRCurrentID,
			Name:    "Emit the architecture catalog from code",
			Version: fixtureVersion,
			Status:  catalog.ADRAccepted,
			Date:    "2026-09-24",
			Summary: "Catalog resources are declared next to the code they describe",
			Owners:  []string{fixtureTeamID},
			DecisionMakers: []string{
				"alice",
			},
			AppliesTo: []catalog.ADRResourceRef{
				{Type: catalog.ADRAppliesToService, ID: fixtureServiceID, Version: fixtureVersion},
				{Type: catalog.ADRAppliesToEvent, ID: fixtureEventID},
			},
			Supersedes: []catalog.ADRRef{{ID: fixtureADRSupersededID}},
			Related:    []catalog.ADRRef{{ID: fixtureADRSupersededID}},
			Badges: []catalog.Badge{
				{Content: "Accepted", BackgroundColor: "green", TextColor: "white"},
			},
			Body: "## Context\n\nArchitecture docs drifted from the services they describe.\n\n" +
				"## Decision\n\nServices declare their catalog resources in code; the exporter emits them.\n\n" +
				"## Consequences\n\nDocs stay current because they build with the code.",
		},
		{
			ID:      fixtureADRSupersededID,
			Name:    "Document the architecture by hand",
			Version: fixtureVersion,
			Status:  catalog.ADRSuperseded,
			Date:    "2026-08-01",
			Summary: "Superseded by emitting the catalog from code",
			Owners:  []string{fixtureTeamID},
			AppliesTo: []catalog.ADRResourceRef{
				{Type: catalog.ADRAppliesToDomain, ID: "ordering"},
			},
			SupersededBy: []catalog.ADRRef{{ID: fixtureADRCurrentID}},
			Body:         "## Decision\n\nHand-written markdown pages.\n\n## Consequences\n\nThey drifted.",
		},
	}
}
