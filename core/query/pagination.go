package query

import (
	"strconv"

	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// Pagination controls page-based result sets.
// Uses uint to make negative values unrepresentable at the type level.
type Pagination struct {
	Page     uint
	PageSize uint
}

// NewPagination creates pagination with defaults for zero values.
func NewPagination(page, pageSize uint) Pagination {
	if page < 1 {
		page = defaultPage
	}

	if pageSize < 1 {
		pageSize = defaultPageSize
	}

	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return Pagination{Page: page, PageSize: pageSize}
}

// Offset calculates the zero-based skip for database queries.
func (p Pagination) Offset() int {
	return int((p.Page - 1) * p.PageSize)
}

// PaginatedResult wraps a page of data with total count metadata.
type PaginatedResult[T any] struct {
	Data       []T
	TotalCount uint
	Page       uint
	PageSize   uint
	TotalPages uint
}

// NewPaginatedResult creates a paginated result, computing TotalPages.
func NewPaginatedResult[T any](data []T, totalCount uint, p Pagination) PaginatedResult[T] {
	var totalPages uint
	if totalCount > 0 {
		totalPages = (totalCount + p.PageSize - 1) / p.PageSize
	}

	return PaginatedResult[T]{
		Data:       data,
		TotalCount: totalCount,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalPages: totalPages,
	}
}

// HasNext returns true if there is a next page.
func (r PaginatedResult[T]) HasNext() bool {
	return r.Page < r.TotalPages
}

// HasPrev returns true if there is a previous page.
func (r PaginatedResult[T]) HasPrev() bool {
	return r.Page > 1
}

// Validate checks pagination values are within bounds.
func (p Pagination) Validate() error {
	if p.Page < 1 {
		return errorfamily.NewRejection(
			"query.invalid_page",
			"page must be >= 1, got "+strconv.FormatUint(uint64(p.Page), 10),
		)
	}

	if p.PageSize < 1 {
		return errorfamily.NewRejection(
			"query.invalid_page_size",
			"page size must be >= 1, got "+strconv.FormatUint(uint64(p.PageSize), 10),
		)
	}

	if p.PageSize > maxPageSize {
		return errorfamily.NewRejection(
			"query.invalid_page_size",
			"page size must be <= "+strconv.Itoa(maxPageSize)+", got "+strconv.FormatUint(uint64(p.PageSize), 10),
		)
	}

	return nil
}
