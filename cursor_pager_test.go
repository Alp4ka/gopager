package gopager

import (
	"database/sql/driver"
	"fmt"
	"gorm.io/gorm"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CursorPager_WithMethods_And_SortDedup(t *testing.T) {
	p := (*CursorPager[*DefaultCursor])(nil)
	p = p.WithLimit(5).
		WithLookahead().
		WithUnlimited().
		WithSubstitutedSort(
			OrderBy{Column: "id", Direction: DirectionASC},
		).
		WithSort(
			OrderBy{Column: "id", Direction: DirectionDESC},
			OrderBy{Column: "created_at", Direction: DirectionASC},
		)

	if !p.lookahead {
		t.Fatalf("expected lookahead")
	}
	if p.limit != NoLimit {
		t.Fatalf("expected NoLimit after WithUnlimited")
	}
	require.Equal(
		t,
		Orderings(
			[]OrderBy{
				{Column: "id", Direction: DirectionDESC},
				{Column: "created_at", Direction: DirectionASC},
			},
		),
		p.sort,
	)
}

func Test_CursorPager_validate(t *testing.T) {
	tests := []struct {
		name    string
		pager   *CursorPager[*DefaultCursor]
		wantErr bool
	}{
		{
			name: "standard case, ok",
			pager: &CursorPager[*DefaultCursor]{
				lookahead: true,
				limit:     10,
				cursor: &DefaultCursor{
					elements: []CursorElement{{Column: "id", Value: 1, Operator: OperatorGT}},
				},
				sort: Orderings([]OrderBy{{
					Column:    "id",
					Direction: DirectionASC,
				}}),
			},
			wantErr: false,
		},
		{
			name: "lookahead with no limit is forbidden",
			pager: &CursorPager[*DefaultCursor]{
				lookahead: true,
				limit:     NoLimit,
				cursor: &DefaultCursor{
					elements: []CursorElement{{Column: "id", Value: 1, Operator: OperatorGT}},
				},
				sort: Orderings([]OrderBy{{
					Column:    "id",
					Direction: DirectionASC,
				}}),
			},
			wantErr: true,
		},
		{
			name: "sort list should contain the same elements as cursor",
			pager: &CursorPager[*DefaultCursor]{
				lookahead: true,
				limit:     10,
				cursor: &DefaultCursor{
					elements: []CursorElement{{Column: "id", Value: 1, Operator: OperatorGT}},
				},
				sort: Orderings([]OrderBy{{
					Column:    "name",
					Direction: DirectionASC,
				}}),
			},
			wantErr: true,
		},
		{
			name: "sort list should contain all elements from cursor",
			pager: &CursorPager[*DefaultCursor]{
				lookahead: true,
				limit:     10,
				cursor: &DefaultCursor{
					elements: []CursorElement{
						{Column: "id", Value: 1, Operator: OperatorGT},
						{Column: "surname", Value: "lol", Operator: OperatorGT},
					},
				},
				sort: Orderings([]OrderBy{
					{
						Column:    "id",
						Direction: DirectionASC,
					},
					{
						Column:    "name",
						Direction: DirectionASC,
					},
				}),
			},
			wantErr: true,
		},
		{
			name: "unsuitable sort direction for operator",
			pager: &CursorPager[*DefaultCursor]{
				lookahead: true,
				limit:     10,
				cursor: &DefaultCursor{
					elements: []CursorElement{
						{Column: "id", Value: 1, Operator: OperatorLT},
					},
				},
				sort: Orderings([]OrderBy{
					{
						Column:    "id",
						Direction: DirectionASC,
					},
				}),
			},
			wantErr: true,
		},
		{
			name:    "nil pager is invalid",
			pager:   (*CursorPager[*DefaultCursor])(nil),
			wantErr: true,
		},
		{
			name: "pager with no sort is invalid",
			pager: &CursorPager[*DefaultCursor]{
				lookahead: true,
				limit:     10,
				cursor: &DefaultCursor{
					elements: []CursorElement{
						{Column: "id", Value: 1, Operator: OperatorLT},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotErr := tt.pager.validate(); (gotErr != nil) != tt.wantErr {
				t.Errorf("%s: got error = %T, want error = %T", tt.name, gotErr, tt.wantErr)
			}
		})
	}
}

func Test_CursorPager_Paginate_PseudoCursor(t *testing.T) {
	sqlMockFnList := []func() (string, *gorm.DB, sqlmock.Sqlmock, error){
		newGORMMySQLMock,
		newGORMPostgresMock,
	}

	type tUser struct {
		ID   uint
		Name string
	}

	tests := []struct {
		name          string
		limit         int
		cursor        *PseudoCursor
		lookahead     bool
		expectedQuery string
		expectedArgs  []driver.Value
		expectedRows  *sqlmock.Rows
	}{
		{
			name:          "basic pagination with limit and offset",
			limit:         3,
			cursor:        &PseudoCursor{offset: 5},
			lookahead:     false,
			expectedQuery: "^SELECT \\* FROM [`'\"]users[`'\"] WHERE name = ['\"]lol['\"] ORDER BY id ASC LIMIT 3 OFFSET 5$",
			expectedArgs:  nil,
			expectedRows:  sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "John Doe"),
		},
		{
			name:          "pagination with lookahead",
			limit:         3,
			cursor:        &PseudoCursor{offset: 5},
			lookahead:     true,
			expectedQuery: "^SELECT \\* FROM [`'\"]users[`'\"] WHERE name = [`'\"]lol[`'\"] ORDER BY id ASC LIMIT 4 OFFSET 5$",
			expectedArgs:  nil,
			expectedRows:  sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "John Doe"),
		},
		{
			name:          "pagination without cursor (offset 0)",
			limit:         5,
			cursor:        &PseudoCursor{offset: 0},
			lookahead:     false,
			expectedQuery: "^SELECT \\* FROM [`'\"]users[`'\"] WHERE name = [`'\"]lol[`'\"] ORDER BY id ASC LIMIT 5$",
			expectedArgs:  nil,
			expectedRows:  sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "John Doe"),
		},
		{
			name:          "pagination with nil cursor",
			limit:         10,
			cursor:        nil,
			lookahead:     false,
			expectedQuery: "^SELECT \\* FROM [`'\"]users[`'\"] WHERE name = [`'\"]lol[`'\"] ORDER BY id ASC LIMIT 10$",
			expectedArgs:  nil,
			expectedRows:  sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "John Doe"),
		},
	}

	for _, sqlMockFn := range sqlMockFnList {
		for _, tt := range tests {
			dialect, db, dbMock, err := sqlMockFn()
			t.Run(fmt.Sprintf("%s %s", dialect, tt.name), func(t *testing.T) {
				if err != nil {
					t.Fatalf("gorm open: %v", err)
				}

				expectation := dbMock.ExpectQuery(tt.expectedQuery)
				if len(tt.expectedArgs) > 0 {
					expectation = expectation.WithArgs(tt.expectedArgs...)
				}
				expectation.WillReturnRows(tt.expectedRows)

				p := new(CursorPager[*PseudoCursor]).
					WithLimit(tt.limit).
					WithCursor(tt.cursor).
					WithSubstitutedSort(
						OrderBy{Column: "id", Direction: DirectionASC},
					)

				if tt.lookahead {
					p = p.WithLookahead()
				}

				paged, err := p.Paginate(db.Select("*").Table("users").Where("name = 'lol'"))
				if err != nil {
					t.Fatalf("paginate: %v", err)
				}

				err = paged.Find(&[]tUser{}).Error
				if err != nil {
					t.Fatalf("find: %v", err)
				}

				assert.NoError(t, dbMock.ExpectationsWereMet())
			})
		}
	}
}

