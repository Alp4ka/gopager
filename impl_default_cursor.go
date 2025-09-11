package gopager

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

// DefaultCursor represents a pagination token that defines the starting
// position for the requested page. An empty token means the beginning of the dataset.
//
// IMPORTANT:
// The token MUST always include a condition on a unique column!
//
// The token consists of a set of conditions of the form:
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

// DecodeCursor attempts to parse a base64-encoded string into *DefaultCursor.
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

// GetElements returns token elements. Cursor elements are a compressed set
// of filter conditions.
//
// IMPORTANT:
// These filter conditions must NOT be applied directly to data, as they are
// not complete. During pagination they are inflated into a full set of
// filtering conditions.
func (c *DefaultCursor) GetElements() []CursorElement {
	if c == nil {
		return nil
	}

	return c.elements
}

// WithElements explicitly sets the token elements.
func (c *DefaultCursor) WithElements(elements []CursorElement) *DefaultCursor {
	if c == nil {
		c = new(DefaultCursor)
	}

	c.elements = elements

	return c
}

// Apply - implements Cursor. Applies filter-based offset to the gorm query.
func (c *DefaultCursor) Apply(db *gorm.DB) *gorm.DB {
	exp := c.toDNF().toGORMExpression()
	if exp == nil {
		return db
	}

	return db.Clauses(exp)
}

// ToSQL - implements Cursor. Returns the SQL expression representing the filter.
//
// Usage:
//
//	query := fmt.Sprintf("SELECT * FROM table WHERE %s", p.ToSQL())
func (c *DefaultCursor) ToSQL() (string, []driver.Value) {
	if c.IsEmpty() {
		return "TRUE", nil
	}

	return c.toDNF().toSQLClause()
}

// toDNF converts DefaultCursor to tDNF.
//
// IMPORTANT:
// The token MUST always include a condition on a unique column!
//
// The token consists of a set of conditions of the form:
//
//	[(C1, O1, V1), (C2, O2, V2)... (Cn, On, Vn)]
//
// Applying sequential Inflate transformations to this set, we get the filter:
//
//	(C1 O1 V1) or (C1 = V1 and C2 O2 V2)
//
// In this form the token represents a DNF sufficient for filtering. This allows
// us to unambiguously determine the position from which to continue fetching data.
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

	// Do not allow mismatch between number of cursor columns and ordering list.
	if len(c.elements) != len(orderings) && len(c.elements) != 0 {
		return fmt.Errorf("cursor column number mismatch")
	}

	// Validate consistency of ordering and filters. Empty element list is allowed.
	for i := range c.elements {
		cond := c.elements[i]
		orderBy := orderings[i]

		// Verify column names match.
		if cond.Column != orderBy.Column {
			return fmt.Errorf("unexpected cursor column '%s'", cond.Column)
		}

		// Verify operator is acceptable and corresponds to ordering.
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

// Getters is a map of getters for a type. Specify the columns used for pagination.
// Example:
//
//	pager.Getters[models.PlayerPushTarget]{
//		"id":          func(last models.PlayerPushTarget) any { return last.ID },
//		"deposit_sum": func(last models.PlayerPushTarget) any { return last.DepositSum },
//	}
type Getters[T any] map[string]func(T) any

// NextPageCursor builds a cursor for the next page of the dataset.
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

// CursorElement represents a triplet (c v o), where:
//
//   - c: an object's field (column)
//   - v: the value compared against the field
//   - o: the operator applied to the pair (c, v)
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
