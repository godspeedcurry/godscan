package utils

import (
	"reflect"
	"testing"
)

func TestDeduplicate(t *testing.T) {
	tests := []struct {
		name string
		input []int
		want  []int
	}{
		{"empty", []int{}, []int{}},
		{"no_dups", []int{1, 2, 3}, []int{1, 2, 3}},
		{"dups", []int{1, 2, 2, 3, 1}, []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Deduplicate(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Deduplicate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveDuplicatesString(t *testing.T) {
	tests := []struct {
		name string
		input []string
		want  []string
	}{
		{"sort_and_dedup", []string{"b", "a", "a", "c", "====skip"}, []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveDuplicatesString(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				// note: deep equal checks nil vs empty slice too, but here we expect basic match
				t.Errorf("RemoveDuplicatesString() = %v, want %v", got, tt.want)
			}
		})
	}
}
