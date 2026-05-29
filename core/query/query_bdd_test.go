package query_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/core/query"
)

var _ = Describe("Pagination", func() {
	Describe("As a developer building paginated queries", func() {
		Context("when I create pagination with valid values", func() {
			It("should use the exact values I provide", func() {
				p := query.NewPagination(2, 25)
				Expect(p.Page).To(Equal(uint(2)))
				Expect(p.PageSize).To(Equal(uint(25)))
				Expect(p.Offset()).To(Equal(25))
			})
		})

		Context("when I create pagination with zero page", func() {
			It("should default to page 1", func() {
				p := query.NewPagination(0, 10)
				Expect(p.Page).To(Equal(uint(1)))
			})
		})

		Context("when I create pagination with zero page size", func() {
			It("should default to 20", func() {
				p := query.NewPagination(1, 0)
				Expect(p.PageSize).To(Equal(uint(20)))
			})
		})

		Context("when I create pagination with page size exceeding max", func() {
			It("should clamp to 100", func() {
				p := query.NewPagination(1, 500)
				Expect(p.PageSize).To(Equal(uint(100)))
			})
		})
	})
})

var _ = Describe("PaginatedResult", func() {
	Describe("As a developer returning paged data", func() {
		Context("when I create a result with 50 total items and page size 10", func() {
			It("should compute 5 total pages with correct navigation flags", func() {
				p := query.NewPagination(1, 10)
				result := query.NewPaginatedResult(
					[]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
					50, p,
				)
				Expect(result.TotalPages).To(Equal(uint(5)))
				Expect(result.HasNext()).To(BeTrue())
				Expect(result.HasPrev()).To(BeFalse())
			})
		})

		Context("when I am on the last page", func() {
			It("should report HasNext as false and HasPrev as true", func() {
				p := query.NewPagination(5, 10)
				result := query.NewPaginatedResult(
					[]string{"a"}, 50, p,
				)
				Expect(result.HasNext()).To(BeFalse())
				Expect(result.HasPrev()).To(BeTrue())
			})
		})

		Context("when I am on page 1 of 1", func() {
			It("should report HasNext and HasPrev as false", func() {
				p := query.NewPagination(1, 10)
				result := query.NewPaginatedResult(
					[]string{"a", "b"}, 2, p,
				)
				Expect(result.TotalPages).To(Equal(uint(1)))
				Expect(result.HasNext()).To(BeFalse())
				Expect(result.HasPrev()).To(BeFalse())
			})
		})

		Context("when I have zero total items", func() {
			It("should report zero total pages", func() {
				p := query.NewPagination(1, 10)
				result := query.NewPaginatedResult(
					[]string{}, 0, p,
				)
				Expect(result.TotalPages).To(Equal(uint(0)))
			})
		})

		Context("when total items don't divide evenly by page size", func() {
			It("should round up total pages", func() {
				p := query.NewPagination(1, 10)
				result := query.NewPaginatedResult(
					[]string{}, 23, p,
				)
				Expect(result.TotalPages).To(Equal(uint(3)))
			})
		})
	})
})
