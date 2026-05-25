package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func (e *Exporter) writeChannel(ch catalog.Channel) error {
	dir := filepath.Join(e.outputDir, "channels", string(ch.ID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return fmt.Errorf("create channel dir: %w", err)
	}

	md := newFrontmatterWriter()
	md.addField("id", string(ch.ID))
	md.addField("name", ch.Name)
	md.addField("version", ch.Version)

	if ch.Summary != "" {
		md.addQuotedField("summary", ch.Summary)
	}

	if ch.Address != "" {
		md.addQuotedField("address", ch.Address)
	}

	md.addListField("protocols", ch.Protocols)

	if ch.DeliveryGuarantee != "" {
		md.addField("deliveryGuarantee", ch.DeliveryGuarantee)
	}

	writeChannelParams(md, ch.Parameters)

	if len(ch.Routes) > 0 {
		_, _ = md.WriteString("routes:\n")
		for _, r := range ch.Routes {
			_, _ = fmt.Fprintf(md, "  - id: %s\n", r.ID)
		}
	}

	md.addListField("owners", ch.Owners)
	writeBadges(md, ch.Badges)
	md.finishWithGraph(ch.Name, ch.Summary)

	return e.writeMDXFile(filepath.Join(dir, "index.mdx"), md.String())
}

func writeChannelParams(md *frontmatterWriter, params map[string]catalog.ChannelParam) {
	if len(params) == 0 {
		return
	}

	_, _ = md.WriteString("parameters:\n")

	for name, p := range params {
		_, _ = fmt.Fprintf(md, "  %s:\n", name)

		if len(p.Enum) > 0 {
			_, _ = md.WriteString("    enum:\n")
			for _, v := range p.Enum {
				_, _ = fmt.Fprintf(md, "      - %s\n", v)
			}
		}

		if p.Default != "" {
			_, _ = fmt.Fprintf(md, "    default: %s\n", p.Default)
		}

		if p.Description != "" {
			_, _ = fmt.Fprintf(md, "    description: %q\n", p.Description)
		}
	}
}

func (e *Exporter) writeDataStore(ds catalog.DataStore) error {
	dir := filepath.Join(e.outputDir, "data", ds.ID)

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return fmt.Errorf("create data store dir: %w", err)
	}

	md := newFrontmatterWriter()
	md.addField("id", ds.ID)
	md.addField("name", ds.Name)
	md.addField("version", ds.Version)
	md.addField("container_type", ds.ContainerType)

	if ds.Summary != "" {
		md.addQuotedField("summary", ds.Summary)
	}

	if ds.Technology != "" {
		md.addField("technology", ds.Technology)
	}

	if ds.Classification != "" {
		md.addField("classification", ds.Classification)
	}

	if ds.Retention != "" {
		md.addField("retention", ds.Retention)
	}

	if ds.Residency != "" {
		md.addField("residency", ds.Residency)
	}

	md.addListField("owners", ds.Owners)
	writeBadges(md, ds.Badges)
	md.finishWithGraph(ds.Name, ds.Summary)

	return e.writeMDXFile(filepath.Join(dir, "index.mdx"), md.String())
}

func (e *Exporter) writeFlow(f catalog.Flow) error {
	dir := filepath.Join(e.outputDir, "flows", f.ID)

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return fmt.Errorf("create flow dir: %w", err)
	}

	md := newFrontmatterWriter()
	md.addField("id", f.ID)
	md.addField("name", f.Name)
	md.addField("version", f.Version)

	if f.Summary != "" {
		md.addQuotedField("summary", f.Summary)
	}

	writeBadges(md, f.Badges)
	writeFlowSteps(md, f.Steps)
	md.finishWithGraph(f.Name, f.Summary)

	return e.writeMDXFile(filepath.Join(dir, "index.mdx"), md.String())
}

