package data

import (
	"testing"

	"github.com/daggix02-m/greenlight-let-s-go-further-/internal/validator"
)

// movieSortSafelist mirrors the safelist the API sets on incoming requests.
var movieSortSafelist = []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"}

func TestValidateFilters(t *testing.T) {
	safelist := movieSortSafelist

	tests := []struct {
		name    string
		filters Filters
		wantErr string
	}{
		{
			"valid filters",
			Filters{Page: 1, PageSize: 20, Sort: "title", SortSafelist: safelist},
			"",
		},
		{
			"page below one",
			Filters{Page: 0, PageSize: 20, Sort: "id", SortSafelist: safelist},
			"page",
		},
		{
			"page above maximum",
			Filters{Page: 10_000_001, PageSize: 20, Sort: "id", SortSafelist: safelist},
			"page",
		},
		{
			"page size below one",
			Filters{Page: 1, PageSize: 0, Sort: "id", SortSafelist: safelist},
			"page_size",
		},
		{
			"page size above maximum",
			Filters{Page: 1, PageSize: 101, Sort: "id", SortSafelist: safelist},
			"page_size",
		},
		{
			"sort value not in safelist",
			Filters{Page: 1, PageSize: 20, Sort: "secret", SortSafelist: safelist},
			"sort",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidateFilters(v, tt.filters)

			if tt.wantErr == "" {
				if !v.Valid() {
					t.Fatalf("ValidateFilters() errors = %v, want none", v.Errors)
				}
				return
			}

			if v.Valid() {
				t.Fatalf("ValidateFilters() errors = none, want an error for %q", tt.wantErr)
			}
			if _, exists := v.Errors[tt.wantErr]; !exists {
				t.Errorf("ValidateFilters() errors = %v, want key %q", v.Errors, tt.wantErr)
			}
		})
	}
}

func TestFiltersSortColumnAndDirection(t *testing.T) {
	tests := []struct {
		name       string
		sort       string
		wantColumn string
		wantDir    string
	}{
		{"ascending column", "title", "title", "ASC"},
		{"descending column", "-title", "title", "DESC"},
		{"default id", "id", "id", "ASC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Filters{Sort: tt.sort, SortSafelist: movieSortSafelist}

			if got := f.sortColumn(); got != tt.wantColumn {
				t.Errorf("sortColumn() = %q, want %q", got, tt.wantColumn)
			}
			if got := f.sortDirection(); got != tt.wantDir {
				t.Errorf("sortDirection() = %q, want %q", got, tt.wantDir)
			}
		})
	}
}

func TestFiltersSortColumnPanicsOnUnsafeValue(t *testing.T) {
	f := Filters{Sort: "1; DROP TABLE movies", SortSafelist: movieSortSafelist}

	defer func() {
		if recover() == nil {
			t.Error("sortColumn() should panic for a value outside the safelist")
		}
	}()

	f.sortColumn()
}

func TestFiltersLimitAndOffset(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantLimit  int
		wantOffset int
	}{
		{"first page", 1, 20, 20, 0},
		{"second page", 2, 20, 20, 20},
		{"later page", 5, 10, 10, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Filters{Page: tt.page, PageSize: tt.pageSize}
			if got := f.limit(); got != tt.wantLimit {
				t.Errorf("limit() = %d, want %d", got, tt.wantLimit)
			}
			if got := f.offset(); got != tt.wantOffset {
				t.Errorf("offset() = %d, want %d", got, tt.wantOffset)
			}
		})
	}
}

func TestCalculateMetadata(t *testing.T) {
	if got := calculateMetadata(0, 1, 20); got != (Metadata{}) {
		t.Errorf("calculateMetadata(0, ...) = %+v, want empty Metadata", got)
	}

	want := Metadata{
		CurrentPage:  3,
		PageSize:     20,
		FirstPage:    1,
		LastPage:     5,
		TotalRecords: 81,
	}
	if got := calculateMetadata(81, 3, 20); got != want {
		t.Errorf("calculateMetadata(81, 3, 20) = %+v, want %+v", got, want)
	}
}
