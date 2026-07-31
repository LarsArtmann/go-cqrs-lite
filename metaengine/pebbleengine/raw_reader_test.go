package pebbleengine_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// raw_reader_test.go validates that the Pebble engine implements the
// RawValueReader and RawScanReader interfaces and that the raw bytes returned
// match the JSON representation of the stored values.

func TestPebbleGetRawValue(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	rvr := eng.(metaengine.RawValueReader)

	g.Expect(mb.MapSet(ctx, "users", "u1", map[string]any{"name": "Alice", "age": float64(30)})).
		To(Succeed())

	raw, found, err := rvr.GetRawValue(ctx, "users", "u1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())

	var decoded map[string]any
	g.Expect(json.Unmarshal(raw, &decoded)).To(Succeed())
	g.Expect(decoded["name"]).To(Equal("Alice"))
	g.Expect(decoded["age"]).To(Equal(float64(30)))
}

func TestPebbleGetRawValueNotFound(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	rvr := eng.(metaengine.RawValueReader)

	raw, found, err := rvr.GetRawValue(context.Background(), "users", "missing")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
	g.Expect(raw).To(BeNil())
}

func TestPebbleScanRawValuesNoFilter(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	rsr := eng.(metaengine.RawScanReader)

	for i := range 5 {
		g.Expect(mb.MapSet(ctx, "items", i, map[string]any{"id": i, "status": "open"})).
			To(Succeed())
	}

	raw, err := rsr.ScanRawValues(ctx, "items", nil, nil, nil, 0)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(raw).To(HaveLen(5))

	for _, b := range raw {
		var v map[string]any
		g.Expect(json.Unmarshal(b, &v)).To(Succeed())
		g.Expect(v["status"]).To(Equal("open"))
	}
}

func TestPebbleScanRawValuesWithFilter(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	rsr := eng.(metaengine.RawScanReader)

	items := []struct {
		key    string
		status string
	}{
		{"s1", "open"},
		{"s2", "done"},
		{"s3", "open"},
		{"s4", "done"},
		{"s5", "open"},
	}
	for _, item := range items {
		g.Expect(mb.MapSet(ctx, "tasks", item.key,
			map[string]any{"status": item.status, "key": item.key})).
			To(Succeed())
	}

	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "open"},
	}
	raw, err := rsr.ScanRawValues(ctx, "tasks", filters, nil, nil, 0)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(raw).To(HaveLen(3))

	for _, b := range raw {
		var v map[string]any
		g.Expect(json.Unmarshal(b, &v)).To(Succeed())
		g.Expect(v["status"]).To(Equal("open"))
	}
}

func TestPebbleScanRawValuesWithSort(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	rsr := eng.(metaengine.RawScanReader)

	items := []struct {
		key      string
		priority float64
	}{
		{"s1", 3},
		{"s2", 1},
		{"s3", 2},
	}
	for _, item := range items {
		g.Expect(mb.MapSet(ctx, "sorted", item.key,
			map[string]any{"priority": item.priority, "key": item.key})).
			To(Succeed())
	}

	sortSpec := &metaengine.SortSpec{Column: "priority", Desc: false}
	raw, err := rsr.ScanRawValues(ctx, "sorted", nil, sortSpec, nil, 0)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(raw).To(HaveLen(3))

	priorities := make([]float64, 0, 3)
	for _, b := range raw {
		var v map[string]any
		g.Expect(json.Unmarshal(b, &v)).To(Succeed())
		p, ok := v["priority"].(float64)
		g.Expect(ok).To(BeTrue())
		priorities = append(priorities, p)
	}

	g.Expect(priorities).To(Equal([]float64{1, 2, 3}))
}

func TestPebbleScanRawValuesWithSortDesc(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	rsr := eng.(metaengine.RawScanReader)

	items := []struct {
		key      string
		priority float64
	}{
		{"s1", 3},
		{"s2", 1},
		{"s3", 2},
	}
	for _, item := range items {
		g.Expect(mb.MapSet(ctx, "sorted", item.key,
			map[string]any{"priority": item.priority, "key": item.key})).
			To(Succeed())
	}

	sortSpec := &metaengine.SortSpec{Column: "priority", Desc: true}
	raw, err := rsr.ScanRawValues(ctx, "sorted", nil, sortSpec, nil, 0)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(raw).To(HaveLen(3))

	priorities := make([]float64, 0, 3)
	for _, b := range raw {
		var v map[string]any
		g.Expect(json.Unmarshal(b, &v)).To(Succeed())
		p, ok := v["priority"].(float64)
		g.Expect(ok).To(BeTrue())
		priorities = append(priorities, p)
	}

	g.Expect(priorities).To(Equal([]float64{3, 2, 1}))
}

