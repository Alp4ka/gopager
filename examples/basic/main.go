package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Alp4ka/gopager"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User represents a user in our system
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Email     string    `gorm:"uniqueIndex" json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func main() {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&User{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed some test data
	seedData(db)

	// Example 1: Basic pagination with single column sorting
	fmt.Println("=== Example 1: Basic Pagination ===")
	basicPaginationExample(db)

	// Example 2: Multi-column sorting
	fmt.Println("\n=== Example 2: Multi-column Sorting ===")
	multiColumnSortingExample(db)

	// Example 3: Pagination with lookahead
	fmt.Println("\n=== Example 3: Pagination with Lookahead ===")
	lookaheadPaginationExample(db)

	// Example 4: Pagination with unlimited results
	fmt.Println("\n=== Example 4: Unlimited Results ===")
	unlimitedResultsExample(db)
}

func basicPaginationExample(db *gorm.DB) {
	// Create a basic cursor pager
	pager := gopager.NewCursorPager[*gopager.DefaultCursor]().
		WithLimit(3).
		WithSort(
			gopager.OrderBy{Column: "id", Direction: gopager.DirectionASC},
		)

	// Apply pagination to query
	query, err := pager.Paginate(db.Model(&User{}))
	if err != nil {
		log.Fatal("Failed to apply pagination:", err)
	}

	// Execute query
	var users []User
	result := query.Find(&users)
	if result.Error != nil {
		log.Fatal("Failed to execute query:", result.Error)
	}

	// Display results
	fmt.Printf("Found %d users:\n", len(users))
	for _, user := range users {
		fmt.Printf("  ID: %d, Name: %s, Email: %s, Age: %d\n",
			user.ID, user.Name, user.Email, user.Age)
	}

	// Generate next page cursor
	getters := gopager.Getters[User]{
		"id": func(u User) any { return u.ID },
	}

	_, nextCursor, err := gopager.NextPageCursor(pager, users, getters)
	if err != nil {
		log.Fatal("Failed to generate next cursor:", err)
	}

	if nextCursor != nil {
		fmt.Printf("Next page token: %s\n", nextCursor.String())
	} else {
		fmt.Println("This is the last page")
	}
}

func multiColumnSortingExample(db *gorm.DB) {
	// Create pager with multi-column sorting
	pager := gopager.NewCursorPager[*gopager.DefaultCursor]().
		WithLimit(3).
		WithSort(
			gopager.OrderBy{Column: "age", Direction: gopager.DirectionDESC},
			gopager.OrderBy{Column: "name", Direction: gopager.DirectionASC},
		)

	// Apply pagination
	query, err := pager.Paginate(db.Model(&User{}))
	if err != nil {
		log.Fatal("Failed to apply pagination:", err)
	}

	var users []User
	result := query.Find(&users)
	if result.Error != nil {
		log.Fatal("Failed to execute query:", result.Error)
	}

	fmt.Printf("Found %d users (sorted by age DESC, name ASC):\n", len(users))
	for _, user := range users {
		fmt.Printf("  Age: %d, Name: %s, Email: %s\n",
			user.Age, user.Name, user.Email)
	}

	// Generate next page cursor with multiple columns
	getters := gopager.Getters[User]{
		"age":  func(u User) any { return u.Age },
		"name": func(u User) any { return u.Name },
	}

	_, nextCursor, err := gopager.NextPageCursor(pager, users, getters)
	if err != nil {
		log.Fatal("Failed to generate next cursor:", err)
	}

	if nextCursor != nil {
		fmt.Printf("Next page token: %s\n", nextCursor.String())
	}
}

func lookaheadPaginationExample(db *gorm.DB) {
	// Create pager with lookahead to detect if there are more pages
	pager := gopager.NewCursorPager[*gopager.DefaultCursor]().
		WithLimit(3).
		WithLookahead().
		WithSort(
			gopager.OrderBy{Column: "created_at", Direction: gopager.DirectionASC},
		)

	// Apply pagination
	query, err := pager.Paginate(db.Model(&User{}))
	if err != nil {
		log.Fatal("Failed to apply pagination:", err)
	}

	var users []User
	result := query.Find(&users)
	if result.Error != nil {
		log.Fatal("Failed to execute query:", result.Error)
	}

	// Check if this is the last page
	isLastPage := gopager.IsLastPage(pager, users)
	trimmedUsers := gopager.TrimResultSet(pager, users)

	fmt.Printf("Found %d users (with lookahead):\n", len(trimmedUsers))
	for _, user := range trimmedUsers {
		fmt.Printf("  ID: %d, Name: %s, Created: %s\n",
			user.ID, user.Name, user.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	if isLastPage {
		fmt.Println("This is the last page")
	} else {
		fmt.Println("There are more pages available")
	}
}

func unlimitedResultsExample(db *gorm.DB) {
	// Create pager with unlimited results
	pager := gopager.NewCursorPager[*gopager.DefaultCursor]().
		WithUnlimited().
		WithSort(
			gopager.OrderBy{Column: "id", Direction: gopager.DirectionASC},
		)

	// Apply pagination
	query, err := pager.Paginate(db.Model(&User{}))
	if err != nil {
		log.Fatal("Failed to apply pagination:", err)
	}

	var users []User
	result := query.Find(&users)
	if result.Error != nil {
		log.Fatal("Failed to execute query:", result.Error)
	}

	fmt.Printf("Found %d users (unlimited results):\n", len(users))
	for _, user := range users {
		fmt.Printf("  ID: %d, Name: %s, Email: %s\n",
			user.ID, user.Name, user.Email)
	}
}

func seedData(db *gorm.DB) {
	users := []User{
		{Name: "Alice Johnson", Email: "alice@example.com", Age: 25, CreatedAt: time.Now().Add(-24 * time.Hour)},
		{Name: "Bob Smith", Email: "bob@example.com", Age: 30, CreatedAt: time.Now().Add(-23 * time.Hour)},
		{Name: "Charlie Brown", Email: "charlie@example.com", Age: 25, CreatedAt: time.Now().Add(-22 * time.Hour)},
		{Name: "Diana Prince", Email: "diana@example.com", Age: 28, CreatedAt: time.Now().Add(-21 * time.Hour)},
		{Name: "Eve Wilson", Email: "eve@example.com", Age: 32, CreatedAt: time.Now().Add(-20 * time.Hour)},
		{Name: "Frank Miller", Email: "frank@example.com", Age: 27, CreatedAt: time.Now().Add(-19 * time.Hour)},
		{Name: "Grace Lee", Email: "grace@example.com", Age: 29, CreatedAt: time.Now().Add(-18 * time.Hour)},
		{Name: "Henry Davis", Email: "henry@example.com", Age: 31, CreatedAt: time.Now().Add(-17 * time.Hour)},
	}

	for _, user := range users {
		db.Create(&user)
	}
}

