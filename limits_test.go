package gopager

import "testing"

func Test_IsNormalizedLimitMax(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		max      int
		want     int
		isStrict bool
	}{
		{"zero uses default", 0, 50, DefaultLimit, false},
		{"negative uses default", -10, 50, DefaultLimit, false},
		{"within max unchanged", 7, 50, 7, true},
		{"equal max unchanged", 50, 50, 50, true},
		{"above max clamped", 51, 50, 50, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, strict := IsNormalizedLimitMax(tt.limit, tt.max)
			if got != tt.want || strict != tt.isStrict {
				t.Errorf("%s: got=(%d,%v) want=(%d,%v)", tt.name, got, strict, tt.want, tt.isStrict)
			}
		})
	}
}

func Test_NormalizeLimitMax(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		max   int
		want  int
	}{
		{"zero -> default", 0, 77, DefaultLimit},
		{"negative -> default", -3, 77, DefaultLimit},
		{"clamp to max", 1000, 77, 77},
		{"keep when ok", 12, 77, 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeLimitMax(tt.limit, tt.max); got != tt.want {
				t.Errorf("%s: got %d want %d", tt.name, got, tt.want)
			}
		})
	}
}

func Test_NormalizeLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"zero -> default", 0, DefaultLimit},
		{"negative -> default", -1, DefaultLimit},
		{"clamp to MaxLimit", MaxLimit + 1, MaxLimit},
		{"keep when ok", 17, 17},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeLimit(tt.limit); got != tt.want {
				t.Errorf("%s: got %d want %d", tt.name, got, tt.want)
			}
		})
	}
}
