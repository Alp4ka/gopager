package gopager

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_DefaultCursor_validate(t *testing.T) {
	c := &DefaultCursor{elements: []CursorElement{{Column: "id", Value: 1, Operator: OperatorGT}}}
	okOrd := Orderings{{Column: "id", Direction: DirectionASC}}
	badCount := Orderings{{Column: "id", Direction: DirectionASC}, {Column: "name", Direction: DirectionASC}}
	badName := Orderings{{Column: "other", Direction: DirectionASC}}
	badOp := Orderings{{Column: "id", Direction: DirectionDESC}}

	tests := []struct {
		name string
		ord  Orderings
		ok   bool
	}{
		{"ok", okOrd, true},
		{"count mismatch", badCount, false},
		{"name mismatch", badName, false},
		{"operator mismatch", badOp, false},
	}
	for _, tt := range tests {
		if err := c.validate(tt.ord); (err == nil) != tt.ok {
			t.Errorf("%s: ok=%v err=%v", tt.name, tt.ok, err)
		}
	}
}

func Test_NextPageCursor(t *testing.T) {
	type item struct {
		ID        int
		CreatedAt string
	}

	getters := Getters[item]{
		"id":         func(i item) any { return i.ID },
		"created_at": func(i item) any { return i.CreatedAt },
	}

	ord := Orderings{{Column: "id", Direction: DirectionASC}, {Column: "created_at", Direction: DirectionASC}}

	tests := []struct {
		name           string
		pager          *CursorPager[*DefaultCursor]
		items          []item
		expectedLen    int
		expectedCursor bool
		expectedID     int
		expectedError  bool
	}{
		{
			name: "ordinary page without lookahead",
			pager: (&CursorPager[*DefaultCursor]{limit: 2, cursor: nil}).
				WithSubstitutedSort(ord...),
			items:          []item{{1, "2024-01-01T00:00:00Z"}, {2, "2024-01-02T00:00:00Z"}},
			expectedLen:    2,
			expectedCursor: true,
			expectedID:     2,
			expectedError:  false,
		},
		{
			name: "last page without lookahead",
			pager: (&CursorPager[*DefaultCursor]{limit: 2, cursor: nil}).
				WithSubstitutedSort(ord...),
			items:          []item{{3, "2024-01-03T00:00:00Z"}},
			expectedLen:    1,
			expectedCursor: false,
			expectedID:     0,
			expectedError:  false,
		},
		{
			name: "lookahead ordinary page",
			pager: (&CursorPager[*DefaultCursor]{limit: 2, cursor: nil}).
				WithSubstitutedSort(ord...).
				WithLookahead(),
			items: []item{{1, "2024-01-01T00:00:00Z"}, {
				2,
				"2024-01-02T00:00:00Z",
			}, {3, "2024-01-03T00:00:00Z"}},
			expectedLen:    2,
			expectedCursor: true,
			expectedID:     2,
			expectedError:  false,
		},
		{
			name: "last page with lookahead",
			pager: (&CursorPager[*DefaultCursor]{limit: 2, cursor: nil}).
				WithSubstitutedSort(ord...).
				WithLookahead(),
			items:          []item{{1, "2024-01-01T00:00:00Z"}},
			expectedLen:    1,
			expectedCursor: false,
			expectedID:     1,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, cur, err := NextPageCursor(tt.pager, tt.items, getters)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(res) != tt.expectedLen {
				t.Errorf("expected result length %d, got %d", tt.expectedLen, len(res))
			}

			if tt.expectedCursor {
				if cur == nil {
					t.Errorf("expected cursor but got nil")
				} else if len(cur.elements) != 2 {
					t.Errorf("expected cursor with 2 elements, got %d", len(cur.elements))
				} else if cur.elements[0].Column != "id" || cur.elements[0].Value != tt.expectedID {
					t.Errorf(
						"unexpected id value: expected column=id, value=%d, got %#v",
						tt.expectedID,
						cur.elements[0],
					)
				}
			} else {
				if cur != nil {
					t.Errorf("expected nil cursor but got %#v", cur)
				}
			}
		})
	}
}

func Test_DefaultCursor_Stringify_Decode_And_Compare(t *testing.T) {
	c := &DefaultCursor{elements: []CursorElement{{Column: "id", Value: 1, Operator: OperatorGT}}}
	enc := c.String()

	c2, err := DecodeCursor(enc)
	if err != nil {
		t.Fatalf("roundtrip failed: %v", err)
	}

	require.Equal(t, c2.String(), c.String())
}
