package system

import (
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
)

// newProjectionAdapter builds the adapter that feeds journal events into the
// metaengine projection Store. Decoder priority (documented on the
// DomainConfig fields): TypeDecoder > EventDecoder > the decoder auto-derived
// from Evolutions > ProjectionDecoder > generic JSON maps.
func newProjectionAdapter(
	sys *System,
	domain DomainConfig,
	autoEventDecoder eventDecoderFn,
) *projectionadapter.Adapter {
	switch {
	case domain.ProjectionTypeDecoder != nil:
		return projectionadapter.NewWithDecoder(
			"projections", sys.projStore, domain.ProjectionTypeDecoder,
		)
	case domain.ProjectionEventDecoder != nil:
		return projectionadapter.New("projections", sys.projStore, nil,
			projectionadapter.WithEventDecoder(domain.ProjectionEventDecoder),
		)
	case autoEventDecoder != nil:
		return projectionadapter.New("projections", sys.projStore, nil,
			projectionadapter.WithEventDecoder(
				projectionadapter.EventDecoder(autoEventDecoder),
			),
		)
	default:
		var decoder projectionadapter.PayloadDecoder

		if domain.ProjectionDecoder != nil {
			decoder = projectionadapter.PayloadDecoder(domain.ProjectionDecoder)
		}

		return projectionadapter.New("projections", sys.projStore, decoder)
	}
}
