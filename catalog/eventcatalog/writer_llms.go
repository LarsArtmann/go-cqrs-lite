package eventcatalog

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func (e *Exporter) writeLLMsTxt(cat *catalog.Catalog) error {
	return e.writeBuilderFile("llms.txt", func(buf *strings.Builder) {
		fmt.Fprintf(buf, "# %s\n\n", cat.Title)
		buf.WriteString("> Auto-generated catalog summary for LLM consumption.\n\n")

		for _, svc := range cat.Services {
			writeLLMsTxtService(buf, svc)
		}

		for _, ch := range cat.Channels {
			writeLLMsTxtChannel(buf, ch)
		}

		for _, ds := range cat.DataStores {
			writeLLMsTxtDataStore(buf, ds)
		}

		for _, f := range cat.Flows {
			writeLLMsTxtFlow(buf, f)
		}

		for _, team := range cat.Teams {
			writeLLMsTxtTeam(buf, team)
		}

		for _, user := range cat.Users {
			writeLLMsTxtUser(buf, user)
		}

		for _, domain := range cat.Domains {
			writeLLMsTxtDomain(buf, domain)
		}

		for _, entity := range cat.Entities {
			writeLLMsTxtEntity(buf, entity)
		}

		for _, dp := range cat.DataProducts {
			writeLLMsTxtDataProduct(buf, dp)
		}

		for _, agent := range cat.Agents {
			writeLLMsTxtAgent(buf, agent)
		}
	})
}

func writeLLMsTxtService(buf *strings.Builder, svc catalog.Service) {
	fmt.Fprintf(buf, "## %s (%s)\n", svc.Name, svc.ID)

	if svc.Summary != "" {
		fmt.Fprintf(buf, "%s\n", svc.Summary)
	}

	writeLLMsTxtMessages(buf, "Commands", svc.Commands)
	writeLLMsTxtEvents(buf, svc.Events)
	writeLLMsTxtMessages(buf, "Queries", svc.Queries)

	buf.WriteString("\n")
}

// writeLLMsSectionHeader writes the "## <kind>: <name> (<id>)" header followed
// by an optional summary line. Shared by every LLM-txt section writer so the
// header shape stays consistent and adjustable in one place.
func writeLLMsSectionHeader(buf *strings.Builder, kind, name, id, summary string) {
	fmt.Fprintf(buf, "## %s: %s (%s)\n", kind, name, id)
	if summary != "" {
		fmt.Fprintf(buf, "%s\n", summary)
	}
}

func writeLLMsTxtMessages(buf *strings.Builder, section string, msgs []catalog.Message) {
	if len(msgs) == 0 {
		return
	}

	fmt.Fprintf(buf, "\n### %s\n", section)

	for _, msg := range msgs {
		line := fmt.Sprintf("- %s (v%s): %s", msg.Name, msg.Version, msg.Summary)

		if msg.Operation != nil && msg.Operation.Method != "" {
			line += fmt.Sprintf(" [%s %s]", msg.Operation.Method, msg.Operation.Path)
		}

		buf.WriteString(line)
		buf.WriteString("\n")

		for _, resp := range msg.Responses {
			fmt.Fprintf(buf, "  - Response %s: %s\n", resp.StatusCode, resp.Description)
		}
	}
}

func writeLLMsTxtEvents(buf *strings.Builder, events []catalog.Message) {
	if len(events) == 0 {
		return
	}

	buf.WriteString("\n### Events\n")

	for _, evt := range events {
		dir := "receives"
		if evt.IsSend() {
			dir = "sends"
		}

		fmt.Fprintf(buf, "- %s (v%s) [%s]: %s\n", evt.Name, evt.Version, dir, evt.Summary)
	}
}

func writeLLMsTxtChannel(buf *strings.Builder, ch catalog.Channel) {
	writeLLMsSectionHeader(buf, "Channel", string(ch.Name), string(ch.ID), string(ch.Summary))

	if len(ch.Protocols) > 0 {
		protocols := make([]string, len(ch.Protocols))
		for i, p := range ch.Protocols {
			protocols[i] = string(p)
		}
		fmt.Fprintf(buf, "Protocols: %s\n", strings.Join(protocols, ", "))
	}

	buf.WriteString("\n")
}

