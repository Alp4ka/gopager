# GoPager - Cursor-Based Pagination for Go
A cursor-based pagination library for Go applications using GORM. 
GoPager provides efficient pagination for large datasets without the performance issues of traditional offset-based pagination.

[![tag](https://img.shields.io/github/tag/samber/lo.svg)](https://github.com/samber/lo/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.18-%23007d9c)
[![GoDoc](https://godoc.org/github.com/samber/lo?status.svg)](https://pkg.go.dev/github.com/samber/lo)
![Build Status](https://github.com/samber/lo/actions/workflows/test.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/samber/lo)](https://goreportcard.com/report/github.com/samber/lo)
[![Coverage](https://img.shields.io/codecov/c/github/samber/lo)](https://codecov.io/gh/samber/lo)
[![Contributors](https://img.shields.io/github/contributors/samber/lo)](https://github.com/samber/lo/graphs/contributors)
[![License](https://img.shields.io/github/license/samber/lo)](./LICENSE)

## Features
- Efficient pagination for large datasets;
- DefaultCursor for complex filtering and PseudoCursor for simple offset-based pagination;
- Seamless integration with GORM ORM;
- Lookahead pagination detects if there are more pages available further from the current one;
- Support for multiple column sorting with custom directions;
- Base64 encoded cursors.

## Installation
```bash
go get github.com/Alp4ka/gopager@latest
```

## Quick Start
### Basic Usage with DefaultCursor
```go
package main

import (
    "fmt"
    "log"
    
    "github.com/Alp4ka/gopager"
    "gorm.io/gorm"
)

type User struct {
    ID        uint   `gorm:"primaryKey"`
    Name      string
    Email     string
    CreatedAt time.Time
}

func main() {
    // Create a new cursor pager
    pager := gopager.NewCursorPager[*gopager.DefaultCursor]().
        WithLimit(10).
        WithSort(
            gopager.OrderBy{Column: "id", Direction: gopager.DirectionASC}, // IMPORTANT: You must include unique column at least once.
            gopager.OrderBy{Column: "created_at", Direction: gopager.DirectionDESC},
        )
    
    // Apply pagination to GORM query
    var users []User
    db, err := pager.Paginate(db.Model(&User{}))
    if err != nil {
        log.Fatal(err)
    }
    
    // Execute the query
    result := db.Find(&users)
    if result.Error != nil {
        log.Fatal(result.Error)
    }
    
    // Generate next page cursor
    getters := gopager.Getters[User]{
        "id":         func(u User) any { return u.ID },
        "created_at": func(u User) any { return u.CreatedAt },
    }
    
    trimmedUsers, nextCursor, err := gopager.NextPageCursor(pager, users, getters)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d users\n", len(trimmedUsers))
    if nextCursor != nil {
        fmt.Printf("Next page token: %s\n", nextCursor.String())
    }
}
```

### Using PseudoCursor for Simple Pagination
```go
package main

import (
    "fmt"
    "log"
    
    "github.com/Alp4ka/gopager"
    "gorm.io/gorm"
)

func main() {
    // Create a pseudo cursor pager (uses OFFSET/LIMIT)
    pager := gopager.NewCursorPager[*gopager.PseudoCursor]().
        WithLimit(10).
        WithSort(
            gopager.OrderBy{Column: "id", Direction: gopager.DirectionASC},
        )
    
    // Apply pagination
    var users []User
    db, err := pager.Paginate(db.Model(&User{}))
    if err != nil {
        log.Fatal(err)
    }
    
    result := db.Find(&users)
    if result.Error != nil {
        log.Fatal(result.Error)
    }
    
    // Generate next page pseudo cursor
    trimmedUsers, nextCursor, err := gopager.NextPagePseudoCursor(pager, users)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d users\n", len(trimmedUsers))
    if nextCursor != nil {
        fmt.Printf("Next page token: %s\n", nextCursor.String())
    }
}
```

### HTTP API Integration
```go
package main

import (
    "encoding/json"
    "net/http"
    
    "github.com/Alp4ka/gopager"
    "gorm.io/gorm"
)

type PaginationRequest struct {
    gopager.RawCursorPager `json:",inline"`
}

type PaginationResponse[T any] struct {
    Items         []T                        `json:"items"`
    NextPageToken *gopager.DefaultCursor     `json:"nextPageToken,omitempty"`
    HasMore       bool                       `json:"hasMore"`
}

func GetUsersHandler(db *gorm.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req PaginationRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid request", http.StatusBadRequest)
            return
        }
        
        // Decode the cursor pager
        pager, err := req.Decode(
            gopager.OrderBy{Column: "id", Direction: gopager.DirectionASC},
        )
        if err != nil {
            http.Error(w, "Invalid pagination parameters", http.StatusBadRequest)
            return
        }
        
        // Apply pagination with lookahead to detect if there are more pages
        pager = pager.WithLookahead()
        
        var users []User
        db, err := pager.Paginate(db.Model(&User{}))
        if err != nil {
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }
        
        result := db.Find(&users)
        if result.Error != nil {
            http.Error(w, "Query error", http.StatusInternalServerError)
            return
        }
        
        // Check if this is the last page
        isLastPage := gopager.IsLastPage(pager, users)
        trimmedUsers := gopager.TrimResultSet(pager, users)
        
        // Generate next page cursor if not last page
        var nextCursor *gopager.DefaultCursor
        if !isLastPage {
            getters := gopager.Getters[User]{
                "id": func(u User) any { return u.ID },
            }
            _, nextCursor, err = gopager.NextPageCursor(pager, trimmedUsers, getters)
            if err != nil {
                http.Error(w, "Cursor generation error", http.StatusInternalServerError)
                return
            }
        }
        
        response := PaginationResponse[User]{
            Items:         trimmedUsers,
            NextPageToken: nextCursor,
            HasMore:       nextCursor != nil,
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }
}
```

## API Reference
### Core Types
#### CursorPager
The main pagination structure that handles cursor-based pagination.
#### DefaultCursor
A cursor that uses complex filtering conditions for precise pagination.
#### PseudoCursor
A simple cursor that uses OFFSET for pagination.

### Main Functions
#### Creating Pagers
```go
// Create a new cursor pager
func NewCursorPager[CursorType Cursor]() *CursorPager[CursorType]

// Decode from raw pagination request
func DecodeCursorPager(limit int, rawStartToken string, orderBy ...OrderBy) (*CursorPager[*DefaultCursor], error)
func DecodePseudoCursorPager(limit int, rawStartToken string, orderBy ...OrderBy) (*CursorPager[*PseudoCursor], error)
```

#### Pager Configuration
```go
// Set the limit for results
func (c *CursorPager[CursorType]) WithLimit(limit int) *CursorPager[CursorType]

// Enable unlimited results (cannot be used with lookahead)
func (c *CursorPager[CursorType]) WithUnlimited() *CursorPager[CursorType]

// Enable lookahead to detect if there are more pages
func (c *CursorPager[CursorType]) WithLookahead() *CursorPager[CursorType]

// Set sorting order
func (c *CursorPager[CursorType]) WithSort(orderBy ...OrderBy) *CursorPager[CursorType]

// Substitute sorting order (replaces existing)
func (c *CursorPager[CursorType]) WithSubstitutedSort(orderBy ...OrderBy) *CursorPager[CursorType]
```

### Custom Column Mapping
```go
// Map external column names to internal database columns
columnMapping := gopager.ColumnMapping{
    "user_id":    "users.id",
    "created":    "users.created_at",
    "email_addr": "users.email",
}

// Parse sort parameters with column mapping
sortParams := []string{"user_id asc", "created desc"}
orderings, err := gopager.ParseSort(sortParams, columnMapping)
if err != nil {
    log.Fatal(err)
}

pager := gopager.NewCursorPager[*gopager.DefaultCursor]().
    WithSort(orderings...)
```

## Performance Considerations
1. **Use DefaultCursor for large datasets** - More efficient than offset-based pagination
2. **Use PseudoCursor for simple cases** - When you need simple offset-based pagination
3. **Always include a unique column in sorting** - Required for consistent pagination
4. **Set appropriate limits** - Avoid unlimited queries in production
