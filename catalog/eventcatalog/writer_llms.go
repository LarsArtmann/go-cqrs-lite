package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func (e *Exporter) writeLLMsTxt(cat *catalog.Catalog) error {
	var buf strings.Builder

	fmt.Fprintf(&buf, "# %s\n\n", cat.Title)
	buf.WriteString("> Auto-generated catalog summary for LLM consumption.\n\n")

	for _, svc := range cat.Services {
		writeLLMsTxtService(&buf, svc)
	}

	for _, ch := range cat.Channels {
		writeLLMsTxtChannel(&buf, ch)
	}

	for _, ds := range cat.DataStores {
		writeLLMsTxtDataStore(&buf, ds)
	}

	for _, f := range cat.Flows {
		writeLLMsTxtFlow(&buf, f)
	}

	for _, team := range cat.Teams {
		writeLLMsTxtTeam(&buf, team)
	}

	for _, user := range cat.Users {
		writeLLMsTxtUser(&buf, user)
	}

	for _, domain := range cat.Domains {
		writeLLMsTxtDomain(&buf, domain)
	}

	for _, entity := range cat.Entities {
		writeLLMsTxtEntity(&buf, entity)
	}

	for _, dp := range cat.DataProducts {
		writeLLMsTxtDataProduct(&buf, dp)
	}

	for _, agent := range cat.Agents {
		writeLLMsTxtAgent(&buf, agent)
	}

	return os.WriteFile( //nolint:wrapcheck // os.WriteFile returns direct error
		filepath.Join(e.outputDir, "llms.txt"),
		[]byte(buf.String()),
		filePerm,
	)
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
	fmt.Fprintf(buf, "## Channel: %s (%s)\n", ch.Name, ch.ID)

	if ch.Summary != "" {
		fmt.Fprintf(buf, "%s\n", ch.Summary)
	}

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
	fmt.Fprintf(buf, "## Flow: %s (%s)\n", f.Name, f.ID)

	if f.Summary != "" {
		fmt.Fprintf(buf, "%s\n", f.Summary)
	}

	fmt.Fprintf(buf, "Steps: %d\n\n", len(f.Steps))
}

func writeLLMsTxtTeam(buf *strings.Builder, team catalog.Team) {
	fmt.Fprintf(buf, "## Team: %s (%s)\n", team.Name, team.ID)

	if team.Summary != "" {
		fmt.Fprintf(buf, "%s\n", team.Summary)
	}

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
	fmt.Fprintf(buf, "## Domain: %s (%s)\n", domain.Name, domain.ID)

	if domain.Summary != "" {
		fmt.Fprintf(buf, "%s\n", domain.Summary)
	}

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
	fmt.Fprintf(buf, "## Entity: %s (%s)\n", entity.Name, entity.ID)

	if entity.Summary != "" {
		fmt.Fprintf(buf, "%s\n", entity.Summary)
	}

	buf.WriteString("\n")
}

func writeLLMsTxtDataProduct(buf *strings.Builder, dp catalog.DataProduct) {
	fmt.Fprintf(buf, "## Data Product: %s (%s)\n", dp.Name, dp.ID)

	if dp.Summary != "" {
		fmt.Fprintf(buf, "%s\n", dp.Summary)
	}

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
