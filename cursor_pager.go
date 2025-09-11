package gopager

import (
	"fmt"
	"slices"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

// RawCursorPager is intended for API payloads. For proper code generation, inline it:
//
//	type MyFilter struct {
//	    Paging RawCursorPager `json:",inline"`
//	}
type RawCursorPager struct {
	// Limit - maximum number of records to return in the response.
	Limit int `json:"limit"`
	// StartToken - base64-encoded cursor token obtained via Cursor.String().
	// If empty, the first page with Limit records is returned.
	StartToken string `json:"startToken"`
}

// Decode converts RawCursorPager into *CursorPager[*DefaultCursor], normalizing
// Limit and validating StartToken. Returns *CursorPager[*DefaultCursor] with
// WithSort applied.
func (p RawCursorPager) Decode(orderBy ...OrderBy) (*CursorPager[*DefaultCursor], error) {
	return DecodeCursorPager(p.Limit, p.StartToken, orderBy...)
}

// DecodePseudo converts RawCursorPager into *CursorPager[*PseudoCursor], normalizing
// Limit and validating StartToken. Returns *CursorPager[*PseudoCursor] with
// WithSort applied.
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

// DecodeCursorPager decodes a cursor token into *CursorPager.
//
// Usage guide: https://doc.office.lan/spaces/MBCSHCH/pages/417057947
func DecodeCursorPager(limit int, rawStartToken string, orderBy ...OrderBy) (*CursorPager[*DefaultCursor], error) {
	cursor, err := DecodeCursor(rawStartToken)
	if err != nil {
		return nil, err
	}

	return (&CursorPager[*DefaultCursor]{
		cursor: cursor,
	}).WithSubstitutedSort(orderBy...).WithLimit(limit), nil
}

// DecodePseudoCursorPager decodes a pseudo-cursor token into *CursorPager.
//
// Usage guide: https://doc.office.lan/spaces/MBCSHCH/pages/417057947
func DecodePseudoCursorPager(limit int, rawStartToken string, orderBy ...OrderBy) (*CursorPager[*PseudoCursor], error) {
	cursor, err := DecodePseudoCursor(rawStartToken)
	if err != nil {
		return nil, err
	}

	return (&CursorPager[*PseudoCursor]{
		cursor: cursor,
	}).WithSubstitutedSort(orderBy...).WithLimit(limit), nil
}

// WithLookahead enables lookahead pagination, which checks the next page to
// determine whether the current page is the last.
//
// IMPORTANT:
// Cannot be used together with WithUnlimited() or WithLimit(NoLimit).
func (c *CursorPager[CursorType]) WithLookahead() *CursorPager[CursorType] {
	if c == nil {
		c = new(CursorPager[CursorType])
	}

	c.lookahead = true

	return c
}

// WithUnlimited allows returning all records without a limit.
//
// IMPORTANT:
// Cannot be used together with WithLookahead.
func (c *CursorPager[CursorType]) WithUnlimited() *CursorPager[CursorType] {
	if c == nil {
		c = new(CursorPager[CursorType])
	}

	c.limit = NoLimit

	return c
}

// WithLimit sets the maximum number of returned records.
//
// IMPORTANT:
//   - NoLimit cannot be used together with WithLookahead.
//   - If the limit is not NoLimit, NormalizeLimit will be applied.
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

// WithCursor sets the cursor explicitly.
func (c *CursorPager[CursorType]) WithCursor(cursor CursorType) *CursorPager[CursorType] {
	if c == nil {
		c = new(CursorPager[CursorType])
	}

	c.cursor = cursor

	return c
}

// WithSubstitutedSort resets previous orderings and applies the provided ones.
func (c *CursorPager[CursorType]) WithSubstitutedSort(orderBy ...OrderBy) *CursorPager[CursorType] {
	if c == nil {
		c = new(CursorPager[CursorType])
	}

	c.sort = nil

	return c.WithSort(orderBy...)
}

// WithSort appends sort orderings without overwriting existing ones.
// Order is preserved as if calling:
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

		// Remove previous occurrence (avoid duplication).
		if idx != -1 {
			c.sort = slices.Delete(c.sort, idx, idx+1)
		}

		c.sort = append(c.sort, o)
	}

	return c
}

// Paginate applies pagination to the dataset. Returns an error if pagination
// cannot be applied.
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

	// Apply limit to the dataset. When lookahead is enabled, fetch one extra
	// record to determine if there is a next page.
	if c.limit != NoLimit {
		db = db.Limit(c.GetDatasetLimit())
	}

	return db, nil
}

// GetSort returns orderings that will be applied to the dataset.
func (c *CursorPager[CursorType]) GetSort() Orderings {
	if c == nil {
		return nil
	}

	return c.sort
}

// IsUnlimited returns true if the limit equals NoLimit (unbounded number of records).
func (c *CursorPager[CursorType]) IsUnlimited() bool {
	if c == nil {
		return false
	}

	return c.limit == NoLimit
}

// IsLookahead returns true if lookahead pagination is enabled.
func (c *CursorPager[CursorType]) IsLookahead() bool {
	if c == nil {
		return false
	}

	return c.lookahead
}

// GetLimit returns the limit as it is stored in CursorPager.
// The return value is >= 0. Returning NoLimit is equivalent to no limit.
func (c *CursorPager[CursorType]) GetLimit() int {
	if c == nil {
		return 0
	}

	return c.limit
}

// GetCursor returns the cursor stored in CursorPager as-is.
func (c *CursorPager[CursorType]) GetCursor() CursorType {
	if c == nil {
		return lo.Empty[CursorType]()
	}

	return c.cursor
}

// GetDatasetLimit returns the limit adjusted for lookahead:
//   - if Lookahead = true → GetLimit() + 1
//   - if Lookahead = false → GetLimit()
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

// IsLastPage returns true if the result set is the last page in the dataset.
//
// The last page is determined by one of two conditions:
//  1. The number of returned records is less than Limit.
//  2. Lookahead = true and the number of returned records is less than or equal to Limit.
//
// In these cases, return the result set unchanged with an empty token to
// signal the end of the dataset to the client.
func IsLastPage[CursorType Cursor, T any](initialPager *CursorPager[CursorType], resultSet []T) bool {
	return len(resultSet) < initialPager.limit ||
		(initialPager.lookahead && len(resultSet) <= initialPager.limit)
}

// TrimResultSet trims the result set to what should be returned to the client.
//
// If lookahead = true, drop the last element before returning. Suppose
// resultSet = [a, b, c].
//
//   - With lookahead → resultSet becomes [a, b].
//   - Without lookahead → resultSet remains unchanged.
//
// This enables building pagination based on a STRICT comparison with the
// last element of the result set.
func TrimResultSet[CursorType Cursor, T any](initialPager *CursorPager[CursorType], resultSet []T) []T {
	if initialPager.lookahead {
		resultSet = resultSet[:len(resultSet)-1]
	}

	return resultSet
}
