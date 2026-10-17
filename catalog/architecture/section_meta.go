package architecture

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

// adrSection builds the section for one architecture decision record.
func adrSection(adr catalog.ADR, idx *refIndex) section {
	return section{
		kind:    kindADR,
		id:      string(adr.ID),
		name:    string(adr.Name),
		version: string(adr.Version),
		summary: string(adr.Summary),
		badges:  adr.Badges,
		meta: metaLine([]markup{
			textMeta(string(adr.Status)),
			ownerMeta(idx, adr.Owners),
			textMeta(adr.Date),
		}),
		left:  adrBlocks(adr, idx),
		right: adrRefs(adr, idx),
	}
}

// adrBlocks builds the left-column blocks of a decision record.
func adrBlocks(adr catalog.ADR, idx *refIndex) []block {
	blocks := make([]block, 0, 2)

	blocks = appendProse(blocks, "Decision", adr.Body, idx.link)

	if len(adr.DecisionMakers) > 0 {
		blocks = append(blocks, blockHTML("Decision makers", plainRefList(adr.DecisionMakers)))
	}

	return blocks
}

// adrRefs builds the relationship boxes of a decision record.
func adrRefs(adr catalog.ADR, idx *refIndex) []block {
	refs := make([]block, 0, 7)
	refs = appendRefs(refs, "Applies to", adrTargets(adr.AppliesTo, idx))
	refs = appendRefs(refs, "Supersedes", adrLinks(adr.Supersedes, idx))
	refs = appendRefs(refs, "Superseded by", adrLinks(adr.SupersededBy, idx))
	refs = appendRefs(refs, "Amends", adrLinks(adr.Amends, idx))
	refs = appendRefs(refs, "Amended by", adrLinks(adr.AmendedBy, idx))
	refs = appendRefs(refs, "Related", adrLinks(adr.Related, idx))
	refs = appendRefs(refs, "Owners", idx.owners(adr.Owners))

	return refs
}

// adrTargets resolves the resources a decision applies to.
func adrTargets(targets []catalog.ADRResourceRef, idx *refIndex) []resourceRef {
	refs := make([]resourceRef, 0, len(targets))

	for _, target := range targets {
		kind, ok := adrKind(string(target.Type))
		if !ok {
			refs = append(refs, resourceRef{name: string(target.Type) + " " + target.ID})

			continue
		}

		if ref := idx.get(kind, target.ID); ref.anchor != "" {
			refs = append(refs, ref)

			continue
		}

		refs = append(refs, resourceRef{name: target.ID})
	}

	return refs
}

// adrLinks resolves supersede/amend/related references.
func adrLinks(links []catalog.ADRRef, idx *refIndex) []resourceRef {
	refs := make([]resourceRef, 0, len(links))

	for _, link := range links {
		ref := idx.get(kindADR, string(link.ID))
		if ref.anchor == "" {
			ref = resourceRef{name: string(link.ID)}
		}

		refs = append(refs, ref)
	}

	return refs
}

// adrKind maps an ADR resource type to the resource kind it names.
func adrKind(resourceType string) (resourceKind, bool) {
	switch resourceType {
	case "service":
		return kindService, true
	case "event":
		return kindEvent, true
	case "command":
		return kindCommand, true
	case "query":
		return kindQuery, true
	case "flow":
		return kindFlow, true
	case "channel":
		return kindChannel, true
	case "domain":
		return kindDomain, true
	case "user":
		return kindUser, true
	case "team":
		return kindTeam, true
	case "container":
		return kindDataStore, true
	case "entity":
		return kindEntity, true
	case "data-product":
		return kindDataProduct, true
	case "agent":
		return kindAgent, true
	default:
		return "", false
	}
}

// agentSection builds the section for one agent.
func agentSection(agent catalog.Agent, idx *refIndex) section {
	meta := []markup{ownerMeta(idx, agent.Owners)}

	if agent.Model != nil {
		meta = append(meta, textMeta(string(agent.Model.Provider)+" "+string(agent.Model.Name)))
	}

	return section{
		kind:    kindAgent,
		id:      string(agent.ID),
		name:    string(agent.Name),
		version: string(agent.Version),
		summary: string(agent.Summary),
		badges:  agent.Badges,
		meta:    metaLine(meta),
		left:    agentBlocks(agent),
		right:   agentRefs(agent, idx),
	}
}

