package pager

import (
	"fmt"
	"slices"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

// RawCursorPager - структура для слоя представления данных. Для корректной кодогенерации использовать инлайнинг:
//
//	type MyFilter struct {
//	    Paging RawCursorPager `json:",inline"`
//	}
type RawCursorPager struct {
	// Limit - максимальное количество записей в ответе на запрос.
	Limit int `json:"limit"`
	// StartToken - строка с закодированным токеном курсора. Получается путем вызова Cursor.String().
	// Если передать пустое значение - в ответе на запрос вернется первая страница с указанным Limit записей.
	StartToken string `json:"startToken"`
}

// Decode переводит RawCursorPager в *CursorPager[*DefaultCursor], нормализуя Limit и валидируя StartToken.
// Возвращает *CursorPager[*DefaultCursor] с примененным *CursorPager.WithSort
func (p RawCursorPager) Decode(orderBy ...OrderBy) (*CursorPager[*DefaultCursor], error) {
	return DecodeCursorPager(p.Limit, p.StartToken, orderBy...)
}

// DecodePseudo переводит RawCursorPager в *CursorPager[*PseudoCursor], нормализуя Limit и валидируя StartToken.
// Возвращает *CursorPager[*PseudoCursor] с примененным *CursorPager.WithSort
func (p RawCursorPager) DecodePseudo(orderBy ...OrderBy) (*CursorPager[*PseudoCursor], error) {
	return DecodePseudoCursorPager(p.Limit, p.StartToken, orderBy...)
}

type CursorPager[CursorType Cursor] struct {
	lookahead bool
	limit     int
	cursor    CursorType
	sort      Orderings
}

func NewCursorPager[CursorType Cursor]() *CursorPager[CursorType] {
	return new(CursorPager[CursorType])
}

// DecodeCursorPager - декодирует токен курсора в *CursorPager.
//
// Руководство по использованию: https://doc.office.lan/spaces/MBCSHCH/pages/417057947
func DecodeCursorPager(limit int, rawStartToken string, orderBy ...OrderBy) (*CursorPager[*DefaultCursor], error) {
	cursor, err := DecodeCursor(rawStartToken)
	if err != nil {
		return nil, err
	}

	return (&CursorPager[*DefaultCursor]{
		cursor: cursor,
	}).WithSubstitutedSort(orderBy...).WithLimit(limit), nil
}

// DecodePseudoCursorPager - декодирует токен псевдо-курсора в *CursorPager.
//
// Руководство по использованию: https://doc.office.lan/spaces/MBCSHCH/pages/417057947
func DecodePseudoCursorPager(limit int, rawStartToken string, orderBy ...OrderBy) (*CursorPager[*PseudoCursor], error) {
	cursor, err := DecodePseudoCursor(rawStartToken)
	if err != nil {
		return nil, err
	}

	return (&CursorPager[*PseudoCursor]{
		cursor: cursor,
	}).WithSubstitutedSort(orderBy...).WithLimit(limit), nil
}

// WithLookahead использует пагинацию с 'заглядыванием' на следующую страницу.
// Lookahead позволяет определить, является ли текущая страница последней.
//
// IMPORTANT:
// Нельзя применить WithLookahead к CursorPager с WithUnlimited() или WithLimit(NoLimit).
func (c *CursorPager[CursorType]) WithLookahead() *CursorPager[CursorType] {
	if c == nil {
		c = new(CursorPager[CursorType])
	}

	c.lookahead = true

	return c
}

// WithUnlimited позволяет возвращать все записи, неограниченное количество.
//
// IMPORTANT:
// Нельзя применить WithUnlimited к CursorPager с WithLookahead.
func (c *CursorPager[CursorType]) WithUnlimited() *CursorPager[CursorType] {
	if c == nil {
		c = new(CursorPager[CursorType])
	}

	c.limit = NoLimit

	return c
}

// WithLimit позволяет задать лимит возвращаемых записей.
//
// IMPORTANT:
//   - Нельзя применить NoLimit к CursorPager с WithLookahead.
//   - Если указанный лимит не равен NoLimit, то к нему будет применена функция NormalizeLimit.
func (c *CursorPager[CursorType]) WithLimit(limit int) *CursorPager[CursorType] {
	if c == nil {
		c = new(CursorPager[CursorType])
	}

	if limit == NoLimit {
		return c.WithUnlimited()
	}
	c.limit = NormalizeLimit(limit)

	return c
}

// WithCursor позволяет вручную задать курсор для пагинации.
func (c *CursorPager[CursorType]) WithCursor(cursor CursorType) *CursorPager[CursorType] {
	if c == nil {
		c = new(CursorPager[CursorType])
	}

	c.cursor = cursor

	return c
}

// WithSubstitutedSort вызывает WithSort, замещая при этом прежние сортировки.
func (c *CursorPager[CursorType]) WithSubstitutedSort(orderBy ...OrderBy) *CursorPager[CursorType] {
	if c == nil {
		c = new(CursorPager[CursorType])
	}

	c.sort = nil

	return c.WithSort(orderBy...)
}

// WithSort не затирает сортировки, а добавляет в массив ордеров.
// Сортировки применяются в порядке добавления, как если бы вызов был в формате:
//
//	OrderBy(o1).ThenBy(o2).ThenBy(o3)...
func (c *CursorPager[CursorType]) WithSort(orderBy ...OrderBy) *CursorPager[CursorType] {
	if c == nil {
		c = new(CursorPager[CursorType])
	}

	for _, o := range orderBy {
		idx := slices.IndexFunc(c.sort, func(processed OrderBy) bool {
			return processed.Column == o.Column
		})

		// Удалить прошлое вхождение (защита от дублирования).
		if idx != -1 {
			c.sort = slices.Delete(c.sort, idx, idx+1)
		}

		c.sort = append(c.sort, o)
	}

	return c
}

// Paginate - применяет пагинацию к датасету. Вернет ошибку, если пагинация не может быть применена.
func (c *CursorPager[CursorType]) Paginate(db *gorm.DB) (*gorm.DB, error) {
	if c == nil {
		c = new(CursorPager[CursorType])
	}

	err := c.validate()
	if err != nil {
		return nil, fmt.Errorf("cannot paginate: %w", err)
	}

	db = c.sort.Apply(db)
	db = c.cursor.Apply(db)

	// Применение лимита к датасету. В случае использования Lookahead -
	// заглядываем на один элемент вперед, чтобы узнать, есть ли элементы дальше.
	if c.limit != NoLimit {
		db = db.Limit(c.GetDatasetLimit())
	}

	return db, nil
}

// GetSort - возвращает список сортировок, которые быдут применены к датасету.
func (c *CursorPager[CursorType]) GetSort() Orderings {
	if c == nil {
		return nil
	}

	return c.sort
}

// IsUnlimited - возвращает true, если лимит равен NoLimit (неограниченное количество записей).
func (c *CursorPager[CursorType]) IsUnlimited() bool {
	if c == nil {
		return false
	}

	return c.limit == NoLimit
}

// IsLookahead - возвращает true, если применена пагинация с 'заглядыванием' на следующую страницу.
func (c *CursorPager[CursorType]) IsLookahead() bool {
	if c == nil {
		return false
	}

	return c.lookahead
}

// GetLimit - возвращает указанный в CursorPager лимит без изменений.
// Возвразаемое значение >= 0. Возврат NoLimit эквивалентен отсутствию лимита.
func (c *CursorPager[CursorType]) GetLimit() int {
	if c == nil {
		return 0
	}

	return c.limit
}

// GetCursor - возвращает указанный в CursorPager курсор без изменений.
func (c *CursorPager[CursorType]) GetCursor() CursorType {
	if c == nil {
		return lo.Empty[CursorType]()
	}

	return c.cursor
}

// GetDatasetLimit - возвращает указанный в CursorPager лимит с учетом Lookahead.
//   - при Lookahead = true - возвращаемое значение будет равно GetLimit() + 1;
//   - при Lookahead = false - возвращаемое значение будет равно GetLimit().
func (c *CursorPager[CursorType]) GetDatasetLimit() int {
	limit := c.GetLimit()
	isLookahead := c.IsLookahead()

	return lo.Ternary(isLookahead, limit+1, limit)
}

func (c *CursorPager[_]) validate() error {
	if c == nil {
		return fmt.Errorf("cursor pager is nil")
	}

	if c.limit == NoLimit && c.lookahead {
		return fmt.Errorf("cannot apply lookahead to unlimited paging")
	}

	err := c.sort.validate()
	if err != nil {
		return err
	}

	return c.cursor.validate(c.sort)
}

// IsLastPage - возвращает true, если результирующий сет является последней страницей в датасете.
//
// Для определения последней страницы датасета используется одно из двух условий:
//  1. В результирующем сете вернулось записей меньше, чем было указано в Limit.
//  2. Lookahead = true и вернулось записей меньше или равно Limit.
//
// Для таких случаев - необходимо вернуть результирующий сет без изменений с пустым токеном.
// Так мы обозначим конец датасета для клиента.
func IsLastPage[CursorType Cursor, T any](initialPager *CursorPager[CursorType], resultSet []T) bool {
	return len(resultSet) < initialPager.limit ||
		(initialPager.lookahead && len(resultSet) <= initialPager.limit)
}

// TrimResultSet - обрезает результирующий сет до состояния, возвращаемого клиенту.
//
// Если lookahead = true, то обрезаем последний элемент перед возвратом результата.
// Предположим, что resultSet = [a, b, c].
//
//   - Если lookahead используется, то resultSet станет равен [a, b].
//   - Если lookahead НЕ используется, то resultSet останется без изменений.
//
// Это необходимо для того, чтобы построить пагинацию на основе СТРОГОГО сравнения с последним элементом
// результирующего сета.
func TrimResultSet[CursorType Cursor, T any](initialPager *CursorPager[CursorType], resultSet []T) []T {
	if initialPager.lookahead {
		resultSet = resultSet[:len(resultSet)-1]
	}

	return resultSet
}
