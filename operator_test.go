package gopager

import "testing"

func Test_Operator_Valid_And_ForOrdering(t *testing.T) {
	tests := []struct {
		name     string
		in       Operator
		valid    bool
		ordering Direction
		panicExp bool
	}{
		{"GT valid maps to ASC", OperatorGT, true, DirectionASC, false},
		{"LT valid maps to DESC", OperatorLT, true, DirectionDESC, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Valid(); got != tt.valid {
				t.Errorf("%s: Valid=%v want %v", tt.name, got, tt.valid)
			}
			if !tt.panicExp {
				if got := tt.in.ForOrdering(); got != tt.ordering {
					t.Errorf("%s: ForOrdering=%v want %v", tt.name, got, tt.ordering)
				}
			}
		})
	}
}
