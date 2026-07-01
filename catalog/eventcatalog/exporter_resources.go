package eventcatalog

import (
	"os"
	"path/filepath"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func (e *Exporter) writeChannel(ch catalog.Channel) error {
	dir := filepath.Join(e.outputDir, "channels", string(ch.ID))

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_resources.1",
			"create channel dir: %v", err)
	}

	protocols := make([]string, len(ch.Protocols))
	for i, p := range ch.Protocols {
		protocols[i] = string(p)
	}

	routes := make([]channelRouteFM, len(ch.Routes))
	for i, r := range ch.Routes {
		routes[i] = channelRouteFM{ID: string(r.ID)}
	}

	fm := channelFM{
		ID:                string(ch.ID),
		Name:              string(ch.Name),
		Version:           string(ch.Version),
		Summary:           string(ch.Summary),
		Address:           string(ch.Address),
		Protocols:         protocols,
		Messages:          toPointers(ch.Messages),
		DeliveryGuarantee: string(ch.DeliveryGuarantee),
		Parameters:        toChannelParams(ch.Parameters),
		Routes:            routes,
		Owners:            ch.Owners,
		Badges:            toBadges(ch.Badges),
	}

	content, err := renderMDX(fm, string(ch.Name), string(ch.Summary), true)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_resources.1b",
			"render channel %s: %v", ch.ID, err)
	}

	return e.writeMDXFile(filepath.Join(dir, indexFile), content)
}

func (e *Exporter) writeDataStore(ds catalog.DataStore) error {
	dir := filepath.Join(e.outputDir, "data", string(ds.ID))

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_resources.2",
			"create data store dir: %v", err)
	}

	fm := dataStoreFM{
		ID:             string(ds.ID),
		Name:           string(ds.Name),
		Version:        string(ds.Version),
		ContainerType:  ds.ContainerType,
		Summary:        string(ds.Summary),
		Technology:     ds.Technology,
		Classification: ds.Classification,
		Retention:      ds.Retention,
		Residency:      ds.Residency,
		Authoritative:  ds.Authoritative,
		AccessMode:     ds.AccessMode,
		Owners:         ds.Owners,
		Badges:         toBadges(ds.Badges),
	}

	content, err := renderMDX(fm, string(ds.Name), string(ds.Summary), true)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_resources.2b",
			"render data store %s: %v", ds.ID, err)
	}

	return e.writeMDXFile(filepath.Join(dir, indexFile), content)
}
