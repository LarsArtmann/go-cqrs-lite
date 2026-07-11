package eventcatalog

import (
	"os"
	"path/filepath"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func (e *Exporter) writeFlow(f catalog.Flow) error {
	dir := filepath.Join(e.outputDir, "flows", string(f.ID))

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_resources_extra.1",
			"create flow dir: %v", err)
	}

	fm := flowFM{
		ID:      string(f.ID),
		Name:    string(f.Name),
		Version: string(f.Version),
		Summary: string(f.Summary),
		Badges:  toBadges(f.Badges),
		Steps:   toFlowSteps(f.Steps),
	}

	content, err := renderMDX(fm, string(f.Name), string(f.Summary), true)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_resources_extra.1b",
			"render flow %s: %v", f.ID, err)
	}

	return e.writeMDXFile(filepath.Join(dir, indexFile), content)
}

func (e *Exporter) writeTeam(team catalog.Team) error {
	dir := filepath.Join(e.outputDir, "teams")

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_resources_extra.2",
			"create teams dir: %v", err)
	}

	fm := teamFM{
		ID:                    string(team.ID),
		Name:                  string(team.Name),
		Summary:               string(team.Summary),
		Members:               team.Members,
		Email:                 string(team.Email),
		AvatarURL:             string(team.AvatarURL),
		Role:                  string(team.Role),
		SlackDirectMessageURL: string(team.SlackDirectMessageURL),
		Hidden:                team.Hidden,
		ReadOnly:              team.ReadOnly,
		Source:                toSource(team.Source),
	}

	content, err := renderMDX(fm, string(team.Name), "", false)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_resources_extra.2b",
			"render team %s: %v", team.ID, err)
	}

	return e.writeMDXFile(filepath.Join(dir, string(team.ID)+".mdx"), content)
}

func (e *Exporter) writeUser(user catalog.User) error {
	dir := filepath.Join(e.outputDir, "users")

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_resources_extra.3",
			"create users dir: %v", err)
	}

	fm := userFM{
		ID:                    string(user.ID),
		Name:                  string(user.Name),
		AvatarURL:             string(user.AvatarURL),
		Role:                  string(user.Role),
		Email:                 string(user.Email),
		SlackDirectMessageURL: string(user.SlackDirectMessageURL),
		Hidden:                user.Hidden,
		ReadOnly:              user.ReadOnly,
		Source:                toSource(user.Source),
	}

	content, err := renderMDX(fm, string(user.Name), "", false)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_resources_extra.3b",
			"render user %s: %v", user.ID, err)
	}

	return e.writeMDXFile(filepath.Join(dir, string(user.ID)+".mdx"), content)
}
