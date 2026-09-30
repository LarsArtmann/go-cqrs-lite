package eventcatalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

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
