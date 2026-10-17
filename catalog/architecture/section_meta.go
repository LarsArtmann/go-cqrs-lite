package architecture

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

// adrSection builds the section for one architecture decision record.
func adrSection(adr catalog.ADR, idx *refIndex) section {
	sec := newSection(
		kindADR,
		string(adr.ID),
		string(adr.Name),
	)

	sec.version = string(adr.Version)
	sec.summary = string(adr.Summary)

	sec.badges = adr.Badges
	sec.meta = metaLine(
		[]markup{textMeta(string(adr.Status)), ownerMeta(idx, adr.Owners), textMeta(adr.Date)},
	)

	sec.left = adrBlocks(adr, idx)
	sec.right = adrRefs(adr, idx)

	return sec
}

// adrBlocks builds the left-column blocks of a decision record.
func adrBlocks(adr catalog.ADR, idx *refIndex) []block {
	blocks := make([]block, 0)

	blocks = appendProse(blocks, "Decision", adr.Body, idx.link)

	if len(adr.DecisionMakers) > 0 {
		blocks = append(blocks, blockHTML("Decision makers", plainRefList(adr.DecisionMakers)))
	}

	return blocks
}

// adrRefs builds the relationship boxes of a decision record.
func adrRefs(adr catalog.ADR, idx *refIndex) []block {
	refs := make([]block, 0)
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
			refs = append(refs, plainRef(string(target.Type)+" "+target.ID))

			continue
		}

		if ref := idx.get(kind, target.ID); ref.anchor != "" {
			refs = append(refs, ref)

			continue
		}

		refs = append(refs, plainRef(target.ID))
	}

	return refs
}

// adrLinks resolves supersede/amend/related references.
func adrLinks(links []catalog.ADRRef, idx *refIndex) []resourceRef {
	refs := make([]resourceRef, 0, len(links))

	for _, link := range links {
		ref := idx.get(kindADR, string(link.ID))
		if ref.anchor == "" {
			ref = plainRef(string(link.ID))
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

	sec := newSection(
		kindAgent,
		string(agent.ID),
		string(agent.Name),
	)

	sec.version = string(agent.Version)
	sec.summary = string(agent.Summary)

	sec.badges = agent.Badges
	sec.meta = metaLine(meta)

	sec.left = agentBlocks(agent)
	sec.right = agentRefs(agent, idx)

	return sec
}

// agentBlocks builds the left-column blocks of an agent section.
func agentBlocks(agent catalog.Agent) []block {
	blocks := make([]block, 0)

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
	refs := make([]block, 0)
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

		for _, header := range []string{"Tool", headerType, headerDescription} {
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
	sec := newSection(
		kindTeam,
		string(team.ID),
		string(team.Name),
	)

	sec.summary = string(team.Summary)
	sec.meta = metaLine(
		[]markup{countMeta(len(team.Members), "member", "members"), textMeta(string(team.Role))},
	)

	sec.left = orgBlocks(string(team.Email), string(team.SlackDirectMessageURL))
	sec.right = appendRefs([]block{}, "Members", idx.owners(team.Members))

	return sec
}

// userSection builds the section for one person.
func userSection(user catalog.User) section {
	sec := newSection(
		kindUser,
		string(user.ID),
		string(user.Name),
	)

	sec.summary = ""
	sec.meta = metaLine([]markup{textMeta(string(user.Role))})

	sec.left = orgBlocks(string(user.Email), string(user.SlackDirectMessageURL))
	sec.right = []block{}

	return sec
}

// orgBlocks renders the contact details shared by teams and people.
func orgBlocks(email, slack string) []block {
	if email == "" && slack == "" {
		return nil
	}

	return []block{blockHTML("Contact", fieldList([][2]string{
		{"Email", email},
		{"Slack", slack},
	}))}
}

// docSection builds the section for one custom documentation page.
func docSection(doc catalog.CustomDoc, idx *refIndex) section {
	blocks := appendProse(make([]block, 0), "", doc.Content, idx.link)

	sec := newSection(
		kindDoc,
		string(doc.ID),
		string(doc.Title),
	)

	sec.summary = string(doc.Summary)
	sec.badges = doc.Badges

	sec.meta = metaLine([]markup{ownerMeta(idx, doc.Owners)})
	sec.left = blocks

	sec.right = appendRefs([]block{}, "Owners", idx.owners(doc.Owners))

	return sec
}
