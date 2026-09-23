package eventcatalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

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

// toChannelRefs converts channel IDs to EventCatalog channelPointer objects
// ({id, version?}). Plain strings fail schema validation
// ("Expected type object, received string").
func toChannelRefs(
	ids []catalog.ChannelID,
	versions map[catalog.ChannelID]catalog.Version,
) []channelRefFM {
	if len(ids) == 0 {
		return nil
	}

	out := make([]channelRefFM, len(ids))
	for i, id := range ids {
		ref := channelRefFM{ID: string(id)}
		if v, ok := versions[id]; ok {
			ref.Version = string(v)
		}
		out[i] = ref
	}

	return out
}

// EventCatalog's message schema declares producers/consumers as plain string
// references into the services collection. The DEFAULT form is the
// composite "<serviceID>-<serviceVersion>" Astro entry ID: @eventcatalog/core's
// content layer (and its visualiser graph edges) resolves exactly that key,
// and a bare service ID makes its build log `Invalid content reference`.
// The composite form is invisible to @eventcatalog/linter, which indexes by
// frontmatter ID — WithPlainRefIDs flips this function to bare IDs for
// governance/lint exports (see the option's doc comment).
// Unknown services fall back to the bare ID (the reference stays resolvable
// once a service with that ID exists).
func toServiceRefs(
	ids []catalog.ServiceID,
	versions map[catalog.ServiceID]catalog.Version,
	plain bool,
) []string {
	if len(ids) == 0 {
		return nil
	}

	out := make([]string, len(ids))
	for i, id := range ids {
		if !plain {
			if v, ok := versions[id]; ok && v != "" {
				out[i] = string(id) + "-" + string(v)

				continue
			}
		}

		out[i] = string(id)
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

// toBadges converts badges, filling EventCatalog's REQUIRED backgroundColor
// and textColor when the caller left them empty (the schema rejects badges
// without them: "backgroundColor: Required").
const (
	defaultBadgeBackgroundColor = "blue"
	defaultBadgeTextColor       = "white"
)

func toBadges(badges []catalog.Badge) []badgeFM {
	if len(badges) == 0 {
		return nil
	}

	out := make([]badgeFM, len(badges))
	for i, b := range badges {
		bg, fg := string(b.BackgroundColor), string(b.TextColor)
		if bg == "" {
			bg = defaultBadgeBackgroundColor
		}
		if fg == "" {
			fg = defaultBadgeTextColor
		}
		out[i] = badgeFM{
			Content:         b.Content,
			BackgroundColor: bg,
			TextColor:       fg,
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

func toResponses(specs []catalog.ResponseSpec) []responseFM {
	if len(specs) == 0 {
		return nil
	}

	out := make([]responseFM, len(specs))
	for i, s := range specs {
		out[i] = responseFM{
			StatusCode:  s.StatusCode,
			Description: s.Description,
		}
	}

	return out
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

// changelogBody renders message changelog entries as the markdown body of
// EventCatalog's changelog.(md|mdx) file. EventCatalog has NO changelog
// frontmatter field on messages — an inline list is rejected as an unknown
// property; the changelog is a separate file loaded by the changelogs
// collection (pattern "**/changelog.(md|mdx)").
func changelogBody(changes []catalog.Change) string {
	var b strings.Builder
	for _, c := range changes {
		entry := fmt.Sprintf("- **%s**: %s\n", string(c.Version), string(c.Summary))
		if c.Date != nil {
			entry = fmt.Sprintf(
				"- **%s** (%s): %s\n",
				string(c.Version),
				c.Date.Format(time.DateOnly),
				string(c.Summary),
			)
		}
		fmt.Fprint(&b, entry)
	}

	return b.String()
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

		if s.Actor != nil {
			// EventCatalog's actor shape is {name, summary} only — no url
			// (that field exists on externalSystem; emitting it on an actor is
			// silently stripped by zod).
			step.Actor = &flowActor{
				Name:    string(s.Actor.Name),
				Summary: string(s.Actor.Summary),
			}
		}

		if s.External != nil {
			step.ExternalSys = &flowActor{
				Name:    string(s.External.Name),
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

		// DataStore and SubFlow map onto EventCatalog's container and flow step
		// node keys. FlowStep.Channel has no EventCatalog flow equivalent (flow
		// steps support message/agent/service/flow/container/dataProduct/actor/
		// custom/externalSystem only), so channel steps are skipped here.
		if s.DataStore != nil {
			step.Container = &pointer{ID: s.DataStore.ID.String()}
		}

		if s.DataProduct != nil {
			step.DataProduct = &pointer{ID: s.DataProduct.ID.String()}
		}

		if s.SubFlow != nil {
			step.Flow = &pointer{ID: s.SubFlow.ID.String()}
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
