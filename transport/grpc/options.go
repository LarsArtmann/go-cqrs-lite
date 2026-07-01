package grpc

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v3"
)

// Option configures a gRPC server or client. It is the shared functional-option
// type for RegisterCommandService, RegisterQueryService, NewCommandClient, and
// NewQueryClient.
type Option func(*config)

type config struct {
	codec codec.Codec
}

func defaultConfig() *config {
	return &config{codec: codec.JSONCodec{}}
}

func (c *config) apply(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
}

// WithCodec sets the codec used to encode/decode payloads on the gRPC wire.
// By default, JSON is used for backwards compatibility. Pass codec.CBORCodec{}
// for smaller payloads and faster decode:
//
//	cqrsgrpc.RegisterQueryService(srv, disp, cqrsgrpc.WithCodec(codec.CBORCodec{}))
//	client := cqrsgrpc.NewQueryClient(conn, cqrsgrpc.WithCodec(codec.CBORCodec{}))
//
// Both server and client MUST use the same codec — there is no format
// negotiation. If the codec doesn't match, unmarshal will fail.
func WithCodec(c codec.Codec) Option {
	return func(cfg *config) {
		if c != nil {
			cfg.codec = c
		}
	}
}

// configForServer builds a config from variadic options (server-side).
func configForServer(opts []Option) *config {
	cfg := defaultConfig()
	cfg.apply(opts...)

	return cfg
}
