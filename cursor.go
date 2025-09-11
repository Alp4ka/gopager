package gopager

import (
	"encoding/base64"

	"gorm.io/gorm"
)

var _encoder = base64.RawURLEncoding

type Cursor interface {
	String() string
	IsEmpty() bool
	Apply(*gorm.DB) *gorm.DB
	validate(orderings Orderings) error
}

// PaginationResult is a generic paginated result container.
type PaginationResult[T any, CursorType Cursor] struct {
	// Items result elements.
	Items []T
	// Total number of elements.
	Total int64
	// AppliedLimit effective limit used for the query.
	AppliedLimit int
	// NextPageToken token for the next page.
	NextPageToken CursorType
}
