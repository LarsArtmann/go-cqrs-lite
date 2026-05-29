package query_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/query"
)

var _ = Describe("Pagination", func() {
	Describe("As a developer building paginated queries", func() {
		Context("when I create pagination with valid values", func() {
			It(
				"should use the exact values I provide, so my API consumers get predictable paging",
				func() {
					p := query.NewPagination(2, 25)
					Expect(p.Page).To(Equal(uint(2)))
					Expect(p.PageSize).To(Equal(uint(25)))
					Expect(p.Offset()).To(Equal(25))
				},
			)
		})

		Context("when I create pagination with zero page", func() {
			It(
				"should default to page 1 so my frontend doesn't break on missing query params",
				func() {
					p := query.NewPagination(0, 10)
					Expect(p.Page).To(Equal(uint(1)))
				},
			)
		})

		DescribeTable(
			"NewPagination clamps invalid page size values",
			func(pageSize, expected uint, description string) {
				p := query.NewPagination(1, pageSize)
				Expect(p.PageSize).To(Equal(expected), description)
			},
			Entry(
				"zero page size defaults to 20 to prevent unbounded result sets",
				uint(0),
				uint(20),
				"zero page size should default to 20",
			),
			Entry(
				"page size exceeding max clamps to 100 to protect the database",
				uint(500),
				uint(100),
				"exceeding max should clamp to 100",
			),
		)
	})
})

var _ = Describe("PaginatedResult", func() {
	Describe("As a developer returning paged data to my frontend", func() {
		DescribeTable(
			"NewPaginatedResult computes pages correctly",
			func(page, pageSize, total, wantPages uint, wantNext, wantPrev bool) {
				p := query.NewPagination(page, pageSize)
				result := query.NewPaginatedResult(
					[]string{"a"}, total, p,
				)
				Expect(result.TotalPages).To(Equal(wantPages))
				Expect(result.HasNext()).To(Equal(wantNext))
				Expect(result.HasPrev()).To(Equal(wantPrev))
			},
			Entry("page 1 of 5", uint(1), uint(10), uint(50), uint(5), true, false),
			Entry("last page (5 of 5)", uint(5), uint(10), uint(50), uint(5), false, true),
			Entry("page 1 of 1", uint(1), uint(10), uint(2), uint(1), false, false),
			Entry("zero total items", uint(1), uint(10), uint(0), uint(0), false, false),
			Entry("uneven division", uint(1), uint(10), uint(23), uint(3), true, false),
		)
	})
})
