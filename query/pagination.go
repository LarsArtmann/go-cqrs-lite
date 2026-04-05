package query

import "fmt"

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// Pagination controls page-based result sets.
type Pagination struct {
	Page     int
	PageSize int
}

// NewPagination creates pagination with defaults for zero values.
func NewPagination(page, pageSize int) Pagination {
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
	return (p.Page - 1) * p.PageSize
}

// PaginatedResult wraps a page of data with total count metadata.
type PaginatedResult[T any] struct {
	Data       []T
	TotalCount int
	Page       int
	PageSize   int
	TotalPages int
}

// NewPaginatedResult creates a paginated result, computing TotalPages.
func NewPaginatedResult[T any](data []T, totalCount int, p Pagination) PaginatedResult[T] {
	totalPages := 0
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
		return fmt.Errorf("page must be >= 1, got %d", p.Page)
	}

	if p.PageSize < 1 {
		return fmt.Errorf("page size must be >= 1, got %d", p.PageSize)
	}

	if p.PageSize > maxPageSize {
		return fmt.Errorf("page size must be <= %d, got %d", maxPageSize, p.PageSize)
	}

	return nil
}
