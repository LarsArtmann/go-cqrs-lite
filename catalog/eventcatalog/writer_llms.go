package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
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
		fmt.Fprintf(buf, "- %s (v%s): %s\n", msg.Name, msg.Version, msg.Summary)
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
		fmt.Fprintf(buf, "Protocols: %s\n", strings.Join(ch.Protocols, ", "))
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
