package metaengine_test

import (
	"context"
	"time"

	"github.com/larsartmann/go-codec"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// The encoded entry points accept payload bytes whose codec is either sniffed
// from the bytes or declared on the Record. JSON is the historical default;
// CBOR is the default for event.New, so accepting it is a correctness
// requirement, not a nicety.
var _ = Describe("ApplyEncoded payload codecs", func() {
	newStore := func() *metaengine.Store {
		store, err := metaengine.Plan(
			[]metaengine.Engine{metaengine.NewMemoryEngine()},
			findTaskQuery(),
		)
		Expect(err).NotTo(HaveOccurred())

		return store
	}

	cborTask := func(id TaskID, title string) []byte {
		payload, err := codec.CBORCodec{}.Encode(TaskCreated{
			ID:       id,
			Title:    title,
			Assignee: "carol",
			Status:   "open",
			Priority: 1,
			At:       time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		})
		Expect(err).NotTo(HaveOccurred())

		return payload
	}

	It("applies a CBOR payload with a sniffed encoding", func() {
		store := newStore()
		DeferCleanup(store.Close)

		Expect(store.ApplyEncoded(
			context.Background(), "TaskCreated", cborTask("t-cbor", "From CBOR"),
		)).To(Succeed())

		result, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
			context.Background(), store, FindTask{ID: "t-cbor"},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Title).To(Equal("From CBOR"))
	})

	It("applies a CBOR payload stamped on the Record", func() {
		store := newStore()
		DeferCleanup(store.Close)

		rec := record.Record{
			Type:     "TaskCreated",
			Encoding: record.EncodingCBOR,
			Payload:  cborTask("t-cbor-rec", "From CBOR record"),
			StreamID: record.NewStreamRef("Task", "t-cbor-rec"),
			Version:  1,
		}

		Expect(store.ApplyEncodedRecord(context.Background(), rec, rec.Payload)).To(Succeed())

		result, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
			context.Background(), store, FindTask{ID: "t-cbor-rec"},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Title).To(Equal("From CBOR record"))
	})

	It("still applies a JSON payload whose Encoding stamp is absent", func() {
		store := newStore()
		DeferCleanup(store.Close)

		payload := []byte(
			`{"ID":"t-json","Title":"From JSON","Assignee":"bob","Status":"open","Priority":2,"At":"2026-01-01T00:00:00Z"}`,
		)

		Expect(store.ApplyEncoded(context.Background(), "TaskCreated", payload)).To(Succeed())

		result, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
			context.Background(), store, FindTask{ID: "t-json"},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Title).To(Equal("From JSON"))
	})

	It("rejects a payload that is neither JSON nor CBOR with a decode error", func() {
		store := newStore()
		DeferCleanup(store.Close)

		err := store.ApplyEncoded(
			context.Background(), "TaskCreated", []byte{0xde, 0xad, 0xbe, 0xef},
		)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("TaskCreated"))
	})
})
