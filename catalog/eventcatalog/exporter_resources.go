package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func (e *Exporter) writeChannel(ch catalog.Channel) error {
	dir := filepath.Join(e.outputDir, "channels", string(ch.ID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_resources.1",
			"create channel dir: %v",
			err,
		)
	}

	md := newFrontmatterWriter()
	md.addField("id", string(ch.ID))
	md.addField("name", string(ch.Name))
	md.addField("version", string(ch.Version))

	if ch.Summary != "" {
		md.addQuotedField("summary", string(ch.Summary))
	}

	if ch.Address != "" {
		md.addQuotedField("address", string(ch.Address))
	}

	protocols := make([]string, len(ch.Protocols))
	for i, p := range ch.Protocols {
		protocols[i] = string(p)
	}
	md.addListField("protocols", protocols)

	if ch.DeliveryGuarantee != "" {
		md.addField("deliveryGuarantee", string(ch.DeliveryGuarantee))
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
	md.finishWithGraph(string(ch.Name), string(ch.Summary))

	return e.writeMDXFile(filepath.Join(dir, indexFile), md.String())
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
	dir := filepath.Join(e.outputDir, "data", string(ds.ID))

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.exporter_resources.2",
			"create data store dir: %v",
			err,
		)
	}

	md := newFrontmatterWriter()
	md.addField("id", string(ds.ID))
	md.addField("name", string(ds.Name))
	md.addField("version", string(ds.Version))
	md.addField("container_type", ds.ContainerType)

	if ds.Summary != "" {
		md.addQuotedField("summary", string(ds.Summary))
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
	md.finishWithGraph(string(ds.Name), string(ds.Summary))

	return e.writeMDXFile(filepath.Join(dir, indexFile), md.String())
}
