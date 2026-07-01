package eventcatalog

import (
	"fmt"
	"time"

	yaml "github.com/go-faster/yaml"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

// renderMDX marshals frontmatter to YAML and wraps it with the MDX body.
func renderMDX(fm any, title, summary string, includeGraph bool) (string, error) {
	data, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}

	body := fmt.Sprintf("---\n%s---\n\n# %s\n", string(data), title)

	if summary != "" {
		body += "\n" + summary + "\n"
	}

	if includeGraph {
		body += "\n<NodeGraph />\n"
	}

	return body, nil
}

// --- Conversion helpers: catalog types → frontmatter types ---

func toPointers[S ~string](ids []S) []pointer {
	if len(ids) == 0 {
		return nil
	}

	out := make([]pointer, len(ids))
	for i, id := range ids {
		out[i] = pointer{ID: string(id)}
	}

	return out
}

func toRefs(refs []catalog.Ref) []pointer {
	if len(refs) == 0 {
		return nil
	}

	out := make([]pointer, len(refs))
	for i, r := range refs {
		out[i] = pointer{ID: string(r.ID), Version: string(r.Version)}
	}

	return out
}

func toBadges(badges []catalog.Badge) []badgeFM {
	if len(badges) == 0 {
		return nil
	}

	out := make([]badgeFM, len(badges))
	for i, b := range badges {
		out[i] = badgeFM{
			Content:         b.Content,
			BackgroundColor: string(b.BackgroundColor),
			TextColor:       string(b.TextColor),
			Icon:            string(b.Icon),
			URL:             string(b.URL),
		}
	}

	return out
}

func toRepository(repo *catalog.Repository) *repositoryFM {
	if repo == nil {
		return nil
	}

	return &repositoryFM{
		Language: string(repo.Language),
		URL:      string(repo.URL),
	}
}

func toOperation(op *catalog.Operation) *operationFM {
	if op == nil {
		return nil
	}

	return &operationFM{
		Method:      string(op.Method),
		Path:        op.Path,
		StatusCodes: op.StatusCodes,
	}
}

func toSpecifications(specs []catalog.Specification) []specificationFM {
	if len(specs) == 0 {
		return nil
	}

	out := make([]specificationFM, len(specs))
	for i, s := range specs {
		out[i] = specificationFM{
			Type: s.Type,
			Path: s.Path,
			Name: string(s.Name),
		}
	}

	return out
}

func toAttachments(attachments []catalog.Attachment) []attachmentFM {
	if len(attachments) == 0 {
		return nil
	}

	out := make([]attachmentFM, len(attachments))
	for i, a := range attachments {
		out[i] = attachmentFM{
			URL:         string(a.URL),
			Title:       string(a.Title),
			Description: string(a.Description),
			Type:        a.Type,
			Icon:        string(a.Icon),
		}
	}

	return out
}

func toChangelog(changes []catalog.Change) []changeFM {
	if len(changes) == 0 {
		return nil
	}

	out := make([]changeFM, len(changes))
	for i, c := range changes {
		out[i] = changeFM{
			Version: string(c.Version),
			Summary: string(c.Summary),
		}

		if c.Date != nil {
			out[i].Date = c.Date.Format(time.DateOnly)
		}
	}

	return out
}

func toUbiquitousLanguage(terms []catalog.UbiquitousLanguageTerm) []ubiquitousLanguageTermFM {
	if len(terms) == 0 {
		return nil
	}

	out := make([]ubiquitousLanguageTermFM, len(terms))
	for i, t := range terms {
		out[i] = ubiquitousLanguageTermFM{
			Name:        string(t.Name),
			Description: t.Description,
		}
	}

	return out
}

func toAgentModel(model *catalog.AgentModel) *agentModelFM {
	if model == nil {
		return nil
	}

	return &agentModelFM{
		Provider: string(model.Provider),
		Name:     string(model.Name),
		Version:  string(model.Version),
	}
}

