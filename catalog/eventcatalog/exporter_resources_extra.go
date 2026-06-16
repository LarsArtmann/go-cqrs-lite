package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
)

func (e *Exporter) writeFlow(f catalog.Flow) error {
	dir := filepath.Join(e.outputDir, "flows", string(f.ID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_resources_extra.1",
			"create flow dir: %v",
			err,
		)
	}

	md := newFrontmatterWriter()
	md.addField("id", string(f.ID))
	md.addField("name", string(f.Name))
	md.addField("version", string(f.Version))

	if f.Summary != "" {
		md.addQuotedField("summary", string(f.Summary))
	}

	writeBadges(md, f.Badges)
	writeFlowSteps(md, f.Steps)
	md.finishWithGraph(string(f.Name), string(f.Summary))

	return e.writeMDXFile(filepath.Join(dir, indexFile), md.String())
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
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_resources_extra.2",
			"create teams dir: %v",
			err,
		)
	}

	md := newFrontmatterWriter()
	md.addField("id", string(team.ID))
	md.addField("name", string(team.Name))

	if team.Summary != "" {
		md.addQuotedField("summary", string(team.Summary))
	}

	md.addListField("members", team.Members)

	if team.Email != "" {
		md.addQuotedField("email", string(team.Email))
	}

	if team.SlackDirectMessageURL != "" {
		md.addQuotedField("slackDirectMessageUrl", string(team.SlackDirectMessageURL))
	}

	md.finish(string(team.Name), "")

	return e.writeMDXFile(filepath.Join(dir, string(team.ID)+".mdx"), md.String())
}

func (e *Exporter) writeUser(user catalog.User) error {
	dir := filepath.Join(e.outputDir, "users")

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_resources_extra.3",
			"create users dir: %v",
			err,
		)
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

	return e.writeMDXFile(filepath.Join(dir, string(user.ID)+".mdx"), sb.String())
}
