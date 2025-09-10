package pager

import (
	"database/sql/driver"
	"testing"
	"time"

	"gorm.io/gorm/clause"
)

func Test_tConjunct_toExpression(t *testing.T) {
	timeNow := time.Now().UTC()
	timeNowStr, _ := timeNow.MarshalText()

	tests := []struct {
		name     string
		conjunct tConjunct
		wantSQL  string
		wantVars []interface{}
	}{
		{
			name:     "string less than",
			conjunct: tConjunct{Column: "name", Operator: OperatorLT, Value: "abc"},
			wantSQL:  "name < ?",
			wantVars: []interface{}{"abc"},
		},
		{
			name:     "timestamp greater than",
			conjunct: tConjunct{Column: "created_at", Operator: OperatorGT, Value: timeNow},
			wantSQL:  "created_at > ?",
			wantVars: []interface{}{timeNow},
		},
		{
			name:     "timestamp string should convert to timestamp",
			conjunct: tConjunct{Column: "created_at", Operator: OperatorGT, Value: timeNowStr},
			wantSQL:  "created_at > ?",
			wantVars: []interface{}{timeNow},
		},
		{
			name:     "integer less than",
			conjunct: tConjunct{Column: "id", Operator: OperatorLT, Value: 10},
			wantSQL:  "id < ?",
			wantVars: []interface{}{10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := tt.conjunct.toGORMExpression()
			clauseExpr := expr.(clause.Expr)

			if clauseExpr.SQL != tt.wantSQL {
				t.Errorf("unexpected SQL: got %s, want %s", clauseExpr.SQL, tt.wantSQL)
			}

			if len(clauseExpr.Vars) != len(tt.wantVars) {
				t.Errorf("unexpected vars length: got %d, want %d", len(clauseExpr.Vars), len(tt.wantVars))
			}

			for i, wantVar := range tt.wantVars {
				if clauseExpr.Vars[i] != wantVar {
					t.Errorf("unexpected var[%d]: got %v, want %v", i, clauseExpr.Vars[i], wantVar)
				}
			}
		})
	}
}

func Test_tDisjunct_toExpression(t *testing.T) {
	tests := []struct {
		name     string
		disjunct tDisjunct
		wantNil  bool
	}{
		{
			name: "non-empty disjunct",
			disjunct: tDisjunct{
				{Column: "id", Operator: OperatorGT, Value: 5},
				{Column: "created_at", Operator: OperatorGT, Value: "2024-01-02T03:04:05Z"},
			},
			wantNil: false,
		},
		{
			name:     "empty disjunct",
			disjunct: tDisjunct{},
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := tt.disjunct.toGORMExpression()
			if (expr == nil) != tt.wantNil {
				t.Errorf("unexpected expression result: got %v, want nil=%v", expr, tt.wantNil)
			}
		})
	}
}

func Test_tDNF_toExpression(t *testing.T) {
	tests := []struct {
		name    string
		dnf     tDNF
		wantNil bool
	}{
		{
			name: "non-empty DNF",
			dnf: tDNF{
				{
					{Column: "id", Operator: OperatorGT, Value: 5},
					{Column: "created_at", Operator: OperatorGT, Value: "2024-01-02T03:04:05Z"},
				},
				{{Column: "id", Operator: OperatorGT, Value: 10}},
			},
			wantNil: false,
		},
		{
			name:    "empty DNF",
			dnf:     tDNF{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := tt.dnf.toGORMExpression()
			if (expr == nil) != tt.wantNil {
				t.Errorf("unexpected expression result: got %v, want nil=%v", expr, tt.wantNil)
			}
		})
	}
}

func Test_tConjunct_toSQLClause(t *testing.T) {
	timeNow := time.Now().UTC()
	timeNowStr, _ := timeNow.MarshalText()

	tests := []struct {
		name     string
		conjunct tConjunct
		wantSQL  string
		wantVal  driver.Value
	}{
		{
			name:     "string less than",
			conjunct: tConjunct{Column: "name", Operator: OperatorLT, Value: "abc"},
			wantSQL:  "name < ?",
			wantVal:  "abc",
		},
		{
			name:     "timestamp greater than",
			conjunct: tConjunct{Column: "created_at", Operator: OperatorGT, Value: timeNow},
			wantSQL:  "created_at > ?",
			wantVal:  timeNow,
		},
		{
			name:     "timestamp string should convert to timestamp",
			conjunct: tConjunct{Column: "created_at", Operator: OperatorGT, Value: timeNowStr},
			wantSQL:  "created_at > ?",
			wantVal:  timeNow,
		},
		{
			name:     "integer less than",
			conjunct: tConjunct{Column: "id", Operator: OperatorLT, Value: 10},
			wantSQL:  "id < ?",
			wantVal:  10,
		},
		{
			name:     "float greater than",
			conjunct: tConjunct{Column: "price", Operator: OperatorGT, Value: 99.99},
			wantSQL:  "price > ?",
			wantVal:  99.99,
		},
		{
			name:     "boolean less than",
			conjunct: tConjunct{Column: "active", Operator: OperatorLT, Value: true},
			wantSQL:  "active < ?",
			wantVal:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotVal := tt.conjunct.toSQLClause()

			if gotSQL != tt.wantSQL {
				t.Errorf("toSQLClause() SQL = %v, want %v", gotSQL, tt.wantSQL)
			}

			if gotVal != tt.wantVal {
				t.Errorf("toSQLClause() Val = %v, want %v", gotVal, tt.wantVal)
			}
		})
	}
}

