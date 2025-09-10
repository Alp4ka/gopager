package pager

import "testing"

func Test_levenshtein(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{"equal -> 0", "kitten", "kitten", 0},
		{"classic kitten-sitting -> 3", "kitten", "sitting", 3},
		{"empty vs word -> len", "", "abc", 3},
		{"transposition like -> 2", "abcd", "abdc", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := levenshtein([]rune(tt.a), []rune(tt.b)); got != tt.want {
				t.Errorf("%s: got %d want %d", tt.name, got, tt.want)
			}
		})
	}
}

func Test_min3(t *testing.T) {
	tests := []struct{ a, b, c, want int }{
		{3, 2, 1, 1},
		{1, 3, 2, 1},
		{2, 1, 3, 1},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := min3(tt.a, tt.b, tt.c); got != tt.want {
				t.Errorf("min3(%d,%d,%d)=%d want %d", tt.a, tt.b, tt.c, got, tt.want)
			}
		})
	}
}
