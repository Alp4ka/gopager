package gopager

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

// DefaultCursor представляет токен пагинации, определяющий начальную позицию
// для запрашиваемой страницы данных. Пустой токен означает начало набора данных.
//
// IMPORTANT:
// Токен ВСЕГДА должен содержать условие по уникальной колонке!
//
// Токен состоит из набора условий следующего вида:
//
//	[(C1, O1, V1), (C2, O2, V2)... (Cn, On, Vn)]
type DefaultCursor struct {
	elements []CursorElement
}

func NewCursor(elements ...CursorElement) *DefaultCursor {
	return NewDefaultCursor(elements...)
}

func NewDefaultCursor(elements ...CursorElement) *DefaultCursor {
	return &DefaultCursor{
		elements: elements,
	}
}

// DecodeCursor производит попытку распарсить закодированную (base64) строку в *DefaultCursor.
func DecodeCursor(b64String string) (*DefaultCursor, error) {
	if len(b64String) == 0 {
		return nil, nil
	}

	jsonData, err := _encoder.DecodeString(b64String)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 encoded cursor: %w", err)
	}

	var elems []CursorElement
	if err = json.Unmarshal(jsonData, &elems); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json encoded cursor: %w", err)
	}

	return &DefaultCursor{
		elements: elems,
	}, nil
}

// String - implements fmt.Stringer.
func (c *DefaultCursor) String() string {
	if c == nil || len(c.elements) == 0 {
		return ""
	}

	jTok, err := json.Marshal(c.elements)
	if err != nil {
		panic(fmt.Errorf("cannot marshal cursor value: %w", err))
	}

	var buf bytes.Buffer
	if err = json.Compact(&buf, jTok); err != nil {
		panic(fmt.Errorf("cannot compact cursor value: %w", err))
	}

	return _encoder.EncodeToString(buf.Bytes())
}

// IsEmpty - implements Cursor.
func (c *DefaultCursor) IsEmpty() bool {
	return c == nil || len(c.elements) == 0
}

// GetElements - возвращает элементы токена. Элементы токена представляют собой сжатый набор условий для фильтрации.
//
// IMPORTANT:
// Эти условия фильтрации нельзя применять к данным напрямую, т.к они не являются полными.
// В процессе пагинации они разжимаются в полный набор условий фильтрации.
func (c *DefaultCursor) GetElements() []CursorElement {
	if c == nil {
		return nil
	}

	return c.elements
}

// WithElements - указать элементы токена вручную.
func (c *DefaultCursor) WithElements(elements []CursorElement) *DefaultCursor {
	if c == nil {
		c = new(DefaultCursor)
	}

	c.elements = elements

	return c
}

// Apply - implements Cursor. Применяет сдвиг на основе фильтров к запросу gorm.
func (c *DefaultCursor) Apply(db *gorm.DB) *gorm.DB {
	exp := c.toDNF().toGORMExpression()
	if exp == nil {
		return db
	}

	return db.Clauses(exp)
}

// ToSQL - implements Cursor. Вернет строковое представление фильтра в виде SQL выражения.
//
// Использование:
//
//	query := fmt.Sprintf("SELECT * FROM table WHERE %s", p.ToSQL())
func (c *DefaultCursor) ToSQL() (string, []driver.Value) {
	if c.IsEmpty() {
		return "TRUE", nil
	}

	return c.toDNF().toSQLClause()
}