func TestPebbleScanRawValuesWithCursor(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	rsr := eng.(metaengine.RawScanReader)

	for i := range 10 {
		g.Expect(mb.MapSet(ctx, "paged", i, map[string]any{"id": float64(i)})).
			To(Succeed())
	}

	sortSpec := &metaengine.SortSpec{Column: "id", Desc: false}

	// First page: first 3 items.
	raw, err := rsr.ScanRawValues(ctx, "paged", nil, sortSpec, nil, 3)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(raw).To(HaveLen(4)) // 3+1 overflow

	var lastID float64
	for _, b := range raw[:3] {
		var v map[string]any
		g.Expect(json.Unmarshal(b, &v)).To(Succeed())
		lastID = v["id"].(float64)
	}

	g.Expect(lastID).To(Equal(float64(2)))

	// Second page: cursor = lastID (2).
	raw2, err := rsr.ScanRawValues(ctx, "paged", nil, sortSpec, lastID, 3)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(raw2).To(HaveLen(4)) // items 3,4,5,6 (3+1 overflow)

	var firstIDPage2 float64
	var v map[string]any

	g.Expect(json.Unmarshal(raw2[0], &v)).To(Succeed())
	firstIDPage2 = v["id"].(float64)
	g.Expect(firstIDPage2).To(Equal(float64(3)))
}

func TestPebbleScanRawValuesWithFilterAndSort(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	rsr := eng.(metaengine.RawScanReader)

	items := []struct {
		key      string
		status   string
		priority float64
	}{
		{"s1", "open", 3},
		{"s2", "done", 5},
		{"s3", "open", 1},
		{"s4", "open", 2},
		{"s5", "done", 4},
	}
	for _, item := range items {
		g.Expect(mb.MapSet(ctx, "combined", item.key,
			map[string]any{"status": item.status, "priority": item.priority, "key": item.key})).
			To(Succeed())
	}

	filters := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterEq, Value: "open"},
	}
	sortSpec := &metaengine.SortSpec{Column: "priority", Desc: false}
	raw, err := rsr.ScanRawValues(ctx, "combined", filters, sortSpec, nil, 0)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(raw).To(HaveLen(3))

	priorities := make([]float64, 0, 3)
	for _, b := range raw {
		var v map[string]any
		g.Expect(json.Unmarshal(b, &v)).To(Succeed())
		g.Expect(v["status"]).To(Equal("open"))
		p, ok := v["priority"].(float64)
		g.Expect(ok).To(BeTrue())
		priorities = append(priorities, p)
	}

	g.Expect(priorities).To(Equal([]float64{1, 2, 3}))
}

func TestPebbleRawValueReaderInterface(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	_, ok := eng.(metaengine.RawValueReader)
	g.Expect(ok).To(BeTrue(), "pebble engine must implement RawValueReader")
}

func TestPebbleRawScanReaderInterface(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	_, ok := eng.(metaengine.RawScanReader)
	g.Expect(ok).To(BeTrue(), "pebble engine must implement RawScanReader")
}

// TestPebbleRawValueBytesAreCopies verifies that the bytes returned by
// GetRawValue are independent of Pebble's internal buffers (safe to retain
// after the closer is closed).
func TestPebbleRawValueBytesAreCopies(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	rvr := eng.(metaengine.RawValueReader)

	g.Expect(mb.MapSet(ctx, "copy", "k1", map[string]any{"v": "hello"})).
		To(Succeed())

	raw, found, err := rvr.GetRawValue(ctx, "copy", "k1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())

	// Overwrite the key — the previously returned raw bytes must be unchanged.
	g.Expect(mb.MapSet(ctx, "copy", "k1", map[string]any{"v": "world"})).
		To(Succeed())

	var decoded map[string]any
	g.Expect(json.Unmarshal(raw, &decoded)).To(Succeed())
	g.Expect(decoded["v"]).To(Equal("hello"))
}

// TestPebbleScanRawValuesLimitZero verifies that limit=0 returns all values.
func TestPebbleScanRawValuesLimitZero(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	rsr := eng.(metaengine.RawScanReader)

	for i := range 7 {
		g.Expect(mb.MapSet(ctx, "nolimit", i, map[string]any{"id": float64(i)})).
			To(Succeed())
	}

	raw, err := rsr.ScanRawValues(ctx, "nolimit", nil, nil, nil, 0)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(raw).To(HaveLen(7))
}
