package pager

import (
	"fmt"
	"math"
	"strings"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

// Direction определяет направление сортировки для запрашиваемого набора данных.
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

	// ColumnMapping - для маппинга алиасов колонок. Создан для тех кейсов, когда указание колонки без таблицы вызывает
	// 'ambiguous column name error'. Ключ - служит внешним отображением колонки, а значение - внутренним.
	ColumnMapping = map[ColumnAlias]string
)

var _availableColumnNameSymbols = append([]rune("_"), lo.AlphanumericCharset...)

func (o OrderBy) validate() error {
	if !o.Direction.Valid() {
		return fmt.Errorf("invalid ordering direction '%s'", o.Direction)
	}

	// Здесь защищаемся от SQL инъекций.
	if !lo.Every(_availableColumnNameSymbols, []rune(o.Column)) {
		return fmt.Errorf("ordering column name contains forbidden symbols '%s'", o.Column)
	}

	return nil
}

// ToSQLSlice конвертирует Orderings в слайс строк "<order_column> <order_direction>" для вставки
// в билдер SQL запросов.
//
// Например, для Orderings: [{"a", "ASC"}, {"b", "DESC"}] вернет слайс строк ["a ASC", "b DESC"].
func (o Orderings) ToSQLSlice() []string {
	ret := make([]string, 0, len(o))
	for _, ordering := range o {
		ret = append(ret, fmt.Sprintf("%s %s", ordering.Column, ordering.Direction))
	}

	return ret
}

// ToSQL конвертирует Orderings в строку "<order_column_1> <order_direction_1>, <order_column_2> <order_direction_2>"
// для вставки в SQL запрос.
// Например, для [{"a", "ASC"}, {"b", "DESC"}] вернет строку "a ASC, b DESC".
//
// Использование:
//
//	query := fmt.Sprintf("SELECT * FROM table ORDER BY %s", orderings.ToSQL())
func (o Orderings) ToSQL() string {
	return strings.Join(o.ToSQLSlice(), ", ")
}

// Apply применяет сортировку к запросу gorm.
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

// ParseSort - создает слайс сортировок на основе списка строк формата "column asc|desc".
// Также передается словарь с маппингами алиасов колонок в реальные колонку таблицы. См. ColumnMapping.
// Если в списке строк сортировок есть колонки, которых нет в маппинге, то вернется ошибка.
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