// toDNF - преобразует DefaultCursor в tDNF.
//
// IMPORTANT:
// Токен ВСЕГДА должен содержать условие по уникальной колонке!
//
// Токен состоит из набора условий следующего вида:
//
//	[(C1, O1, V1), (C2, O2, V2)... (Cn, On, Vn)]
//
// Последовательно применяя к этому набору условий преобразования(Inflate), получаем фильтр:
//
//	(C1 O1 V1) or (C1 = V1 and C2 O2 V2)
//
// В таком виде, токен представляет собой ДНФ, достаточный для фильтрации.
// Это позволяет однозначно определить позицию, с которой следует продолжить выборку данных.
func (c *DefaultCursor) toDNF() tDNF {
	if c.IsEmpty() {
		return nil
	}

	dnf := make(tDNF, 0, len(c.elements))
	for i := range c.elements {
		previousElementsWithEqualityCondition := lo.Map(c.elements[:i], func(item CursorElement, _ int) tConjunct {
			return item.toConjunctWithEqualityCondition()
		})

		disjunct := make([]tConjunct, 0, len(previousElementsWithEqualityCondition)+1)
		disjunct = append(disjunct, previousElementsWithEqualityCondition...)
		disjunct = append(disjunct, tConjunct(c.elements[i]))

		dnf = append(dnf, disjunct)
	}

	return dnf
}

// validate - implements Cursor.
func (c *DefaultCursor) validate(orderings Orderings) error {
	if c.IsEmpty() {
		return nil
	}

	// Не допускаем расхождений между количеством колонок в токене и в списке сортировки.
	if len(c.elements) != len(orderings) && len(c.elements) != 0 {
		return fmt.Errorf("cursor column number mismatch")
	}

	// Проверка соответствия сортировки и фильтров. Допускается пустой список элементов.
	for i := range c.elements {
		cond := c.elements[i]
		orderBy := orderings[i]

		// Проверяем совпадение имен колонок.
		if cond.Column != orderBy.Column {
			return fmt.Errorf("unexpected cursor column '%s'", cond.Column)
		}

		// Проверяем допустимость оператора.
		if !cond.Operator.Valid() {
			return fmt.Errorf("invalid cursor operator '%s'", cond.Operator)
		} else if cond.Operator.ForOrdering() != orderBy.Direction {
			return fmt.Errorf("unexpected cursor operator '%s'", cond.Operator)
		}
	}

	return nil
}

var (
	_ Cursor       = (*DefaultCursor)(nil)
	_ fmt.Stringer = (*DefaultCursor)(nil)
)

// Getters - словарь геттеров для объекта. Указывать те колонки, на основе которых производится пагинация.
// Пример:
//
//	pager.Getters[models.PlayerPushTarget]{
//		"id":          func(last models.PlayerPushTarget) any { return last.ID },
//		"deposit_sum": func(last models.PlayerPushTarget) any { return last.DepositSum },
//	}
type Getters[T any] map[string]func(T) any

// NextPageCursor - получить курсор для следующей страницы датасета.
func NextPageCursor[T any](
	initialPager *CursorPager[*DefaultCursor],
	resultSet []T,
	getters Getters[T],
) ([]T, *DefaultCursor, error) {
	err := initialPager.validate()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot build next page cursor: %w", err)
	}

	if IsLastPage(initialPager, resultSet) {
		return resultSet, nil, nil
	}
	resultSet = TrimResultSet(initialPager, resultSet)
	last := lo.LastOrEmpty(resultSet)

	ret := DefaultCursor{elements: nil}
	for _, orderBy := range initialPager.sort {
		getter, ok := getters[orderBy.Column]
		if !ok {
			return nil, nil, fmt.Errorf("cannot find getter for column '%s' met in ordering", orderBy.Column)
		}

		value := getter(last)
		ret.elements = append(ret.elements, CursorElement{
			Column:   orderBy.Column,
			Value:    value,
			Operator: orderBy.Direction.ForOperator(),
		})
	}

	return resultSet, &ret, nil
}

// CursorElement представляет тройку значений вида (c v o), где:
//
//   - "c" - поле объекта.
//   - "v" - значение, с которым сравниваем поле объекта.
//   - "o" - оператор, который применяем к паре (c, v);
type CursorElement struct {
	Column   string   `json:"c"`
	Value    any      `json:"v"`
	Operator Operator `json:"o"`
}

func (c *CursorElement) toConjunctWithEqualityCondition() tConjunct {
	return tConjunct{
		Column:   c.Column,
		Value:    c.Value,
		Operator: operatorEq,
	}
}
