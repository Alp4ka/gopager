package gopager

import (
	"fmt"
	"math"
	"strings"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

// Direction defines the sort direction for the requested dataset.
type Direction string

const (
	DirectionASC  Direction = "ASC"
	DirectionDESC Direction = "DESC"
)

func (o Direction) Valid() bool {
	return o == DirectionASC || o == DirectionDESC
}

func (o Direction) ForOperator() Operator {
	switch o {
	case DirectionASC:
		return OperatorGT
	case DirectionDESC:
		return OperatorLT
	default:
		panic(fmt.Errorf("cannot map direction '%s' to operator", o))
	}
}

type (
	Orderings []OrderBy
	OrderBy   struct {
		Column    string
		Direction Direction
	}

	ColumnAlias = string

	// ColumnMapping maps external column aliases to fully qualified column names.
	// Use it when bare column names could cause an "ambiguous column name" error.
	// Key is an external alias, value is an internal column name.
	ColumnMapping = map[ColumnAlias]string
)

var _availableColumnNameSymbols = append([]rune("_.'`\""), lo.AlphanumericCharset...)

func (o OrderBy) validate() error {
	if !o.Direction.Valid() {
		return fmt.Errorf("invalid ordering direction '%s'", o.Direction)
	}

	// Guard against SQL injection by restricting allowed characters in column names.
	if !lo.Every(_availableColumnNameSymbols, []rune(o.Column)) {
		return fmt.Errorf("ordering column name contains forbidden symbols '%s'", o.Column)
	}

	return nil
}

// ToSQLSlice converts Orderings to a slice of strings in the form
// "<order_column> <order_direction>" suitable for SQL query builders.
//
// Example: for Orderings: [{"a", "ASC"}, {"b", "DESC"}] returns ["a ASC", "b DESC"].
func (o Orderings) ToSQLSlice() []string {
	ret := make([]string, 0, len(o))
	for _, ordering := range o {
		ret = append(ret, fmt.Sprintf("%s %s", ordering.Column, ordering.Direction))
	}

	return ret
}

// ToSQL converts Orderings to a single string
// "<order_column_1> <order_direction_1>, <order_column_2> <order_direction_2>"
// suitable for embedding into an SQL query.
// Example: for [{"a", "ASC"}, {"b", "DESC"}] returns "a ASC, b DESC".
//
// Usage:
//
//	query := fmt.Sprintf("SELECT * FROM table ORDER BY %s", orderings.ToSQL())
func (o Orderings) ToSQL() string {
	return strings.Join(o.ToSQLSlice(), ", ")
}

// Apply applies the ordering to a gorm query.
func (o Orderings) Apply(db *gorm.DB) *gorm.DB {
	return db.Order(o.ToSQL())
}

func (o Orderings) validate() error {
	if len(o) == 0 {
		return fmt.Errorf("empty ordering list")
	}

	var err error
	for _, ordering := range o {
		err = ordering.validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// ParseSort builds Orderings from a list of strings in the format
// "column asc|desc". Column aliases are resolved via ColumnMapping.
// Returns an error if an alias is not found in the mapping.
func ParseSort(stringsOrderings []string, columnMapping ColumnMapping) (Orderings, error) {
	ret := make([]OrderBy, 0, len(stringsOrderings))
	aliases := lo.Keys(columnMapping)

	for _, stringOrdering := range stringsOrderings {
		cutStringOrdering := strings.Split(strings.TrimSpace(stringOrdering), " ")
		if len(cutStringOrdering) != 2 {
			return nil, fmt.Errorf("invalid ordering string format '%s'", stringOrdering)
		}

		columnAlias := cutStringOrdering[0]
		direction := Direction(strings.ToUpper(cutStringOrdering[1]))
		columnName := columnMapping[columnAlias]
		if columnName == "" {
			return nil, fmt.Errorf("invalid column alias. closest: '%s'", closestAlias(columnAlias, aliases))
		}

		ret = append(ret, OrderBy{
			Column:    columnName,
			Direction: direction,
		})
	}

	return ret, nil
}

func closestAlias(input ColumnAlias, dataSet []ColumnAlias) ColumnAlias {
	minDist := math.MaxInt
	closest := ""

	for _, dataSetAlias := range dataSet {
		dist := levenshtein([]rune(dataSetAlias), []rune(input))
		if dist < minDist {
			minDist = dist
			closest = dataSetAlias
		}
	}

	return closest
}
