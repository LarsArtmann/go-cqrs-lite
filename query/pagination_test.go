package query

import "testing"

func TestNewPagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
	}{
		{"defaults for zeros", 0, 0, defaultPage, defaultPageSize},
		{"negative page defaults", -1, 10, defaultPage, 10},
		{"negative page size defaults", 1, -1, 1, defaultPageSize},
		{"page size capped at max", 1, 200, 1, maxPageSize},
		{"valid values kept", 3, 50, 3, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewPagination(tt.page, tt.pageSize)
			if p.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", p.Page, tt.wantPage)
			}

			if p.PageSize != tt.wantPageSize {
				t.Errorf("PageSize = %d, want %d", p.PageSize, tt.wantPageSize)
			}
		})
	}
}

func TestPagination_Offset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantOffset int
	}{
		{"first page", 1, 20, 0},
		{"second page", 2, 20, 20},
		{"third page", 3, 10, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewPagination(tt.page, tt.pageSize)
			if got := p.Offset(); got != tt.wantOffset {
				t.Errorf("Offset() = %d, want %d", got, tt.wantOffset)
			}
		})
	}
}

func TestPagination_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		page    int
		pageSz  int
		wantErr bool
	}{
		{"valid", 1, 20, false},
		{"zero page", 0, 20, true},
		{"zero page size", 1, 0, true},
		{"negative page", -1, 20, true},
		{"over max page size", 1, 200, true},
		{"max page size valid", 1, maxPageSize, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := Pagination{Page: tt.page, PageSize: tt.pageSz}
			err := p.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewPaginatedResult(t *testing.T) {
	t.Parallel()

	data := []string{"a", "b", "c"}
	p := NewPagination(2, 3)

	result := NewPaginatedResult(data, 10, p)

	if len(result.Data) != 3 {
		t.Errorf("Data len = %d, want 3", len(result.Data))
	}

	if result.TotalCount != 10 {
		t.Errorf("TotalCount = %d, want 10", result.TotalCount)
	}

	if result.TotalPages != 4 {
		t.Errorf("TotalPages = %d, want 4", result.TotalPages)
	}

	if !result.HasNext() {
		t.Error("HasNext() = false, want true")
	}

	if !result.HasPrev() {
		t.Error("HasPrev() = false, want true")
	}
}

func TestPaginatedResult_HasNext_HasPrev(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		page     int
		total    int
		wantNext bool
		wantPrev bool
	}{
		{"first page of many", 1, 100, true, false},
		{"middle page", 3, 100, true, true},
		{"last page", 4, 10, false, true},
		{"only page", 1, 3, false, false},
		{"empty", 1, 0, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewPagination(tt.page, 10)
			result := NewPaginatedResult([]string{}, tt.total, p)

			if result.HasNext() != tt.wantNext {
				t.Errorf("HasNext() = %v, want %v", result.HasNext(), tt.wantNext)
			}

			if result.HasPrev() != tt.wantPrev {
				t.Errorf("HasPrev() = %v, want %v", result.HasPrev(), tt.wantPrev)
			}
		})
	}
}
