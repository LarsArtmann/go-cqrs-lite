package eventcatalog

import (
	"encoding/json"
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

type frontmatterWriter struct {
	*strings.Builder
}

func newFrontmatterWriter() *frontmatterWriter {
	f := &frontmatterWriter{Builder: new(strings.Builder)}
	_, _ = f.WriteString("---\n")

	return f
}

func (f *frontmatterWriter) addField(key, value string) {
	_, _ = fmt.Fprintf(f, "%s: %s\n", key, value)
}

func (f *frontmatterWriter) addQuotedField(key, value string) {
	_, _ = fmt.Fprintf(f, "%s: %q\n", key, value)
}

func (f *frontmatterWriter) addListField(key string, values []string) {
	if len(values) == 0 {
		return
	}

	_, _ = fmt.Fprintf(f, "%s:\n", key)

	for _, v := range values {
		_, _ = fmt.Fprintf(f, "  - %s\n", v)
	}
}

func (f *frontmatterWriter) addObjectIDsListField(key string, ids []string) {
	if len(ids) == 0 {
		return
	}

	_, _ = fmt.Fprintf(f, "%s:\n", key)

	for _, id := range ids {
		_, _ = fmt.Fprintf(f, "  - id: %s\n", id)
	}
}

func (f *frontmatterWriter) finish(title, summary string) {
	_, _ = f.WriteString("---\n\n")
	_, _ = fmt.Fprintf(f, "# %s\n\n%s\n", title, summary)
}

func (f *frontmatterWriter) finishWithGraph(title, summary string) {
	_, _ = f.WriteString("---\n\n")
	_, _ = fmt.Fprintf(f, "# %s\n\n%s\n\n<NodeGraph />\n", title, summary)
}

func (e *Exporter) writeMDXFile(path, content string) error {
	return os.WriteFile(path, []byte(content), filePerm) //nolint:wrapcheck
}

func (e *Exporter) writeSchema(dir string, schema *catalog.Schema) error {
	schemaDir := filepath.Join(dir, "schemas")

	err := os.MkdirAll(schemaDir, dirPerm)
	if err != nil {
		return fmt.Errorf("create schema dir %s in %s: %w", schemaDir, dir, err)
	}

	data, err := catalog.SchemaToJSON(schema)
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	return os.WriteFile(filepath.Join(schemaDir, "schema.json"), data, filePerm) //nolint:wrapcheck
}

func (e *Exporter) writeExamples(dir string, examples []json.RawMessage) error {
	if len(examples) == 0 {
		return nil
	}

	data, err := json.MarshalIndent(examples, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal examples: %w", err)
	}

	return os.WriteFile(filepath.Join(dir, "examples.json"), data, filePerm) //nolint:wrapcheck
}

func (e *Exporter) writeConfig(cat *catalog.Catalog) error {
	var cfg strings.Builder
	cfg.WriteString("/** @type {import('@eventcatalog/core/bin/eventcatalog.config').Config} */\n")
	cfg.WriteString("export default {\n")
	fmt.Fprintf(&cfg, "  title: %q,\n", cat.Title)
	fmt.Fprintf(&cfg, "  organizationName: %q,\n", cat.Title)
	cfg.WriteString("  landingPage: '',\n")
	cfg.WriteString("};\n")

	err := os.WriteFile(
		filepath.Join(e.outputDir, "eventcatalog.config.js"),
		[]byte(cfg.String()),
		filePerm,
	)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return e.writePackageJSON(cat)
}

func (e *Exporter) writePackageJSON(cat *catalog.Catalog) error {
	pkg := map[string]any{
		"type":        "module",
		"name":        strings.ToLower(strings.ReplaceAll(cat.Title, " ", "-")),
		"version":     cat.Version,
		"private":     true,
		"description": cat.Title + " event catalog",
		"dependencies": map[string]string{
			"@eventcatalog/core": "latest",
		},
	}

	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal package.json: %w", err)
	}

	return os.WriteFile( //nolint:wrapcheck
		filepath.Join(e.outputDir, "package.json"),
		data,
		filePerm,
	)
}

func writeIDListField[S ~string](md *frontmatterWriter, key string, ids []S) {
	if len(ids) == 0 {
		return
	}

	_, _ = fmt.Fprintf(md, "%s:\n", key)

	for _, id := range ids {
		_, _ = fmt.Fprintf(md, "  - id: %s\n", id)
	}
}

func writeBadges(md *frontmatterWriter, badges []catalog.Badge) {
	if len(badges) == 0 {
		return
	}

	_, _ = md.WriteString("badges:\n")

	for _, b := range badges {
		_, _ = fmt.Fprintf(md, "  - content: %q\n", b.Content)

		if b.BackgroundColor != "" {
			_, _ = fmt.Fprintf(md, "    backgroundColor: %s\n", b.BackgroundColor)
		}

		if b.TextColor != "" {
			_, _ = fmt.Fprintf(md, "    textColor: %s\n", b.TextColor)
		}

		if b.Icon != "" {
			_, _ = fmt.Fprintf(md, "    icon: %s\n", b.Icon)
		}
	}
}

func writeRepository(md *frontmatterWriter, repo *catalog.Repository) {
	if repo == nil {
		return
	}

	_, _ = md.WriteString("repository:\n")

	if repo.Language != "" {
		_, _ = fmt.Fprintf(md, "  language: %q\n", repo.Language)
	}

	if repo.URL != "" {
		_, _ = fmt.Fprintf(md, "  url: %q\n", repo.URL)
	}
}

func writeOperation(md *frontmatterWriter, op *catalog.Operation) {
	if op == nil {
		return
	}

	_, _ = md.WriteString("operation:\n")
	_, _ = fmt.Fprintf(md, "  method: %s\n", op.Method)
	_, _ = fmt.Fprintf(md, "  path: %q\n", op.Path)

	if len(op.StatusCodes) > 0 {
		_, _ = md.WriteString("  statusCodes:\n")

		for _, sc := range op.StatusCodes {
			_, _ = fmt.Fprintf(md, "    - %q\n", sc)
		}
	}
}

func writeSpecifications(md *frontmatterWriter, specs []catalog.Specification) {
	if len(specs) == 0 {
		return
	}

	_, _ = md.WriteString("specifications:\n")

	for _, s := range specs {
		_, _ = fmt.Fprintf(md, "  - type: %s\n", s.Type)
		_, _ = fmt.Fprintf(md, "    path: %q\n", s.Path)

		if s.Name != "" {
			_, _ = fmt.Fprintf(md, "    name: %q\n", s.Name)
		}
	}
}

func writeAttachments(md *frontmatterWriter, attachments []catalog.Attachment) {
	if len(attachments) == 0 {
		return
	}

	_, _ = md.WriteString("attachments:\n")

	for _, a := range attachments {
		_, _ = fmt.Fprintf(md, "  - url: %q\n", a.URL)

		if a.Title != "" {
			_, _ = fmt.Fprintf(md, "    title: %q\n", a.Title)
		}

		if a.Type != "" {
			_, _ = fmt.Fprintf(md, "    type: %q\n", a.Type)
		}
	}
}

func writeMessagePointers(md *frontmatterWriter, key string, ptrs []catalog.Ref) {
	if len(ptrs) == 0 {
		return
	}

	_, _ = fmt.Fprintf(md, "%s:\n", key)

	for _, p := range ptrs {
		_, _ = fmt.Fprintf(md, "  - id: %s\n", p.ID)

		if p.Version != "" {
			_, _ = fmt.Fprintf(md, "    version: %s\n", p.Version)
		}
	}
}