func toAgentTools(tools []catalog.AgentTool) []agentToolFM {
	if len(tools) == 0 {
		return nil
	}

	out := make([]agentToolFM, len(tools))
	for i, t := range tools {
		out[i] = agentToolFM{
			Name:        string(t.Name),
			Type:        t.Type,
			URL:         string(t.URL),
			Description: string(t.Description),
			Icon:        string(t.Icon),
		}
	}

	return out
}

func toChannelParams(params map[string]catalog.ChannelParam) map[string]channelParamFM {
	if len(params) == 0 {
		return nil
	}

	out := make(map[string]channelParamFM, len(params))
	for k, v := range params {
		out[k] = channelParamFM{
			Enum:        v.Enum,
			Default:     v.Default,
			Description: string(v.Description),
		}
	}

	return out
}

func toFlowSteps(steps []catalog.FlowStep) []flowStepFM {
	if len(steps) == 0 {
		return nil
	}

	out := make([]flowStepFM, len(steps))
	for i, s := range steps {
		step := flowStepFM{
			ID:      string(s.ID),
			Title:   string(s.Title),
			Summary: string(s.Summary),
		}

		if s.Service != nil {
			step.Service = &pointer{ID: s.Service.ID.String(), Version: string(s.Service.Version)}
		}

		if s.Message != nil {
			step.Message = &pointer{ID: s.Message.ID.String(), Version: string(s.Message.Version)}
		}

		if s.Channel != nil {
			step.Channel = &pointer{ID: s.Channel.ID.String()}
		}

		if s.Actor != nil {
			step.Actor = &flowActor{
				Name: string(
					s.Actor.Name,
				),
				Summary: string(s.Actor.Summary),
				URL:     string(s.Actor.URL),
			}
		}

		if s.External != nil {
			step.ExternalSys = &flowActor{
				Name: string(
					s.External.Name,
				),
				Summary: string(s.External.Summary),
				URL:     string(s.External.URL),
			}
		}

		if s.Custom != nil {
			step.Custom = &flowCustom{
				Title: string(s.Custom.Title),
				Icon:  string(s.Custom.Icon),
				Type:  s.Custom.Type,
				Summary: string(
					s.Custom.Summary,
				),
				URL:   string(s.Custom.URL),
				Color: string(s.Custom.Color),
			}
		}

		if s.Agent != nil {
			step.Agent = &pointer{ID: s.Agent.ID.String(), Version: string(s.Agent.Version)}
		}

		if s.DataStore != nil {
			step.DataStore = &pointer{ID: s.DataStore.ID.String()}
		}

		if s.DataProduct != nil {
			step.DataProduct = &pointer{ID: s.DataProduct.ID.String()}
		}

		if s.SubFlow != nil {
			step.SubFlow = &pointer{ID: s.SubFlow.ID.String()}
		}

		if s.NextStep != nil {
			step.NextStep = &flowEdgeFM{ID: string(s.NextStep.ID), Label: s.NextStep.Label}
		}

		if len(s.NextSteps) > 0 {
			step.NextSteps = make([]flowEdgeFM, len(s.NextSteps))
			for j, ns := range s.NextSteps {
				step.NextSteps[j] = flowEdgeFM{ID: string(ns.ID), Label: ns.Label}
			}
		}

		out[i] = step
	}

	return out
}