// agentBlocks builds the left-column blocks of an agent section.
func agentBlocks(agent catalog.Agent) []block {
	blocks := make([]block, 0, 2)

	if model := agentModelFields(agent.Model); model != "" {
		blocks = append(blocks, blockHTML("Model", model))
	}

	if tools := agentToolTable(agent.Tools); tools != "" {
		blocks = append(blocks, blockHTML("Tools", tools))
	}

	return blocks
}

// agentRefs builds the relationship boxes of an agent section.
func agentRefs(agent catalog.Agent, idx *refIndex) []block {
	refs := make([]block, 0, 6)
	refs = appendRefs(refs, "Sends", idx.anyOf(refIDs(agent.Sends)))
	refs = appendRefs(refs, "Receives", idx.anyOf(refIDs(agent.Receives)))
	refs = appendRefs(refs, "Writes to", idx.messagesOf(kindDataStore, idStrings(agent.WritesTo)))
	refs = appendRefs(refs, "Reads from", idx.messagesOf(kindDataStore, idStrings(agent.ReadsFrom)))
	refs = appendRefs(refs, "Flows", idx.messagesOf(kindFlow, idStrings(agent.Flows)))
	refs = appendRefs(refs, "Owners", idx.owners(agent.Owners))

	return refs
}

// agentModelFields describes the model an agent runs on.
func agentModelFields(model *catalog.AgentModel) markup {
	if model == nil {
		return ""
	}

	return fieldList([][2]string{
		{"Provider", string(model.Provider)},
		{"Model", string(model.Name)},
		{"Version", string(model.Version)},
	})
}

// agentToolTable renders the tools an agent may call.
func agentToolTable(tools []catalog.AgentTool) markup {
	if len(tools) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		w.open("div", htmlClass("md-table"))
		w.open("table")
		w.open("thead")
		w.open("tr")

		for _, header := range []string{"Tool", "Type", "Description"} {
			w.element("th", header)
		}

		w.close("tr")
		w.close("thead")
		w.open("tbody")

		for _, tool := range tools {
			w.open("tr")
			w.element("td", string(tool.Name), htmlClass("mono"))
			w.element("td", tool.Type)
			w.element("td", string(tool.Description))
			w.close("tr")
		}

		w.close("tbody")
		w.close("table")
		w.close("div")
	})
}

// teamSection builds the section for one team.
func teamSection(team catalog.Team, idx *refIndex) section {
	return section{
		kind:    kindTeam,
		id:      string(team.ID),
		name:    string(team.Name),
		summary: string(team.Summary),
		meta:    metaLine([]markup{countMeta(len(team.Members), "member", "members"), textMeta(string(team.Role))}),
		left:    orgBlocks(string(team.Email), string(team.SlackDirectMessageURL)),
		right:   appendRefs([]block{}, "Members", idx.owners(team.Members)),
	}
}

// userSection builds the section for one person.
func userSection(user catalog.User, idx *refIndex) section {
	return section{
		kind:    kindUser,
		id:      string(user.ID),
		name:    string(user.Name),
		summary: "",
		meta:    metaLine([]markup{textMeta(string(user.Role))}),
		left:    orgBlocks(string(user.Email), string(user.SlackDirectMessageURL)),
		right:   []block{},
	}
}

// orgBlocks renders the contact details shared by teams and people.
func orgBlocks(email, slack string) []block {
	fields := fieldList([][2]string{
		{"Email", email},
		{"Slack", slack},
	})

	if fields == "" {
		return nil
	}

	return []block{blockHTML("Contact", fields)}
}

// docSection builds the section for one custom documentation page.
func docSection(doc catalog.CustomDoc, idx *refIndex) section {
	blocks := appendProse(make([]block, 0, 1), "", doc.Content, idx.link)

	return section{
		kind:    kindDoc,
		id:      string(doc.ID),
		name:    string(doc.Title),
		summary: string(doc.Summary),
		badges:  doc.Badges,
		meta:    metaLine([]markup{ownerMeta(idx, doc.Owners)}),
		left:    blocks,
		right:   appendRefs([]block{}, "Owners", idx.owners(doc.Owners)),
	}
}
