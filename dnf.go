package gopager

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm/clause"
)

type (
	tConjunct struct {
		Column   string
		Value    any
		Operator Operator
	}

	tDisjunct []tConjunct

	// tDNF represents the disjunctive normal form (DNF) of a logical expression.
	// Each disjunct is joined by OR, and each disjunct consists of a list of
	// conjuncts which are joined by AND. A conjunct is the value of
	// Operator(Column, Value).
	//
	// Thus:
	//
	//	DNF = X1 OR X2 ... OR Xn, where Xi = Ai1 AND Ai2 ... AND Aim.
	//	DNF = (A11 AND A12 AND A13) OR (A21 AND A22 AND A23), for n=2, m=3.
	//
	//  Where (A11 AND A12 AND A13), (A21 AND A22 AND A23) are disjuncts and
	//  A11, A12, A13, A21, A22, A23 are conjuncts.
	tDNF []tDisjunct
)

// toGORMExpression converts a conjunct of the form Operator(Column, Value)
// into an SQL condition "Column Operator Value" represented as a clause.Expression.
//
// IMPORTANT: The method uses the SQL placeholder "?".
//
// Example:
//
//	tConjunct = { Column: "id", Operator: ">", Value: "123"}
//
// Result:
//
//	"id > 123"
func (c tConjunct) toGORMExpression() clause.Expression {
	sqlClause, arg := c.toSQLClause()

	return clause.Expr{
		SQL:  sqlClause,
		Vars: []any{arg},
	}
}

// toSQLClause converts a conjunct of the form Operator(Column, Value) to
// an SQL condition of the form "Column Operator ?" with a corresponding value.
// Returns the SQL string and the value for the placeholder.
//
// Example:
//
//	tConjunct = { Column: "id", Operator: ">", Value: 123}
//
// Result:
//
//	("id > ?", 123)
func (c tConjunct) toSQLClause() (string, driver.Value) {
	return fmt.Sprintf("%s %s ?", c.Column, c.Operator), parseAnyValue(c.Value)
}

func parseAnyValue(v any) any {
	// Try parsing a value as time.Time. If it succeeds, return time.Time.
	// Otherwise return the original value.
	fnParseBytesToTimeOrValue := func(vBytes []byte) any {
		dst := time.Time{}
		err := dst.UnmarshalText(vBytes)
		if err == nil {
			return dst
		}

		return v
	}

	switch vt := v.(type) {
	case string:
		return fnParseBytesToTimeOrValue([]byte(vt))
	case []byte:
		return fnParseBytesToTimeOrValue(vt)
	default:
		return v
	}
}

// toGORMExpression converts a disjunct (K1, K2, K3) into a gorm expression
// "K1 AND K2 AND K3" where each Ki is expanded via tConjunct.toGORMExpression.
func (d tDisjunct) toGORMExpression() clause.Expression {
	andExpressions := make([]clause.Expression, 0, len(d))
	for _, conjunct := range d {
		andExpressions = append(andExpressions, conjunct.toGORMExpression())
	}

	if len(andExpressions) == 1 {
		return andExpressions[0]
	} else if len(andExpressions) > 1 {
		return clause.And(andExpressions...)
	}

	return nil
}

// toSQLClause converts a disjunct (K1, K2, K3) into an SQL condition
// "(K1 AND K2 AND K3)" with corresponding values. Returns the SQL string and
// the list of values for placeholders.
//
// Example:
//
//	tDisjunct = {
//		{Column: "id", Operator: ">", Value: 5},
//		{Column: "name", Operator: "<", Value: "abc"}
//	}
//
// Result:
//
//	("(id > ? AND name < ?)", [5, "abc"])
func (d tDisjunct) toSQLClause() (string, []driver.Value) {
	andClauses := make([]string, 0, len(d))
	andValues := make([]driver.Value, 0, len(d))

	for _, conjunct := range d {
		andClause, andValue := conjunct.toSQLClause()
		andClauses = append(andClauses, andClause)
		andValues = append(andValues, andValue)
	}

	if len(andClauses) >= 1 {
		return fmt.Sprintf("(%s)", strings.Join(andClauses, " AND ")), andValues
	}

	return "", nil
}

// toGORMExpression converts a DNF (tDNF) into a clause.Expression.
// For each disjunct it calls tDisjunct.toGORMExpression and joins disjuncts with OR.
func (d tDNF) toGORMExpression() clause.Expression {
	orExpressions := make([]clause.Expression, 0, len(d))

	for _, disjunct := range d {
		andExpressions := disjunct.toGORMExpression()
		if andExpressions == nil {
			continue
		}

		orExpressions = append(orExpressions, andExpressions)
	}

	if len(orExpressions) == 1 {
		return orExpressions[0]
	} else if len(orExpressions) > 1 {
		return clause.Or(orExpressions...)
	}

	return nil
}

// toSQLClause converts a DNF (tDNF) into an SQL condition. For each disjunct it
// calls tDisjunct.toSQLClause and joins disjuncts with OR. Returns the SQL
// string and the list of values for placeholders.
//
// Example:
//
//	tDNF = {
//		{{Column: "id", Operator: "<", Value: 10}},
//		{{Column: "id", Operator: "=", Value: 10}, {Column: "name", Operator: "<", Value: "abc"}},
//	}
//
// Result:
//
//	("((id < ?) OR (id = ? AND name < ?))", [10, 10, "abc"])
func (d tDNF) toSQLClause() (string, []driver.Value) {
	orClauses := make([]string, 0, len(d))
	values := make([]driver.Value, 0, len(d))

	for _, disjunct := range d {
		orClause, orValues := disjunct.toSQLClause()
		if orClause == "" {
			continue
		}

		orClauses = append(orClauses, orClause)
		values = append(values, orValues...)
	}

	if len(orClauses) >= 1 {
		return fmt.Sprintf("(%s)", strings.Join(orClauses, " OR ")), values
	}

	return "TRUE", nil
}
