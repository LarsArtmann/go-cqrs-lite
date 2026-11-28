package search_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/search/v4"
)

var _ = Describe("Search", func() {
	var (
		index  *search.Index[Note]
		engine = search.NoEngine()
		spec   search.Spec[Note]
		result search.Result[Note]
		err    error
	)
	searchFor := func(query string) search.Result[Note] {
		result, err = index.Search(context.Background(), search.Request{Query: query})

		return result
	}
	idsOf := func() []string { return ids(result.Data) }

	BeforeEach(func() {
		engine = search.NoEngine()
		spec = testSpec()
	})

	JustBeforeEach(func() {
		var buildErr error
		index, buildErr = search.New(spec, engine, notes())
		Expect(buildErr).NotTo(HaveOccurred())
	})

	Describe("ranking", func() {
		It("ranks a title match above a body match", func() {
			searchFor("invoice")
			Expect(idsOf()).To(Equal([]string{"n1", "n4", "n3"}))
			Expect(result.Data[0].Snippet).To(Equal([]search.Segment{
				{Text: "Big ", Marked: false}, {Text: "invoice", Marked: true},
			}))
		})

		It("requires every term", func() {
			searchFor("big invoice")
			Expect(idsOf()).To(Equal([]string{"n1", "n3"}))
			searchFor("big holiday")
			Expect(idsOf()).To(BeEmpty())
		})

		It("keeps an unknown key:value token as text", func() {
			spec.Keys = nil
			searchFor("amount:10")
			Expect(result.TotalCount).To(BeZero())
			searchFor("title:x")
			Expect(idsOf()).To(BeEmpty())
		})

		It("keeps a declared operator token out of the text search", func() {
			operator, buildErr := search.New(
				search.Spec[Note]{Fields: testSpec().Fields, Keys: []string{"status"}},
				search.NoEngine(),
				notes(),
			)
			Expect(buildErr).NotTo(HaveOccurred())
			byOperator, searchErr := operator.Search(context.Background(), search.Request{Query: "status:done invoice"})
			Expect(searchErr).NotTo(HaveOccurred())
			Expect(byOperator.Via).To(Equal(search.ViaScan))
			Expect(ids(byOperator.Data)).To(Equal([]string{"n1", "n4", "n3"}), "only the free term is searched")
		})
	})

	Describe("paging", func() {
		It("pages the ranked hits", func() {
			result, err = index.Search(context.Background(), search.Request{
				Query: "invoice",
				Page:  query.Pagination{Page: 2, PageSize: 1},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.TotalCount).To(Equal(uint(3)))
			Expect(idsOf()).To(HaveLen(1))
			Expect(result.HasPrev()).To(BeTrue())
		})

		It("reports Truncated when the source caps its scan", func() {
			capped, err := search.New(testSpec(), search.NoEngine(), &cappedSource{Slice: notes(), cap: 1})
			Expect(err).NotTo(HaveOccurred())
			cappedResult, err := capped.Search(context.Background(), search.Request{})
			Expect(err).NotTo(HaveOccurred())
			Expect(cappedResult.Truncated).To(BeTrue())
			Expect(cappedResult.TotalCount).To(Equal(uint(1)))
		})
	})

	Describe("the ladder", func() {
		It("rejects an explicit mode the engine lacks", func() {
			strict, buildErr := search.New(
				search.Spec[Note]{Fields: spec.Fields, Mode: search.ModeVector},
				search.NoEngine(),
				notes(),
			)
			Expect(buildErr).NotTo(HaveOccurred())
			_, err = strict.Search(context.Background(), proseQuery)
			Expect(err).To(MatchError(search.ErrModeUnavailable))
		})

		It("serves prose on the scan when the deployment has no embedder", func() {
			searchFor("unpaid invoices")
			Expect(result.Via).To(Equal(search.ViaScan))
			Expect(result.Degraded).To(BeFalse())
		})

		It("steps down and says so when the engine advertises a tier without an executor", func() {
			strict, buildErr := search.New(
				search.Spec[Note]{Fields: spec.Fields, Mode: search.ModeFullText},
				fakeEngine{caps: fullText},
				notes(),
			)
			Expect(buildErr).NotTo(HaveOccurred())
			_, err = strict.Search(context.Background(), proseQuery)
			Expect(err).To(MatchError(search.ErrModeUnavailable))

			degraded, searchErr := autoIndex(fakeEngine{caps: fullText}).Search(context.Background(), proseQuery)
			Expect(searchErr).NotTo(HaveOccurred())
			Expect(degraded.Via).To(Equal(search.ViaScan))
			Expect(degraded.Degraded).To(BeTrue())
		})

		It("keeps the order stable across runs", func() {
			first := searchFor("invoice")
			for range 20 {
				Expect(searchFor("invoice").Data).To(Equal(first.Data))
			}
		})
	})
})

// autoIndex builds an Auto index over the notes with the given engine.
func autoIndex(eng search.Engine) *search.Index[Note] {
	index, err := search.New(testSpec(), eng, notes())
	Expect(err).NotTo(HaveOccurred())

	return index
}

// proseQuery is a query Auto would send to the highest tier the deployment
// offers.
var proseQuery = search.Request{Query: "unpaid invoices"}