func writeFlowSteps(md *frontmatterWriter, steps []catalog.FlowStep) {
	if len(steps) == 0 {
		return
	}

	_, _ = md.WriteString("steps:\n")

	for _, s := range steps {
		_, _ = fmt.Fprintf(md, "  - id: %s\n", s.ID)
		_, _ = fmt.Fprintf(md, "    title: %q\n", s.Title)

		if s.Summary != "" {
			_, _ = fmt.Fprintf(md, "    summary: %q\n", s.Summary)
		}

		if s.Service != nil {
			_, _ = fmt.Fprintf(md, "    service:\n      id: %s\n", s.Service.ID)
			if s.Service.Version != "" {
				_, _ = fmt.Fprintf(md, "      version: %s\n", s.Service.Version)
			}
		}

		if s.Message != nil {
			_, _ = fmt.Fprintf(md, "    message:\n      id: %s\n", s.Message.ID)
			if s.Message.Version != "" {
				_, _ = fmt.Fprintf(md, "      version: %s\n", s.Message.Version)
			}
		}

		if s.Channel != nil {
			_, _ = fmt.Fprintf(md, "    channel:\n      id: %s\n", s.Channel.ID)
		}

		if s.Actor != nil {
			_, _ = fmt.Fprintf(md, "    actor:\n      name: %q\n", s.Actor.Name)
		}

		if s.External != nil {
			_, _ = fmt.Fprintf(md, "    externalSystem:\n      name: %q\n", s.External.Name)
		}

		if s.Custom != nil {
			_, _ = fmt.Fprintf(md, "    custom:\n      title: %q\n", s.Custom.Title)
		}

		if s.NextStep != nil {
			_, _ = fmt.Fprintf(md, "    next_step:\n      id: %s\n", s.NextStep.ID)
			if s.NextStep.Label != "" {
				_, _ = fmt.Fprintf(md, "      label: %q\n", s.NextStep.Label)
			}
		}

		if len(s.NextSteps) > 0 {
			_, _ = md.WriteString("    next_steps:\n")
			for _, ns := range s.NextSteps {
				_, _ = fmt.Fprintf(md, "      - id: %s\n", ns.ID)
				if ns.Label != "" {
					_, _ = fmt.Fprintf(md, "        label: %q\n", ns.Label)
				}
			}
		}
	}
}

func (e *Exporter) writeTeam(team catalog.Team) error {
	dir := filepath.Join(e.outputDir, "teams")

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return fmt.Errorf("create teams dir: %w", err)
	}

	md := newFrontmatterWriter()
	md.addField("id", team.ID)
	md.addField("name", team.Name)

	if team.Summary != "" {
		md.addQuotedField("summary", team.Summary)
	}

	md.addListField("members", team.Members)

	if team.Email != "" {
		md.addQuotedField("email", team.Email)
	}

	if team.SlackDirectMessageURL != "" {
		md.addQuotedField("slackDirectMessageUrl", team.SlackDirectMessageURL)
	}

	md.finish(team.Name, "")

	return e.writeMDXFile(filepath.Join(dir, team.ID+".mdx"), md.String())
}

func (e *Exporter) writeUser(user catalog.User) error {
	dir := filepath.Join(e.outputDir, "users")

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return fmt.Errorf("create users dir: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "id: %s\n", user.ID)
	fmt.Fprintf(&sb, "name: %s\n", user.Name)

	if user.AvatarURL != "" {
		fmt.Fprintf(&sb, "avatarUrl: %q\n", user.AvatarURL)
	}

	if user.Role != "" {
		fmt.Fprintf(&sb, "role: %q\n", user.Role)
	}

	if user.Email != "" {
		fmt.Fprintf(&sb, "email: %q\n", user.Email)
	}

	if user.SlackDirectMessageURL != "" {
		fmt.Fprintf(&sb, "slackDirectMessageUrl: %q\n", user.SlackDirectMessageURL)
	}

	fmt.Fprintf(&sb, "---\n\n# %s\n", user.Name)

	return e.writeMDXFile(filepath.Join(dir, user.ID+".mdx"), sb.String())
}
