package pager

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

	// tDNF - представляет дизъюнктивную нормальную форму логического выражения.
	// К каждому дизъюнкту применяется операция логического ИЛИ. При этом каждый дизъюнкт состоит
	// из списка конъюнктов, к каждому из которых применяется логическая операция И. В
	// роли конъюнкта выступает значение выражения Operator(Column, Value).
	//
	// Так,
	//
	//	ДНФ = X1 ИЛИ X2 ... ИЛИ Xn, при Xi = Ai1 И Ai2 ... И Aim.
	//	ДНФ = (A11 И A12 И A13) ИЛИ (A21 И A22 И A23), при n=2, m=3.
	//
	// 	Где (A11 И A12 И A13), (A21 И A22 И A23) - дизъюнкты.
	// 	A11, A12, A13, A21, A22, A23 - конъюнкты.
	tDNF []tDisjunct
)

// toGORMExpression переводит конъюнкт вида Operator(Column, Value) в SQL условие
// вида "Column Operator Value". Необходимо для совместимости с gorm.
//
// IMPORTANT: Метод использует SQL placeholder "?".
//
// Например:
//
//	tConjunct = { Column: "id", Operator: ">", Value: "123"}
//
// Результат:
//
//	"id > 123"
func (c tConjunct) toGORMExpression() clause.Expression {
	sqlClause, arg := c.toSQLClause()

	return clause.Expr{
		SQL:  sqlClause,
		Vars: []any{arg},
	}
}

// toSQLClause переводит конъюнкт вида Operator(Column, Value) в SQL условие
// вида "Column Operator ?" с соответствующим значением. Возвращает SQL строку
// и значение для подстановки в placeholder.
//
// Например:
//
//	tConjunct = { Column: "id", Operator: ">", Value: 123}
//
// Результат:
//
//	("id > ?", 123)
func (c tConjunct) toSQLClause() (string, driver.Value) {
	return fmt.Sprintf("%s %s ?", c.Column, c.Operator), parseAnyValue(c.Value)
}

func parseAnyValue(v any) any {
	// Парсит значение в Time. Если получается, возвращает Time, если нет - то же значение, что было передано.
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

// toGORMExpression переводит дизъюнкт вида (K1, K2, K3) в SQL условие
// вида "K1 AND K2 AND K3". При этом раскрывает каждый Кi (конъюнкт), используя
// tConjunct.toGORMExpression. Необходимо для совместимости с gorm.
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

// toSQLClause переводит дизъюнкт вида (K1, K2, K3) в SQL условие
// вида "(K1 AND K2 AND K3)" с соответствующими значениями. Возвращает SQL строку
// и массив значений для подстановки в placeholders.
//
// Например:
//
//	tDisjunct = {
//		{Column: "id", Operator: ">", Value: 5},
//		{Column: "name", Operator: "<", Value: "abc"}
//	}
//
// Результат:
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

// toGORMExpression переводит логическую запись ДНФ(tDNF) в clause.Expression.
// Для каждого дизъюнкта вызывается tDisjunct.toGORMExpression. Дизъюнкты объединяются через логическое ИЛИ.
// Необходимо для совместимости с gorm.
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

// toSQLClause переводит логическую запись ДНФ(tDNF) в SQL условие.
// Для каждого дизъюнкта вызывается tDisjunct.toSQLClause. Дизъюнкты объединяются
// через логическое ИЛИ. Возвращает SQL строку и массив значений для подстановки
// в placeholders.
//
// Например:
//
//	tDNF = {
//		{{Column: "id", Operator: "<", Value: 10}}
//		{{Column: "id", Operator: "=", Value: 10}, {Column: "name", Operator: "<", Value: "abc"}},
//	}
//
// Результат:
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
