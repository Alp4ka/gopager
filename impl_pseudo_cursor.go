package gopager

import (
	"fmt"
	"strconv"

	"gorm.io/gorm"
)

// PseudoCursor is used when an API requires cursor-based pagination but only
// LIMIT/OFFSET pagination is available.
//
// It implements Cursor and generates a token based on the last offset within
// the dataset.
type PseudoCursor struct {
	offset int
}

func NewPseudoCursor(offset int) *PseudoCursor {
	return &PseudoCursor{
		offset: offset,
	}
}

// DecodePseudoCursor attempts to parse a base64-encoded string into *PseudoCursor.
func DecodePseudoCursor(b64String string) (*PseudoCursor, error) {
	if len(b64String) == 0 {
		return nil, nil
	}

	offsetBytes, err := _encoder.DecodeString(b64String)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 encoded pseudo cursor: %w", err)
	}

	offset, err := strconv.Atoi(string(offsetBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode pseudo cursor offset value: %w", err)
	}

	return &PseudoCursor{
		offset: offset,
	}, nil
}

// ToSQL - implements Cursor. Returns the string form of the numeric offset value.
//
// Usage:
//
//	query := fmt.Sprintf("SELECT * FROM table OFFSET %s", p.ToSQL())
func (p *PseudoCursor) ToSQL() string {
	return strconv.Itoa(p.offset)
}

// String - implements fmt.Stringer.
func (p *PseudoCursor) String() string {
	if p == nil || p.offset == 0 {
		return ""
	}

	return _encoder.EncodeToString([]byte(strconv.Itoa(p.offset)))
}

// IsEmpty - implements Cursor.
func (p *PseudoCursor) IsEmpty() bool {
	return p == nil || p.offset == 0
}

// Apply - implements Cursor. Applies the offset to a gorm query.
func (p *PseudoCursor) Apply(db *gorm.DB) *gorm.DB {
	return db.Offset(p.GetOffset())
}

// GetOffset returns the numeric offset value.
func (p *PseudoCursor) GetOffset() int {
	if p != nil {
		return p.offset
	}

	return 0
}

// WithOffset sets the numeric offset value and returns the cursor.
func (p *PseudoCursor) WithOffset(offset int) *PseudoCursor {
	if p == nil {
		p = new(PseudoCursor)
	}

	p.offset = offset

	return p
}

// validate - implements Cursor.
func (p *PseudoCursor) validate(_ Orderings) error {
	return nil
}

var (
	_ Cursor       = (*PseudoCursor)(nil)
	_ fmt.Stringer = (*PseudoCursor)(nil)
)

// NextPagePseudoCursor builds a pseudo-cursor for the next page of the dataset.
func NextPagePseudoCursor[T any](
	initialPager *CursorPager[*PseudoCursor],
	resultSet []T,
) ([]T, *PseudoCursor, error) {
	err := initialPager.validate()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot build next page pseudo cursor: %w", err)
	}

	if IsLastPage(initialPager, resultSet) {
		return resultSet, nil, nil
	}
	resultSet = TrimResultSet(initialPager, resultSet)

	return resultSet,
		&PseudoCursor{
			offset: initialPager.cursor.GetOffset() + len(resultSet),
		},
		nil
}
