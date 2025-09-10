package gopager

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_PseudoCursor_Decode(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOffset int
		expectedEmpty  bool
	}{
		{
			"zero empty",
			"",
			0,
			true,
		},
		{
			"zero encoded",
			base64.RawURLEncoding.EncodeToString([]byte("0")),
			0,
			true,
		},
		{
			"non-zero encodes",
			base64.RawURLEncoding.EncodeToString([]byte("15")),
			15,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc, err := DecodePseudoCursor(tt.input)
			if err != nil {
				t.Fatalf("decode failed: %v pc=%#v", err, pc)
			}

			if e := pc.IsEmpty(); e != tt.expectedEmpty {
				t.Errorf("%s: IsEmpty=%v want %v", tt.name, e, tt.expectedEmpty)
			}
			if off := pc.GetOffset(); off != tt.expectedOffset {
				t.Errorf("%s: GetOffset=%d want %d", tt.name, off, tt.expectedOffset)
			}
		})
	}
}

func Test_NextPagePseudoCursor(t *testing.T) {
	type item struct{ ID int }

	tests := []struct {
		name        string
		description string
		pager       *CursorPager[*PseudoCursor]
		input       []item
		expectedRes []item
		expectedCur *PseudoCursor
		expectError bool
	}{
		{
			name:        "last page without lookahead",
			description: "Количество элементов в результирующем сете строго меньше лимита. При lookahead = false, это говорит о конце датасета.",
			pager: func() *CursorPager[*PseudoCursor] {
				p := &CursorPager[*PseudoCursor]{limit: 3, cursor: &PseudoCursor{offset: 0}}
				p.WithSort(OrderBy{
					Column:    "id",
					Direction: DirectionASC,
				})
				return p
			}(),
			input:       []item{{1}, {2}},
			expectedRes: []item{{1}, {2}},
			expectedCur: nil,
			expectError: false,
		},
		{
			name:        "ordinary page without lookahead",
			description: "Количество элементов в результирующем сете строго равно лимиту. При lookahead = false, это говорит: 1. Либо о том, что это НЕ конец датасета. 2. Либо о том, что в следующей странице будет пустой набор элементов.",
			pager: func() *CursorPager[*PseudoCursor] {
				p := &CursorPager[*PseudoCursor]{limit: 2, cursor: &PseudoCursor{offset: 4}}
				p.WithSort(OrderBy{
					Column:    "id",
					Direction: DirectionASC,
				})
				return p
			}(),
			input:       []item{{1}, {2}},
			expectedRes: []item{{1}, {2}},
			expectedCur: &PseudoCursor{offset: 6},
			expectError: false,
		},
		{
			name:        "last page with lookahead",
			description: "Количество элементов в результирующем сете строго равно лимиту. При lookahead = true, это говорит о конце датасета. Приэтом функция должна вернуть полный набор данных, не обрезая последний элемент.",
			pager: func() *CursorPager[*PseudoCursor] {
				p := (&CursorPager[*PseudoCursor]{limit: 2, cursor: &PseudoCursor{offset: 2}}).WithLookahead()
				p.WithSort(OrderBy{
					Column:    "id",
					Direction: DirectionASC,
				})
				return p
			}(),
			input:       []item{{1}, {2}},
			expectedRes: []item{{1}, {2}},
			expectedCur: nil,
			expectError: false,
		},
		{
			name:        "ordinary page with lookahead",
			description: "Количество элементов в результирующем сете строго больше лимита. При lookahead = true, это говорит о налчии следующей страницы. Приэтом функция должна обрезать последний элемент, так как он выполняет ТОЛЬКО задачу определения конца датасета.",
			pager: func() *CursorPager[*PseudoCursor] {
				p := (&CursorPager[*PseudoCursor]{limit: 2, cursor: &PseudoCursor{offset: 2}}).WithLookahead()
				p.WithSort(OrderBy{
					Column:    "id",
					Direction: DirectionASC,
				})
				return p
			}(),
			input:       []item{{1}, {2}, {3}},
			expectedRes: []item{{1}, {2}},
			expectedCur: &PseudoCursor{offset: 4},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Выводим описание теста для лучшего понимания
			t.Logf("Test description: %s", tt.description)

			res, cur, err := NextPagePseudoCursor(tt.pager, tt.input)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedRes, res)

			if tt.expectedCur == nil {
				require.Nil(t, cur, "expected nil cursor")
			} else {
				require.NotNil(t, cur, "expected non-nil cursor")
				require.Equal(t, tt.expectedCur.offset, cur.offset, "unexpected cursor offset")
			}
		})
	}
}
