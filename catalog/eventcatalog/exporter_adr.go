package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	yaml "github.com/go-faster/yaml"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// adrStatuses and adrResourceTypes mirror the EventCatalog ADR schema
// exactly (verified against core 5.1.1's generated adrs collection schema,
// 2026-10-15). Values outside these sets fail the downstream build, so the
// exporter rejects them here with a precise message instead.
var adrStatuses = map[catalog.ADRStatus]struct{}{
	catalog.ADRProposed:   {},
	catalog.ADRAccepted:   {},
	catalog.ADRRejected:   {},
	catalog.ADRDeprecated: {},
	catalog.ADRSuperseded: {},
}

var adrResourceTypes = map[catalog.ADRResourceType]struct{}{
	catalog.ADRAppliesToAgent:       {},
	catalog.ADRAppliesToService:     {},
	catalog.ADRAppliesToEvent:       {},
	catalog.ADRAppliesToCommand:     {},
	catalog.ADRAppliesToQuery:       {},
	catalog.ADRAppliesToFlow:        {},
	catalog.ADRAppliesToChannel:     {},
	catalog.ADRAppliesToDomain:      {},
	catalog.ADRAppliesToSystem:      {},
	catalog.ADRAppliesToUser:        {},
	catalog.ADRAppliesToTeam:        {},
	catalog.ADRAppliesToContainer:   {},
	catalog.ADRAppliesToEntity:      {},
	catalog.ADRAppliesToDiagram:     {},
	catalog.ADRAppliesToDataProduct: {},
}

func validateADR(adr catalog.ADR) error {
	if adr.ID.IsZero() {
		return errorfamily.Newf(errorfamily.Rejection, "catalog.exporter_adr.2",
			"ADR with name %q: id is required (EventCatalog rejects id-less ADRs)", adr.Name)
	}

	if adr.Name == "" {
		return errorfamily.Newf(errorfamily.Rejection, "catalog.exporter_adr.2",
			"ADR %s: name is required (EventCatalog rejects name-less ADRs)", adr.ID)
	}

	if adr.Version == "" {
		return errorfamily.Newf(errorfamily.Rejection, "catalog.exporter_adr.2",
			"ADR %s: version is required (EventCatalog rejects version-less ADRs)", adr.ID)
	}

	if _, ok := adrStatuses[adr.Status]; !ok {
		return errorfamily.Newf(errorfamily.Rejection, "catalog.exporter_adr.2",
			"ADR %s: status %q is not one of proposed|accepted|rejected|deprecated|superseded", adr.ID, adr.Status)
	}

	if _, err := time.Parse(time.DateOnly, adr.Date); err != nil {
		return errorfamily.Newf(errorfamily.Rejection, "catalog.exporter_adr.2",
			"ADR %s: date %q must be a calendar date in YYYY-MM-DD form", adr.ID, adr.Date)
	}

	for _, ref := range adr.AppliesTo {
		if _, ok := adrResourceTypes[ref.Type]; !ok {
			return errorfamily.Newf(errorfamily.Rejection, "catalog.exporter_adr.2",
				"ADR %s: appliesTo type %q is not an EventCatalog resource kind "+
					"(agent|service|event|command|query|flow|channel|domain|system|"+
					"user|team|container|entity|diagram|data-product)", adr.ID, ref.Type)
		}
	}

	return nil
}

func (e *Exporter) writeADR(adr catalog.ADR) error {
	if err := validateADR(adr); err != nil {
		return err
	}

	dir := filepath.Join(e.outputDir, "adrs", string(adr.ID))

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_adr.1",
			"create adr dir: %v", err)
	}

	content, err := renderADRMDX(toADRFrontmatter(adr, e.plainRefIDs), string(adr.Name), adr.Body)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_adr.3",
			"render adr %s: %v", adr.ID, err)
	}

	return e.writeMDXFile(filepath.Join(dir, indexFile), content)
}

// renderADRMDX marshals ADR frontmatter to YAML and wraps it with the page
// body: the ADR name as heading, then the decision prose (Body).
func renderADRMDX(fm adrFM, name, body string) (string, error) {
	data, err := yaml.Marshal(fm)
	if err != nil {
		return "", errorfamily.WrapCorruption(err, "catalog.marshal_frontmatter",
			"marshal frontmatter to YAML")
	}

	content := fmt.Sprintf("---\n%s---\n\n# %s\n", string(data), name)

	if body != "" {
		content += "\n" + body + "\n"
	}

	return content, nil
}

func toADRFrontmatter(adr catalog.ADR, plainRefIDs bool) adrFM {
	return adrFM{
		ID:             string(adr.ID),
		Name:           string(adr.Name),
		Version:        string(adr.Version),
		Status:         string(adr.Status),
		Date:           adr.Date,
		Summary:        string(adr.Summary),
		Owners:         adr.Owners,
		DecisionMakers: adr.DecisionMakers,
		AppliesTo:      toADRAppliesTo(adr.AppliesTo, plainRefIDs),
		Supersedes:     toADRPointers(adr.Supersedes, plainRefIDs),
		SupersededBy:   toADRPointers(adr.SupersededBy, plainRefIDs),
		Amends:         toADRPointers(adr.Amends, plainRefIDs),
		AmendedBy:      toADRPointers(adr.AmendedBy, plainRefIDs),
		Related:        toADRPointers(adr.Related, plainRefIDs),
		Badges:         toBadges(adr.Badges),
	}
}

func toADRPointers(refs []catalog.ADRRef, plainRefIDs bool) []pointer {
	if len(refs) == 0 {
		return nil
	}

	out := make([]pointer, len(refs))
	for i, ref := range refs {
		out[i] = pointer{ID: string(ref.ID)}
		if !plainRefIDs {
			out[i].Version = string(ref.Version)
		}
	}

	return out
}

func toADRAppliesTo(refs []catalog.ADRResourceRef, plainRefIDs bool) []adrAppliesToFM {
	if len(refs) == 0 {
		return nil
	}

	out := make([]adrAppliesToFM, len(refs))
	for i, ref := range refs {
		out[i] = adrAppliesToFM{Type: string(ref.Type), ID: ref.ID}
		if !plainRefIDs {
			out[i].Version = string(ref.Version)
		}
	}

	return out
}