func Test_CursorPager_Paginate_DefaultCursor(t *testing.T) {
	sqlMockFnList := []func() (string, *gorm.DB, sqlmock.Sqlmock, error){
		newGORMMySQLMock,
		newGORMPostgresMock,
	}

	type tUser struct {
		ID   uint
		Name string
	}

	tests := []struct {
		name          string
		limit         int
		cursor        *DefaultCursor
		orderings     Orderings
		lookahead     bool
		expectedQuery string
		expectedArgs  []driver.Value
		expectedRows  *sqlmock.Rows
	}{
		{
			name:          "basic pagination with cursor",
			limit:         3,
			cursor:        &DefaultCursor{elements: []CursorElement{{Column: "id", Value: 5, Operator: OperatorGT}}},
			orderings:     Orderings([]OrderBy{{Column: "id", Direction: DirectionASC}}),
			lookahead:     false,
			expectedQuery: "^SELECT \\* FROM [`'\"]users[`'\"] WHERE name = [`'\"]lol[`'\"] AND id > (?:\\$\\d|\\?) ORDER BY id ASC LIMIT 3$",
			expectedArgs:  []driver.Value{5},
			expectedRows:  sqlmock.NewRows([]string{"id", "name"}).AddRow(6, "John Doe"),
		},
		{
			name:          "pagination with lookahead",
			limit:         3,
			cursor:        &DefaultCursor{elements: []CursorElement{{Column: "id", Value: 5, Operator: OperatorGT}}},
			orderings:     Orderings([]OrderBy{{Column: "id", Direction: DirectionASC}}),
			lookahead:     true,
			expectedQuery: "^SELECT \\* FROM [`'\"]users[`'\"] WHERE name = [`'\"]lol[`'\"] AND id > (?:\\$\\d|\\?) ORDER BY id ASC LIMIT 4$",
			expectedArgs:  []driver.Value{5},
			expectedRows:  sqlmock.NewRows([]string{"id", "name"}).AddRow(6, "John Doe"),
		},
		{
			name:  "pagination with multiple cursor elements",
			limit: 5,
			cursor: &DefaultCursor{
				elements: []CursorElement{
					{Column: "id", Value: 10, Operator: OperatorGT},
					{Column: "created_at", Value: "2023-01-01", Operator: OperatorGT},
				},
			},
			orderings: Orderings([]OrderBy{
				{Column: "id", Direction: DirectionASC},
				{Column: "created_at", Direction: DirectionASC},
			}),
			lookahead:     false,
			expectedQuery: "^SELECT \\* FROM [`'\"]users[`'\"] WHERE name = [`'\"]lol[`'\"] AND \\(id > (?:\\$\\d|\\?) OR \\(id = (?:\\$\\d|\\?) AND created_at > (?:\\$\\d|\\?)\\)\\) ORDER BY id ASC, created_at ASC LIMIT 5$",
			expectedArgs:  []driver.Value{10, 10, "2023-01-01"},
			expectedRows:  sqlmock.NewRows([]string{"id", "name"}).AddRow(11, "Jane Doe"),
		},
		{
			name:   "pagination with nil cursor",
			limit:  10,
			cursor: nil,
			orderings: Orderings([]OrderBy{
				{Column: "id", Direction: DirectionASC},
			}),
			lookahead:     false,
			expectedQuery: "^SELECT \\* FROM [`'\"]users[`'\"] WHERE name = [`'\"]lol[`'\"] ORDER BY id ASC LIMIT 10$",
			expectedArgs:  nil,
			expectedRows:  sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "John Doe"),
		},
		{
			name:   "pagination with empty cursor",
			limit:  10,
			cursor: &DefaultCursor{elements: []CursorElement{}},
			orderings: Orderings([]OrderBy{
				{Column: "id", Direction: DirectionASC},
			}),
			lookahead:     false,
			expectedQuery: "^SELECT \\* FROM [`'\"]users[`'\"] WHERE name = [`'\"]lol[`'\"] ORDER BY id ASC LIMIT 10$",
			expectedArgs:  nil,
			expectedRows:  sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "John Doe"),
		},
		{
			name:   "pagination with DESC ordering",
			limit:  3,
			cursor: &DefaultCursor{elements: []CursorElement{{Column: "id", Value: 5, Operator: OperatorLT}}},
			orderings: Orderings([]OrderBy{
				{Column: "id", Direction: DirectionDESC},
			}),
			lookahead:     false,
			expectedQuery: "^SELECT \\* FROM [`'\"]users[`'\"] WHERE name = [`'\"]lol[`'\"] AND id < (?:\\$\\d|\\?) ORDER BY id DESC LIMIT 3$",
			expectedArgs:  []driver.Value{5},
			expectedRows:  sqlmock.NewRows([]string{"id", "name"}).AddRow(4, "Jane Doe"),
		},
	}

	for _, sqlMockFn := range sqlMockFnList {
		for _, tt := range tests {
			dialect, db, dbMock, err := sqlMockFn()
			t.Run(fmt.Sprintf("%s %s", dialect, tt.name), func(t *testing.T) {
				if err != nil {
					t.Fatalf("gorm open: %v", err)
				}

				expectation := dbMock.ExpectQuery(tt.expectedQuery)
				if len(tt.expectedArgs) > 0 {
					expectation = expectation.WithArgs(tt.expectedArgs...)
				}
				expectation.WillReturnRows(tt.expectedRows)

				p := new(CursorPager[*DefaultCursor]).
					WithLimit(tt.limit).
					WithCursor(tt.cursor).
					WithSubstitutedSort(tt.orderings...)

				if tt.lookahead {
					p = p.WithLookahead()
				}

				paged, err := p.Paginate(db.Select("*").Table("users").Where("name = 'lol'"))
				if err != nil {
					t.Fatalf("paginate: %v", err)
				}

				err = paged.Find(&[]tUser{}).Error
				if err != nil {
					t.Fatalf("find: %v", err)
				}

				assert.NoError(t, dbMock.ExpectationsWereMet())
			})
		}
	}
}
