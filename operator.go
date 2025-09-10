package gopager

import "fmt"

// Operator определяет оператор сравнения для фильтрации по колонке.
// Используется в условиях фильтрации при пагинации.
type Operator string

func (o Operator) Valid() bool {
	return o == OperatorLT || o == OperatorGT
}

func (o Operator) ForOrdering() Direction {
	switch o {
	case OperatorGT:
		return DirectionASC
	case OperatorLT:
		return DirectionDESC
	default:
		panic(fmt.Errorf("cannot map operator '%s' to ordering", o))
	}
}

const (
	OperatorGT Operator = ">"
	OperatorLT Operator = "<"

	// operatorEq - это оператор равенства. Приватность обусловлена тем,
	// что мы используем оператор ТОЛЬКО при построении условия фильтрации.
	operatorEq Operator = "="
)
