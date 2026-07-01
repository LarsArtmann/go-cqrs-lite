package codec_test

import (
	"bytes"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
)

func BenchmarkJSONCodec_Encode(b *testing.B) {
	b.ReportAllocs()

	c := codec.JSONCodec{}
	payload := map[string]string{"name": "Alice", "email": "alice@example.com"}

	b.ResetTimer()

	for b.Loop() {
		_, err := c.Encode(payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONCodec_Decode(b *testing.B) {
	b.ReportAllocs()

	c := codec.JSONCodec{}
	data, _ := c.Encode(map[string]string{"name": "Alice", "email": "alice@example.com"})

	b.ResetTimer()

	for b.Loop() {
		var result map[string]string
		if err := c.Decode(data, &result); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCBORCodec_Encode(b *testing.B) {
	b.ReportAllocs()

	c := codec.CBORCodec{}
	payload := map[string]string{"name": "Alice", "email": "alice@example.com"}

	b.ResetTimer()

	for b.Loop() {
		_, err := c.Encode(payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCBORCodec_Decode(b *testing.B) {
	b.ReportAllocs()

	c := codec.CBORCodec{}
	data, _ := c.Encode(map[string]string{"name": "Alice", "email": "alice@example.com"})

	b.ResetTimer()

	for b.Loop() {
		var result map[string]string
		if err := c.Decode(data, &result); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodecComparison_Encode(b *testing.B) {
	b.ReportAllocs()

	jsonCodec := codec.JSONCodec{}
	cborCodec := codec.CBORCodec{}
	payload := map[string]string{"name": "Alice", "email": "alice@example.com"}

	b.Run("JSON", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := jsonCodec.Encode(payload)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CBOR", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := cborCodec.Encode(payload)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkCodecComparison_Decode(b *testing.B) {
	b.ReportAllocs()

	jsonCodec := codec.JSONCodec{}
	cborCodec := codec.CBORCodec{}
	payload := map[string]string{"name": "Alice", "email": "alice@example.com"}

	jsonData, _ := jsonCodec.Encode(payload)
	cborData, _ := cborCodec.Encode(payload)

	b.Run("JSON", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			var result map[string]string
			if err := jsonCodec.Decode(jsonData, &result); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CBOR", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			var result map[string]string
			if err := cborCodec.Decode(cborData, &result); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRawCodec_Encode(b *testing.B) {
	b.ReportAllocs()

	c := codec.RawCodec{}
	data := []byte("raw payload bytes")

	b.ResetTimer()

	for b.Loop() {
		_, err := c.Encode(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRawCodec_Decode(b *testing.B) {
	b.ReportAllocs()

	c := codec.RawCodec{}
	data := []byte("raw payload bytes")

	b.ResetTimer()

	for b.Loop() {
		var result []byte
		if err := c.Decode(data, &result); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCBORCompact_vs_Canon_Size(b *testing.B) {
	type eventPayload struct {
		Name    string
		Email   string
		Version int
		Active  bool
	}
	payload := eventPayload{Name: "Alice", Email: "alice@example.com", Version: 42, Active: true}

	canonical := codec.CBORCodec{}
	compact := codec.CBORCompactCodec{}

	canonicalData, _ := canonical.Encode(payload)
	compactData, _ := compact.Encode(payload)

	b.Logf(
		"CBOR (canonical): %d bytes, CBOR (compact): %d bytes, savings: %.1f%%",
		len(canonicalData), len(compactData),
		float64(len(canonicalData)-len(compactData))/float64(len(canonicalData))*100,
	)

	codecs := []struct {
		name string
		c    codec.Codec
	}{
		{"Canonical", canonical},
		{"Compact", compact},
	}

	for _, tc := range codecs {
		b.Run(tc.name+"/Encode", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err := tc.c.Encode(payload)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// realisticOrder simulates a real-world event payload with mixed field types.
type realisticOrder struct {
	OrderID    string
	CustomerID string
	Items      []orderItem
	TotalCents int64
	Currency   string
	Status     string
	CreatedAt  int64
}

type orderItem struct {
	SKU       string
	Quantity  int
	UnitPrice int64
}

// realisticOrderArray uses the toarray tag for positional CBOR encoding.
type realisticOrderArray struct {
	_          struct{} `cbor:",toarray"`
	OrderID    string
	CustomerID string
	Items      []orderItem
	TotalCents int64
	Currency   string
	Status     string
	CreatedAt  int64
}

func sampleOrder() realisticOrder {
	return realisticOrder{
		OrderID:    "01HQ3TS7HNW3K4PR9XJ8Z2V5MS",
		CustomerID: "01HQ3TR9JNW3K4PR9XJ8Z2V5NS",
		Items: []orderItem{
			{SKU: "WIDGET-001", Quantity: 2, UnitPrice: 1999},
			{SKU: "GADGET-042", Quantity: 1, UnitPrice: 4999},
		},
		TotalCents: 8997,
		Currency:   "USD",
		Status:     "pending",
		CreatedAt:  1700000000,
	}
}

func BenchmarkRealisticPayload_Encode(b *testing.B) {
	order := sampleOrder()
	orderArr := realisticOrderArray{
		OrderID:    order.OrderID,
		CustomerID: order.CustomerID,
		Items:      order.Items,
		TotalCents: order.TotalCents,
		Currency:   order.Currency,
		Status:     order.Status,
		CreatedAt:  order.CreatedAt,
	}

	jsonCodec := codec.JSONCodec{}
	cborCodec := codec.CBORCodec{}
	compactCodec := codec.CBORCompactCodec{}

	jsonData, _ := jsonCodec.Encode(order)
	cborData, _ := cborCodec.Encode(order)
	compactData, _ := compactCodec.Encode(orderArr)

	b.Logf("Realistic order payload sizes:")
	b.Logf("  JSON:              %d bytes", len(jsonData))
	b.Logf("  CBOR canonical:    %d bytes (%.1f%% of JSON)",
		len(cborData), float64(len(cborData))/float64(len(jsonData))*100)
	b.Logf("  CBOR compact+toarray: %d bytes (%.1f%% of JSON)",
		len(compactData), float64(len(compactData))/float64(len(jsonData))*100)

	codecs := []struct {
		name string
		c    codec.Codec
		v    any
	}{
		{"JSON", jsonCodec, order},
		{"CBOR", cborCodec, order},
		{"CBOR_compact_toarray", compactCodec, orderArr},
	}

	for _, tc := range codecs {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err := tc.c.Encode(tc.v)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkRealisticPayload_Decode(b *testing.B) {
	order := sampleOrder()
	orderArr := realisticOrderArray{
		OrderID:    order.OrderID,
		CustomerID: order.CustomerID,
		Items:      order.Items,
		TotalCents: order.TotalCents,
		Currency:   order.Currency,
		Status:     order.Status,
		CreatedAt:  order.CreatedAt,
	}

	jsonCodec := codec.JSONCodec{}
	cborCodec := codec.CBORCodec{}
	compactCodec := codec.CBORCompactCodec{}

	jsonData, _ := jsonCodec.Encode(order)
	cborData, _ := cborCodec.Encode(order)
	compactData, _ := compactCodec.Encode(orderArr)

	b.Run("JSON", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			var result realisticOrder
			if err := jsonCodec.Decode(jsonData, &result); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CBOR", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			var result realisticOrder
			if err := cborCodec.Decode(cborData, &result); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CBOR_compact_toarray", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			var result realisticOrderArray
			if err := compactCodec.Decode(compactData, &result); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkBufferEncoder(b *testing.B) {
	order := sampleOrder()

	codecs := []struct {
		name string
		c    codec.BufferEncoder
	}{
		{"JSON", codec.JSONCodec{}},
		{"CBOR", codec.CBORCodec{}},
		{"CBOR_compact", codec.CBORCompactCodec{}},
	}

	for _, tc := range codecs {
		b.Run(tc.name, func(b *testing.B) {
			buf := &bytes.Buffer{}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				buf.Reset()
				if err := tc.c.EncodeToBuffer(order, buf); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