func writeLLMsTxtDataStore(buf *strings.Builder, ds catalog.DataStore) {
	fmt.Fprintf(buf, "## Data Store: %s (%s)\n", ds.Name, ds.ID)
	fmt.Fprintf(buf, "Type: %s\n", ds.ContainerType)

	if ds.Technology != "" {
		fmt.Fprintf(buf, "Technology: %s\n", ds.Technology)
	}

	if ds.Summary != "" {
		fmt.Fprintf(buf, "%s\n", ds.Summary)
	}

	buf.WriteString("\n")
}

func writeLLMsTxtFlow(buf *strings.Builder, f catalog.Flow) {
	writeLLMsSectionHeader(buf, "Flow", string(f.Name), string(f.ID), string(f.Summary))

	fmt.Fprintf(buf, "Steps: %d\n\n", len(f.Steps))
}

func writeLLMsTxtTeam(buf *strings.Builder, team catalog.Team) {
	writeLLMsSectionHeader(buf, "Team", string(team.Name), string(team.ID), string(team.Summary))

	if len(team.Members) > 0 {
		fmt.Fprintf(buf, "Members: %s\n", strings.Join(team.Members, ", "))
	}

	buf.WriteString("\n")
}

func writeLLMsTxtUser(buf *strings.Builder, user catalog.User) {
	fmt.Fprintf(buf, "## User: %s (%s)\n", user.Name, user.ID)

	if user.Role != "" {
		fmt.Fprintf(buf, "Role: %s\n", user.Role)
	}

	buf.WriteString("\n")
}

func writeLLMsTxtDomain(buf *strings.Builder, domain catalog.Domain) {
	writeLLMsSectionHeader(
		buf,
		"Domain",
		string(domain.Name),
		string(domain.ID),
		string(domain.Summary),
	)

	if len(domain.Services) > 0 {
		ids := make([]string, len(domain.Services))
		for i, s := range domain.Services {
			ids[i] = string(s)
		}
		fmt.Fprintf(buf, "Services: %s\n", strings.Join(ids, ", "))
	}

	if len(domain.UbiquitousLanguage) > 0 {
		buf.WriteString("\nUbiquitous Language:\n")
		for _, term := range domain.UbiquitousLanguage {
			fmt.Fprintf(buf, "- %s: %s\n", term.Name, term.Description)
		}
	}

	buf.WriteString("\n")
}

func writeLLMsTxtEntity(buf *strings.Builder, entity catalog.Entity) {
	writeLLMsSectionHeader(
		buf,
		"Entity",
		string(entity.Name),
		string(entity.ID),
		string(entity.Summary),
	)

	buf.WriteString("\n")
}

func writeLLMsTxtDataProduct(buf *strings.Builder, dp catalog.DataProduct) {
	writeLLMsSectionHeader(buf, "Data Product", string(dp.Name), string(dp.ID), string(dp.Summary))

	buf.WriteString("\n")
}

func writeLLMsTxtAgent(buf *strings.Builder, agent catalog.Agent) {
	fmt.Fprintf(buf, "## Agent: %s (%s)\n", agent.Name, agent.ID)

	if agent.Summary != "" {
		fmt.Fprintf(buf, "%s\n", agent.Summary)
	}

	if len(agent.Sends) > 0 {
		fmt.Fprintf(buf, "Sends: %d messages\n", len(agent.Sends))
	}

	if len(agent.Receives) > 0 {
		fmt.Fprintf(buf, "Receives: %d messages\n", len(agent.Receives))
	}

	if agent.Model != nil {
		fmt.Fprintf(buf, "Model: %s/%s\n", agent.Model.Provider, agent.Model.Name)
	}

	if len(agent.Tools) > 0 {
		fmt.Fprintf(buf, "Tools: %d\n", len(agent.Tools))
	}

	buf.WriteString("\n")
}
