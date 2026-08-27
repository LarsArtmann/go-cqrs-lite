package query_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// TestNewPaginatedResult_ZeroPageSizeDoesNotPanic pins the zero-value
// Pagination path: TotalPages must be 0, not a divide-by-zero panic.
func TestNewPaginatedResult_ZeroPageSizeDoesNotPanic(t *testing.T) {
	t.Parallel()

	result := query.NewPaginatedResult([]string{"a"}, 10, query.Pagination{})
	if result.TotalPages != 0 {
		t.Errorf("TotalPages = %d, want 0 for zero PageSize", result.TotalPages)
	}

	if result.TotalCount != 10 {
		t.Errorf("TotalCount = %d, want 10", result.TotalCount)
	}
}
