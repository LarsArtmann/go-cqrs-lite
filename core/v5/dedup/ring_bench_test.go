package dedup

import (
	"strconv"
	"testing"
)

// art-dupl:accept v5 copy-forward twin of dedup/v4 (ADR-0152 dual-support); removed with v4 in T26
func BenchmarkRing_Add(b *testing.B) {
	r := NewRing(1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Add(strconv.Itoa(i))
	}
}

func BenchmarkRing_AddEvict(b *testing.B) {
	r := NewRing(1024)
	for i := 0; i < 1024; i++ {
		r.Add(strconv.Itoa(i))
	}

	b.ResetTimer()
	for i := 1024; i < b.N+1024; i++ {
		r.Add(strconv.Itoa(i))
	}
}

func BenchmarkRing_Has(b *testing.B) {
	r := NewRing(1024)
	for i := 0; i < 1024; i++ {
		r.Add(strconv.Itoa(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Has(strconv.Itoa(i % 1024))
	}
}

func BenchmarkRing_HasMiss(b *testing.B) {
	r := NewRing(1024)
	for i := 0; i < 1024; i++ {
		r.Add(strconv.Itoa(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Has(strconv.Itoa(i + 100000))
	}
}