func toBaseConfig(b catalog.BaseConfig) baseConfigFM {
	var fm baseConfigFM
	if b.Sidebar != nil {
		fm.Sidebar = &sidebarFM{Badge: b.Sidebar.Badge, Label: b.Sidebar.Label}
	}
	if b.Styles != nil {
		fm.Styles = &stylesFM{Icon: b.Styles.Icon, NodeColor: b.Styles.NodeColor, NodeLabel: b.Styles.NodeLabel}
	}
	fm.EditUrl = b.EditUrl
	if b.Draft != nil {
		fm.Draft = &draftFM{Title: b.Draft.Title, Message: b.Draft.Message}
	}
	fm.Visualiser = b.Visualiser
	if len(b.ResourceGroups) > 0 {
		fm.ResourceGroups = make([]resourceGroupFM, len(b.ResourceGroups))
		for i, rg := range b.ResourceGroups {
			fm.ResourceGroups[i] = resourceGroupFM{
				ID: rg.ID, Title: rg.Title, Items: rg.Items, Limit: rg.Limit,
			}
		}
	}
	if b.DetailsPanel != nil {
		fm.DetailsPanel = &detailsPanelFM{Sections: b.DetailsPanel.Sections}
	}
	return fm
}

func toSource(s *catalog.Source) *sourceFM {
	if s == nil {
		return nil
	}

	return &sourceFM{
		Provider: s.Provider,
		ID:       s.ID,
		URL:      string(s.URL),
	}
}

func toDeprecated(deprecated bool, info *catalog.DeprecationInfo) any {
	if info != nil {
		fm := deprecationFM{Message: info.Message}
		if info.Date != nil {
			fm.Date = info.Date.Format(time.DateOnly)
		}

		return fm
	}

	if deprecated {
		return true
	}

	return nil
}

func channelIDsToStrings(ids []catalog.ChannelID) []string {
	if len(ids) == 0 {
		return nil
	}

	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}

	return out
}

func toDataProductOutputs(outputs []catalog.DataProductOutput) []dataProductOutputFM {
	if len(outputs) == 0 {
		return nil
	}

	out := make([]dataProductOutputFM, len(outputs))
	for i, o := range outputs {
		fm := dataProductOutputFM{
			ID:      string(o.ID),
			Version: string(o.Version),
		}

		if o.Contract != nil {
			fm.Contract = &dataContractFM{
				Path: o.Contract.Path,
				Name: string(o.Contract.Name),
				Type: o.Contract.Type,
			}
		}

		out[i] = fm
	}

	return out
}

func toSchemas(schemas []catalog.SchemaPointer) []schemaPointerFM {
	if len(schemas) == 0 {
		return nil
	}

	out := make([]schemaPointerFM, len(schemas))
	for i, s := range schemas {
		out[i] = schemaPointerFM{
			ID: s.ID, Ref: s.Ref, File: s.File, Path: s.Path,
			Name: string(s.Name), Format: s.Format, Default: s.Default,
		}
	}

	return out
}

func toEntityProperties(props []catalog.EntityProperty) []entityPropertyFM {
	if len(props) == 0 {
		return nil
	}

	out := make([]entityPropertyFM, len(props))
	for i, p := range props {
		out[i] = entityPropertyFM{
			Name:                 string(p.Name),
			Type:                 p.Type,
			Required:             p.Required,
			Description:          p.Description,
			References:           string(p.References),
			ReferencesIdentifier: p.ReferencesIdentifier,
			RelationType:         p.RelationType,
		}
	}

	return out
}

// --- Message ID collection (shared by service and agent) ---

func collectSendsReceives(svc catalog.Service) ([]pointer, []pointer) {
	sends := make([]pointer, 0, len(svc.Events))
	receives := make([]pointer, 0, len(svc.Events)+len(svc.Commands)+len(svc.Queries))

	for _, msg := range svc.Events {
		id := catalog.Key(msg)
		if msg.IsSend() {
			sends = append(sends, pointer{ID: string(id), Version: string(msg.Version)})
		} else {
			receives = append(receives, pointer{ID: string(id), Version: string(msg.Version)})
		}
	}

	for _, cmd := range svc.Commands {
		receives = append(
			receives,
			pointer{ID: string(catalog.Key(cmd)), Version: string(cmd.Version)},
		)
	}

	for _, q := range svc.Queries {
		receives = append(receives, pointer{ID: string(catalog.Key(q)), Version: string(q.Version)})
	}

	return sends, receives
}
