package gopager

// Package gopager provides cursor-based pagination primitives for GORM.
//
// Overview
//
// gopager implements two cursor strategies:
//   - DefaultCursor: keyset pagination using comparison operators against the
//     last element of the previous page. This scales well on large datasets and
//     requires a deterministic ordering with at least one unique column.
//   - PseudoCursor: a compatibility layer over LIMIT/OFFSET when true cursors
//     are not possible.
//
// Key concepts
//   - CursorPager: orchestrates pagination, lookahead, sorting and applying
//     cursors to GORM queries.
//   - Orderings: defines multi-column ordering with explicit directions.
//   - Getters: maps model fields to values for building the next page cursor.
//
// See README for examples and usage details.
