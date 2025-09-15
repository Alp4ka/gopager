# GoPager - Cursor-Based Pagination for Go
A cursor-based pagination library for Go applications using GORM. 
GoPager provides efficient pagination for large datasets without the performance issues of traditional offset-based pagination.

[![tag](https://img.shields.io/github/tag/Alp4ka/gopager.svg)](https://github.com/Alp4ka/gopager/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.23-%23007d9c)
![Build Status](https://github.com/Alp4ka/gopager/actions/workflows/test.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/Alp4ka/gopager)](https://goreportcard.com/report/github.com/Alp4ka/gopager)
[![Coverage](https://img.shields.io/codecov/c/github/Alp4ka/gopager)](https://codecov.io/gh/Alp4ka/gopager)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/Alp4ka/gopager)](https://pkg.go.dev/github.com/Alp4ka/gopager)
[![License](https://img.shields.io/github/license/Alp4ka/gopager)](./LICENSE)

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
### [Examples](examples)
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
	...
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




## Core Types
### CursorPager
The main structure for handling cursor-based pagination. 
Create a new instance with `NewCursorPager` or decode an existing one from a string using 
`DecodeCursorPager` or `DecodePseudoCursorPager`.
#### WithLimit(limit int)
Set the select-request fetch limit.
#### WithUnlimited()
Enable unlimited select (cannot be used with lookahead option).
#### WithLookahead()
Enable lookahead to detect if there are more pages. 
If the page is the last one in the dataset, the next token will be nil.
#### WithSort(orderBy ...OrderBy)
Set sorting order. MUST include a column with a unique constraint.
#### WithSubstitutedSort(orderBy ...OrderBy)
This function works the same as `CursorPager.WithSort`, 
but it clears all existing sorts and replaces them with the new ones.
#### Paginate(db *gorm.DB)
Applies pagination to the select statement.

### DefaultCursor
A cursor that uses complex filtering conditions for precise pagination. 
It relies on the values of a specific field from the last element of the previous page. 
For example, given the unique sorted dataset `[1, 2, 3, 4]` and a previous page of `[1, 2]`, 
the next page would start from the first element satisfying the condition `x > 2`. 
This approach is significantly faster than using `LIMIT/OFFSET` on large datasets. 

This cursor type requires:
1. Sorted dataset;
2. At least one unique element is required for correct filtering.

### PseudoCursor
A simple cursor that uses `LIMIT/OFFSET` for pagination. It's less efficient on big datasets.

### ParseSort
Converts a list of strings to the list of sorts. 
It is considered that each string is given in the next format: `<column_alias> <ASC/DESC/asc/desc>`.
You should pass `ColumnMapping` as an argument in order to convert `column_alias` to a real column name inside the dataset.
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
