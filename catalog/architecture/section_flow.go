package architecture

import (
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// flowSection builds the section for one flow: its steps, in order, each
// linked to the resources it touches.
func flowSection(flow catalog.Flow, idx *refIndex) section {
	return section{
		kind:    kindFlow,
		id:      string(flow.ID),
		name:    string(flow.Name),
		version: string(flow.Version),
		summary: string(flow.Summary),
		badges:  flow.Badges,
		meta: metaLine([]markup{
			ownerMeta(idx, flow.Owners),
			countMeta(len(flow.Steps), "step", "steps"),
		}),
		left: []block{
			blockHTML("Steps", flowSteps(flow.Steps, idx)),
		},
		right: flowRefs(flow, idx),
	}
}

// flowRefs builds the relationship boxes of a flow section: every resource
// the flow touches, gathered once.
func flowRefs(flow catalog.Flow, idx *refIndex) []block {
	touched := touchedRefs(flow, idx)

	refs := make([]block, 0, 3)

	if len(touched) > 0 {
		refs = append(refs, blockCounted("Involved", len(touched), refList(touched)))
	}

	return appendRefs(refs, "Owners", idx.owners(flow.Owners))
}

// touchedRefs collects the distinct resources a flow's steps reference.
func touchedRefs(flow catalog.Flow, idx *refIndex) []resourceRef {
	seen := make(map[string]struct{}, len(flow.Steps))

	refs := make([]resourceRef, 0, len(flow.Steps))

	for _, step := range flow.Steps {
		for _, ref := range stepRefs(step, idx) {
			if ref.anchor == "" {
				continue
			}

			if _, dup := seen[ref.anchor]; dup {
				continue
			}

			seen[ref.anchor] = struct{}{}

			refs = append(refs, ref)
		}
	}

	return refs
}

// stepRefs resolves the resources a single step points at.
func stepRefs(step catalog.FlowStep, idx *refIndex) []resourceRef {
	refs := make([]resourceRef, 0, 4)

	for _, target := range []struct {
		ref  *catalog.FlowStepRef
		kind resourceKind
	}{
		{step.Service, kindService},
		{step.Message, kindEvent},
		{step.Channel, kindChannel},
		{step.DataStore, kindDataStore},
		{step.DataProduct, kindDataProduct},
		{step.Agent, kindAgent},
		{step.SubFlow, kindFlow},
	} {
		if target.ref == nil {
			continue
		}

		if ref := idx.get(target.kind, string(target.ref.ID)); ref.anchor != "" {
			refs = append(refs, ref)

			continue
		}

		refs = append(refs, idx.any(string(target.ref.ID)))
	}

	return refs
}

// flowSteps renders a flow's steps as an ordered list.
func flowSteps(steps []catalog.FlowStep, idx *refIndex) markup {
	if len(steps) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		w.open("ol", htmlClass("steps"))

		for i, step := range steps {
			writeStep(w, i+1, step, idx)
		}

		w.close("ol")
	})
}

// writeStep renders one step with its resources and its outgoing edges.
func writeStep(w *htmlWriter, number int, step catalog.FlowStep, idx *refIndex) {
	w.open("li", htmlClass("step"))
	w.open("div", htmlClass("step-head"))
	w.element("span", strconv.Itoa(number), htmlClass("step-num"))
	w.element("span", string(step.Title), htmlClass("step-title"))
	w.close("div")

	if step.Summary != "" {
		w.element("p", string(step.Summary), htmlClass("step-summary"))
	}

	if refs := stepRefs(step, idx); len(refs) > 0 {
		w.open("div", htmlClass("step-refs"))
		w.raw(refList(refs))
		w.close("div")
	}

	if nodes := flowNodes(step); nodes != "" {
		w.container("p", nodes, htmlClass("step-nodes"))
	}

	if edges := flowingEdges(step); edges != "" {
		w.element("p", "Next: "+edges, htmlClass("step-edges"))
	}

	w.close("li")
}

// flowNodes renders the non-catalog participants of a step (actors and
// custom nodes), which have no section to link to.
func flowNodes(step catalog.FlowStep) markup {
	nodes := make([]string, 0, 3)

	for _, node := range []*catalog.FlowActor{step.Actor, step.External} {
		if node != nil && node.Name != "" {
			nodes = append(nodes, string(node.Name))
		}
	}

	if step.Custom != nil && step.Custom.Title != "" {
		nodes = append(nodes, string(step.Custom.Title))
	}

	if len(nodes) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) { w.text(joinIDs(nodes)) })
}

// flowingEdges renders a step's outgoing edge labels.
func flowingEdges(step catalog.FlowStep) string {
	labels := make([]string, 0, len(step.NextSteps)+1)

	if step.NextStep != nil {
		labels = append(labels, edgeLabel(*step.NextStep))
	}

	for _, edge := range step.NextSteps {
		labels = append(labels, edgeLabel(edge))
	}

	return joinIDs(labels)
}

// edgeLabel returns an edge's label, defaulting to its target id.
func edgeLabel(edge catalog.FlowEdge) string {
	if edge.Label != "" {
		return edge.Label
	}

	return string(edge.ID)
}
