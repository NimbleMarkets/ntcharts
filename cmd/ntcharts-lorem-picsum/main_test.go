package main

import "testing"

func TestCatalogIndicesSortAndFilter(t *testing.T) {
	items := []picsumItem{
		{ID: "10", Author: "Zoe"},
		{ID: "2", Author: "Ada"},
		{ID: "1", Author: "ada lovelace"},
	}

	tests := []struct {
		name   string
		filter string
		sortBy catalogSort
		want   []int
	}{
		{name: "id asc", sortBy: sortIDAsc, want: []int{2, 1, 0}},
		{name: "id desc", sortBy: sortIDDesc, want: []int{0, 1, 2}},
		{name: "author asc", sortBy: sortAuthorAsc, want: []int{1, 2, 0}},
		{name: "author desc", sortBy: sortAuthorDesc, want: []int{0, 2, 1}},
		{name: "author filter", filter: "ada", sortBy: sortIDAsc, want: []int{2, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := catalogIndices(items, tt.filter, tt.sortBy)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d (%v)", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("catalogIndices() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