func Test_tDisjunct_toSQLClause(t *testing.T) {
	timeNow := time.Now().UTC()
	timeNowStr, _ := timeNow.MarshalText()

	tests := []struct {
		name     string
		disjunct tDisjunct
		wantSQL  string
		wantVals []driver.Value
	}{
		{
			name: "single conjunct",
			disjunct: tDisjunct{
				{Column: "id", Operator: OperatorGT, Value: 5},
			},
			wantSQL:  "(id > ?)",
			wantVals: []driver.Value{5},
		},
		{
			name: "multiple conjuncts",
			disjunct: tDisjunct{
				{Column: "id", Operator: OperatorGT, Value: 5},
				{Column: "name", Operator: OperatorLT, Value: "abc"},
				{Column: "active", Operator: OperatorGT, Value: true},
			},
			wantSQL:  "(id > ? AND name < ? AND active > ?)",
			wantVals: []driver.Value{5, "abc", true},
		},
		{
			name: "timestamp conversion",
			disjunct: tDisjunct{
				{Column: "created_at", Operator: OperatorGT, Value: timeNowStr},
				{Column: "updated_at", Operator: OperatorLT, Value: timeNow},
			},
			wantSQL:  "(created_at > ? AND updated_at < ?)",
			wantVals: []driver.Value{timeNow, timeNow},
		},
		{
			name:     "empty disjunct",
			disjunct: tDisjunct{},
			wantSQL:  "",
			wantVals: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotVals := tt.disjunct.toSQLClause()

			if gotSQL != tt.wantSQL {
				t.Errorf("toSQLClause() SQL = %v, want %v", gotSQL, tt.wantSQL)
			}

			if len(gotVals) != len(tt.wantVals) {
				t.Errorf("toSQLClause() Vals length = %v, want %v", len(gotVals), len(tt.wantVals))
			}

			for i, wantVal := range tt.wantVals {
				if gotVals[i] != wantVal {
					t.Errorf("toSQLClause() Vals[%d] = %v, want %v", i, gotVals[i], wantVal)
				}
			}
		})
	}
}

func Test_tDNF_toSQLClause(t *testing.T) {
	timeNow := time.Now().UTC()
	timeNowStr, _ := timeNow.MarshalText()

	tests := []struct {
		name     string
		dnf      tDNF
		wantSQL  string
		wantVals []driver.Value
	}{
		{
			name: "single disjunct with single conjunct",
			dnf: tDNF{
				{{Column: "id", Operator: OperatorGT, Value: 5}},
			},
			wantSQL:  "((id > ?))",
			wantVals: []driver.Value{5},
		},
		{
			name: "single disjunct with multiple conjuncts",
			dnf: tDNF{
				{
					{Column: "id", Operator: OperatorGT, Value: 5},
					{Column: "name", Operator: OperatorLT, Value: "abc"},
				},
			},
			wantSQL:  "((id > ? AND name < ?))",
			wantVals: []driver.Value{5, "abc"},
		},
		{
			name: "multiple disjuncts",
			dnf: tDNF{
				{
					{Column: "id", Operator: OperatorGT, Value: 5},
					{Column: "name", Operator: OperatorLT, Value: "abc"},
				},
				{
					{Column: "id", Operator: OperatorGT, Value: 10},
				},
			},
			wantSQL:  "((id > ? AND name < ?) OR (id > ?))",
			wantVals: []driver.Value{5, "abc", 10},
		},
		{
			name: "complex DNF with timestamp conversion",
			dnf: tDNF{
				{
					{Column: "created_at", Operator: OperatorGT, Value: timeNowStr},
					{Column: "active", Operator: OperatorLT, Value: true},
				},
				{
					{Column: "id", Operator: OperatorGT, Value: 100},
					{Column: "price", Operator: OperatorLT, Value: 99.99},
				},
			},
			wantSQL:  "((created_at > ? AND active < ?) OR (id > ? AND price < ?))",
			wantVals: []driver.Value{timeNow, true, 100, 99.99},
		},
		{
			name:     "empty DNF",
			dnf:      tDNF{},
			wantSQL:  "TRUE",
			wantVals: nil,
		},
		{
			name: "DNF with empty disjuncts",
			dnf: tDNF{
				{},
				{{Column: "id", Operator: OperatorGT, Value: 5}},
				{},
			},
			wantSQL:  "((id > ?))",
			wantVals: []driver.Value{5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotVals := tt.dnf.toSQLClause()

			if gotSQL != tt.wantSQL {
				t.Errorf("toSQLClause() SQL = %v, want %v", gotSQL, tt.wantSQL)
			}

			if len(gotVals) != len(tt.wantVals) {
				t.Errorf("toSQLClause() Vals length = %v, want %v", len(gotVals), len(tt.wantVals))
			}

			for i, wantVal := range tt.wantVals {
				if gotVals[i] != wantVal {
					t.Errorf("toSQLClause() Vals[%d] = %v, want %v", i, gotVals[i], wantVal)
				}
			}
		})
	}
}
