package metaengine_test

import (
	"encoding/json"
	"math"
	"math/big"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ─── DecodeFloat ───

func TestDecodeFloat_Nil(t *testing.T) {
	t.Parallel()

	got, err := metaengine.DecodeFloat(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != 0 {
		t.Errorf("DecodeFloat(nil) = %v, want 0", got)
	}
}

func TestDecodeFloat_Float64(t *testing.T) {
	t.Parallel()

	got, err := metaengine.DecodeFloat(float64(3.14))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != 3.14 {
		t.Errorf("DecodeFloat(float64) = %v, want 3.14", got)
	}
}

func TestDecodeFloat_Float32(t *testing.T) {
	t.Parallel()

	got, err := metaengine.DecodeFloat(float32(2.5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != 2.5 {
		t.Errorf("DecodeFloat(float32) = %v, want 2.5", got)
	}
}

func TestDecodeFloat_Int64(t *testing.T) {
	t.Parallel()

	got, err := metaengine.DecodeFloat(int64(42))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != 42 {
		t.Errorf("DecodeFloat(int64) = %v, want 42", got)
	}
}

func TestDecodeFloat_Int(t *testing.T) {
	t.Parallel()

	got, err := metaengine.DecodeFloat(int(99))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != 99 {
		t.Errorf("DecodeFloat(int) = %v, want 99", got)
	}
}

func TestDecodeFloat_BigInt(t *testing.T) {
	t.Parallel()

	bi := big.NewInt(1_000_000)
	got, err := metaengine.DecodeFloat(bi)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != 1_000_000 {
		t.Errorf("DecodeFloat(*big.Int) = %v, want 1000000", got)
	}
}

func TestDecodeFloat_BigInt_LargeValue(t *testing.T) {
	t.Parallel()

	// 2^200 is well beyond int64 range. Float64 can represent powers of two
	// exactly up to 2^1023, so this conversion is lossless.
	bi := new(big.Int).Lsh(big.NewInt(1), 200) // 2^200
	got, err := metaengine.DecodeFloat(bi)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := math.Ldexp(1.0, 200) // 2^200 as exact float64
	if got != expected {
		t.Errorf("DecodeFloat(2^200) = %v, want %v", got, expected)
	}
}

func TestDecodeFloat_ByteSlice(t *testing.T) {
	t.Parallel()

	raw, _ := json.Marshal(7.5)
	got, err := metaengine.DecodeFloat(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != 7.5 {
		t.Errorf("DecodeFloat([]byte) = %v, want 7.5", got)
	}
}

func TestDecodeFloat_ByteSlice_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := metaengine.DecodeFloat([]byte("not-json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON []byte, got nil")
	}

	if !strings.Contains(err.Error(), "DecodeFloat") {
		t.Errorf("error should mention DecodeFloat, got: %v", err)
	}
}

func TestDecodeFloat_UnknownType(t *testing.T) {
	t.Parallel()

	_, err := metaengine.DecodeFloat("a string")
	if err == nil {
		t.Fatal("expected error for string type, got nil")
	}

	if !strings.Contains(err.Error(), "unexpected type") {
		t.Errorf("error should mention unexpected type, got: %v", err)
	}

	if !strings.Contains(err.Error(), "string") {
		t.Errorf("error should mention the type name, got: %v", err)
	}
}

func TestDecodeFloat_TableDriven(t *testing.T) {
	t.Parallel()

	bigInt := big.NewInt(500)
	validBytes, _ := json.Marshal(12.5)

	tests := []struct {
		name    string
		input   any
		want    float64
		wantErr bool
	}{
		{"nil", nil, 0, false},
		{"float64", float64(1.1), 1.1, false},
		{"float64_zero", float64(0), 0, false},
		{"float64_negative", float64(-3.7), -3.7, false},
		{"float32", float32(9.5), 9.5, false},
		{"int64", int64(777), 777, false},
		{"int64_zero", int64(0), 0, false},
		{"int", int(33), 33, false},
		{"int_negative", int(-10), -10, false},
		{"big_int", bigInt, 500, false},
		{"big_int_zero", big.NewInt(0), 0, false},
		{"byte_slice_valid", validBytes, 12.5, false},
		{"byte_slice_zero", []byte("0"), 0, false},
		{"string_error", "hello", 0, true},
		{"bool_error", true, 0, true},
		{"uint_error", uint(5), 0, true},
		{"map_error", map[string]int{"x": 1}, 0, true},
		{"slice_error", []int{1, 2}, 0, true},
		{"chan_error", make(chan int), 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := metaengine.DecodeFloat(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("DecodeFloat(%v) expected error, got nil", tc.input)
				}

				return
			}

			if err != nil {
				t.Fatalf("DecodeFloat(%v) unexpected error: %v", tc.input, err)
			}

			if got != tc.want {
				t.Errorf("DecodeFloat(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ─── DecodeFloatResults ───

func TestDecodeFloatResults_EmptySpecs(t *testing.T) {
	t.Parallel()

	result, err := metaengine.DecodeFloatResults(nil, nil, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestDecodeFloatResults_NilRaws(t *testing.T) {
	t.Parallel()

	specs := []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "count"},
		{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
	}
	raws := []any{nil, nil}

	result, err := metaengine.DecodeFloatResults(raws, specs, "MultiAggregate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["count"] != 0 {
		t.Errorf("nil raw → expected 0, got %v", result["count"])
	}

	if result["total"] != 0 {
		t.Errorf("nil raw → expected 0, got %v", result["total"])
	}
}

func TestDecodeFloatResults_ExplicitAlias(t *testing.T) {
	t.Parallel()

	specs := []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "count"},
		{Fn: metaengine.AggregateSum, Column: "price", Alias: "total"},
		{Fn: metaengine.AggregateMin, Column: "price", Alias: "min_price"},
	}
	raws := []any{int64(5), float64(55.0), int64(-5)}

	result, err := metaengine.DecodeFloatResults(raws, specs, "MultiAggregate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["count"] != 5 {
		t.Errorf("count = %v, want 5", result["count"])
	}

	if result["total"] != 55 {
		t.Errorf("total = %v, want 55", result["total"])
	}

	if result["min_price"] != -5 {
		t.Errorf("min_price = %v, want -5", result["min_price"])
	}
}

func TestDecodeFloatResults_DefaultAlias(t *testing.T) {
	t.Parallel()

	// Specs without explicit Alias — AliasOr() generates default names.
	specs := []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount},                   // → "count"
		{Fn: metaengine.AggregateSum, Column: "price"},    // → "SUM(price)"
	}
	raws := []any{int64(3), float64(42.0)}

	result, err := metaengine.DecodeFloatResults(raws, specs, "MultiAggregate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["count"] != 3 {
		t.Errorf("default alias count = %v, want 3", result["count"])
	}

	if result["SUM(price)"] != 42 {
		t.Errorf("default alias SUM(price) = %v, want 42", result["SUM(price)"])
	}
}

func TestDecodeFloatResults_MixedTypes(t *testing.T) {
	t.Parallel()

	bigInt := big.NewInt(100)
	rawBytes, _ := json.Marshal(7.7)

	specs := []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "a"},    // int64 from COUNT
		{Fn: metaengine.AggregateSum, Column: "x", Alias: "b"}, // *big.Int from DuckDB
		{Fn: metaengine.AggregateAvg, Column: "x", Alias: "c"}, // []byte encoding
		{Fn: metaengine.AggregateMax, Column: "x", Alias: "d"}, // float64 from DOUBLE
	}
	raws := []any{int64(10), bigInt, rawBytes, float64(99.9)}

	result, err := metaengine.DecodeFloatResults(raws, specs, "MultiAggregate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["a"] != 10 {
		t.Errorf("a = %v, want 10", result["a"])
	}

	if result["b"] != 100 {
		t.Errorf("b = %v, want 100", result["b"])
	}

	if result["c"] != 7.7 {
		t.Errorf("c = %v, want 7.7", result["c"])
	}

	if result["d"] != 99.9 {
		t.Errorf("d = %v, want 99.9", result["d"])
	}
}

func TestDecodeFloatResults_ErrorPropagation(t *testing.T) {
	t.Parallel()

	specs := []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateCount, Alias: "ok"},
		{Fn: metaengine.AggregateSum, Column: "x", Alias: "bad"},
	}
	raws := []any{int64(5), "not-a-number"}

	_, err := metaengine.DecodeFloatResults(raws, specs, "MultiAggregate")
	if err == nil {
		t.Fatal("expected error for unknown type in raws")
	}

	// Error should contain the errPrefix and the alias.
	msg := err.Error()
	if !strings.Contains(msg, "MultiAggregate") {
		t.Errorf("error should contain errPrefix %q, got: %v", "MultiAggregate", msg)
	}

	if !strings.Contains(msg, "bad") {
		t.Errorf("error should contain alias %q, got: %v", "bad", msg)
	}

	if !strings.Contains(msg, "unexpected type") {
		t.Errorf("error should contain the DecodeFloat cause, got: %v", msg)
	}
}

func TestDecodeFloatResults_InvalidByteSlice(t *testing.T) {
	t.Parallel()

	specs := []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateAvg, Column: "x", Alias: "avg"},
	}
	raws := []any{[]byte("invalid")}

	_, err := metaengine.DecodeFloatResults(raws, specs, "MultiAggregate")
	if err == nil {
		t.Fatal("expected error for invalid JSON []byte")
	}

	if !strings.Contains(err.Error(), "avg") {
		t.Errorf("error should contain alias %q, got: %v", "avg", err)
	}
}

func TestDecodeFloatResults_DefaultAliasForNonCount(t *testing.T) {
	t.Parallel()

	specs := []metaengine.AggregateSpec{
		{Fn: metaengine.AggregateMin, Column: "price"},
	}
	raws := []any{float64(1.5)}

	result, err := metaengine.DecodeFloatResults(raws, specs, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Default alias for non-COUNT is "FN(Column)".
	val, ok := result["MIN(price)"]
	if !ok {
		t.Fatalf("expected key MIN(price), got keys: %v", result)
	}

	if val != 1.5 {
		t.Errorf("MIN(price) = %v, want 1.5", val)
	}
}
