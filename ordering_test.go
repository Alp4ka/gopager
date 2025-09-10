package gopager

import (
	"testing"
)

func Test_Direction_Valid_And_ForOperator(t *testing.T) {
	tests := []struct {
		name     string
		in       Direction
		valid    bool
		operator Operator
		panicExp bool
	}{
		{"ASC valid maps to GT", DirectionASC, true, OperatorGT, false},
		{"DESC valid maps to LT", DirectionDESC, true, OperatorLT, false},
	}
	for _, tt := range tests {
		if got := tt.in.Valid(); got != tt.valid {
			t.Errorf("%s: Valid=%v want %v", tt.name, got, tt.valid)
		}
		if !tt.panicExp {
			if got := tt.in.ForOperator(); got != tt.operator {
				t.Errorf("%s: ForOperator=%v want %v", tt.name, got, tt.operator)
			}
		}
	}
}

func Test_Orderings_validate(t *testing.T) {
	tests := []struct {
		name string
		ord  Orderings
		ok   bool
	}{
		{"empty returns error", Orderings{}, false},
		{"invalid direction", Orderings{{Column: "id", Direction: "bad"}}, false},
		{"valid list", Orderings{{Column: "id", Direction: DirectionASC}}, true},
	}
	for _, tt := range tests {
		if err := tt.ord.validate(); (err == nil) != tt.ok {
			t.Errorf("%s: ok=%v err=%v", tt.name, tt.ok, err)
		}
	}
}

func Test_ParseSort(t *testing.T) {
	mapping := ColumnMapping{
		"id":   "t.id",
		"name": "t.name",
	}

	tests := []struct {
		name  string
		in    []string
		ok    bool
		first OrderBy
	}{
		{"invalid format", []string{"id"}, false, OrderBy{}},
		{"unknown alias", []string{"idx asc"}, false, OrderBy{}},
		{"valid asc", []string{"id asc"}, true, OrderBy{Column: "t.id", Direction: DirectionASC}},
		{"valid desc", []string{"name desc"}, true, OrderBy{Column: "t.name", Direction: DirectionDESC}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSort(tt.in, mapping)
			if (err == nil) != tt.ok {
				t.Errorf("%s: ok=%v err=%v", tt.name, tt.ok, err)
				return
			}
			if tt.ok {
				if len(got) == 0 || got[0] != tt.first {
					t.Errorf("%s: first=%v want %v", tt.name, got, tt.first)
				}
			}
		})
	}
}

func Test_closestAlias(t *testing.T) {
	aliases := []ColumnAlias{"id", "name", "created_at"}
	tests := []struct {
		name string
		in   ColumnAlias
		out  ColumnAlias
	}{
		{"closest to id", "idx", "id"},
		{"closest to name", "nme", "name"},
		{"closest to created_at", "createdat", "created_at"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := closestAlias(tt.in, aliases); got != tt.out {
				t.Errorf("%s: got %s want %s", tt.name, got, tt.out)
			}
		})
	}
}
