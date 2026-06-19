//go:build !js

package otel

import (
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// NewCQRSViews returns SDK metric views optimized for CQRS operations.
// The views configure histogram boundaries covering the typical CQRS
// operation latency range (0.05ms to 10s) and are applied to all
// instruments matching the "cqrs." prefix.
//
// This function requires the OTel SDK and is excluded from WASM builds
// (GOOS=js) because the SDK's resource auto-detection depends on os/user.
//
//	provider, _ := sdkmetric.NewMeterProvider(
//	    sdkmetric.WithReader(reader),
//	    sdkmetric.WithView(cqrsotel.NewCQRSViews()...),
//	)
func NewCQRSViews() []sdkmetric.View {
	return []sdkmetric.View{
		sdkmetric.NewView(
			sdkmetric.Instrument{ //nolint:exhaustruct // only Name is a filter criteria
				Name: "cqrs.*",
			},
			sdkmetric.Stream{ //nolint:exhaustruct // only Aggregation is configured
				Aggregation: sdkmetric.AggregationExplicitBucketHistogram{ //nolint:exhaustruct // NoMinMax defaults to false
					Boundaries: CQRSHistogramBoundaries,
				},
			},
		),
	}
}
