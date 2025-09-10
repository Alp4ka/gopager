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

// PaginationResult результат с пагинацией.
type PaginationResult[T any, CursorType Cursor] struct {
	// Items элементы результата.
	Items []T
	// Total общее количество элементов.
	Total int64
	// AppliedLimit примененный лимит.
	AppliedLimit int
	// NextPageToken токен для следующей страницы.
	NextPageToken CursorType
}
