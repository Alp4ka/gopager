package gopager

import "fmt"

// Operator defines a comparison operator for filtering by column.
// Used in pagination filtering conditions.
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

	// operatorEq is the equality operator. It is private because we use it
	// ONLY while building filtering conditions.
	operatorEq Operator = "="
)
