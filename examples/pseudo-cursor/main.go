package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Alp4ka/gopager"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Product represents a product in our e-commerce system
type Product struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	InStock     bool      `json:"in_stock"`
	CreatedAt   time.Time `json:"created_at"`
}

func main() {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&Product{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed some test data
	seedData(db)

	// Example 1: Basic pseudo cursor pagination
	fmt.Println("=== Example 1: Basic Pseudo Cursor Pagination ===")
	basicPseudoPaginationExample(db)

	// Example 2: Pseudo cursor with filtering
	fmt.Println("\n=== Example 2: Pseudo Cursor with Filtering ===")
	filteredPseudoPaginationExample(db)

	// Example 3: Simulating pagination through multiple pages
	fmt.Println("\n=== Example 3: Simulating Multi-page Navigation ===")
	multiPageNavigationExample(db)

	// Example 4: Pseudo cursor with lookahead
	fmt.Println("\n=== Example 4: Pseudo Cursor with Lookahead ===")
	pseudoLookaheadExample(db)
}

func basicPseudoPaginationExample(db *gorm.DB) {
	// Create a pseudo cursor pager (uses OFFSET/LIMIT)
	pager := gopager.NewCursorPager[*gopager.PseudoCursor]().
		WithLimit(4).
		WithSort(
			gopager.OrderBy{Column: "id", Direction: gopager.DirectionASC},
		)

	// Apply pagination to query
	query, err := pager.Paginate(db.Model(&Product{}))
	if err != nil {
		log.Fatal("Failed to apply pagination:", err)
	}

	// Execute query
	var products []Product
	result := query.Find(&products)
	if result.Error != nil {
		log.Fatal("Failed to execute query:", result.Error)
	}

	// Display results
	fmt.Printf("Found %d products (page 1):\n", len(products))
	for _, product := range products {
		fmt.Printf("  ID: %d, Name: %s, Price: $%.2f, Category: %s\n",
			product.ID, product.Name, product.Price, product.Category)
	}

	// Generate next page pseudo cursor
	_, nextCursor, err := gopager.NextPagePseudoCursor(pager, products)
	if err != nil {
		log.Fatal("Failed to generate next cursor:", err)
	}

	if nextCursor != nil {
		fmt.Printf("Next page token: %s (offset: %d)\n", nextCursor.String(), nextCursor.GetOffset())
	} else {
		fmt.Println("This is the last page")
	}
}

func filteredPseudoPaginationExample(db *gorm.DB) {
	// Create pager with filtering
	pager := gopager.NewCursorPager[*gopager.PseudoCursor]().
		WithLimit(3).
		WithSort(
			gopager.OrderBy{Column: "price", Direction: gopager.DirectionASC},
		)

	// Apply pagination to filtered query
	query, err := pager.Paginate(db.Model(&Product{}).Where("in_stock = ?", true))
	if err != nil {
		log.Fatal("Failed to apply pagination:", err)
	}

	var products []Product
	result := query.Find(&products)
	if result.Error != nil {
		log.Fatal("Failed to execute query:", result.Error)
	}

	fmt.Printf("Found %d in-stock products (sorted by price):\n", len(products))
	for _, product := range products {
		fmt.Printf("  Name: %s, Price: $%.2f, Category: %s\n",
			product.Name, product.Price, product.Category)
	}

	// Generate next page cursor
	_, nextCursor, err := gopager.NextPagePseudoCursor(pager, products)
	if err != nil {
		log.Fatal("Failed to generate next cursor:", err)
	}

	if nextCursor != nil {
		fmt.Printf("Next page token: %s\n", nextCursor.String())
	}
}

func multiPageNavigationExample(db *gorm.DB) {
	// Simulate navigating through multiple pages
	pageSize := 3
	currentOffset := 0

	for page := 1; page <= 3; page++ {
		fmt.Printf("\n--- Page %d (offset: %d) ---\n", page, currentOffset)

		// Create pseudo cursor with current offset
		pager := gopager.NewCursorPager[*gopager.PseudoCursor]().
			WithLimit(pageSize).
			WithSort(
				gopager.OrderBy{Column: "name", Direction: gopager.DirectionASC},
			)

		// Set the offset manually
		pager = pager.WithCursor(gopager.NewPseudoCursor(currentOffset))

		// Apply pagination
		query, err := pager.Paginate(db.Model(&Product{}))
		if err != nil {
			log.Fatal("Failed to apply pagination:", err)
		}

		var products []Product
		result := query.Find(&products)
		if result.Error != nil {
			log.Fatal("Failed to execute query:", result.Error)
		}

		if len(products) == 0 {
			fmt.Println("No more products")
			break
		}

		fmt.Printf("Products on page %d:\n", page)
		for _, product := range products {
			fmt.Printf("  %s - $%.2f\n", product.Name, product.Price)
		}

		// Update offset for next page
		currentOffset += len(products)
	}
}

func pseudoLookaheadExample(db *gorm.DB) {
	// Create pager with lookahead
	pager := gopager.NewCursorPager[*gopager.PseudoCursor]().
		WithLimit(3).
		WithLookahead().
		WithSort(
			gopager.OrderBy{Column: "created_at", Direction: gopager.DirectionDESC},
		)

	// Apply pagination
	query, err := pager.Paginate(db.Model(&Product{}))
	if err != nil {
		log.Fatal("Failed to apply pagination:", err)
	}

	var products []Product
	result := query.Find(&products)
	if result.Error != nil {
		log.Fatal("Failed to execute query:", result.Error)
	}

	// Check if this is the last page
	isLastPage := gopager.IsLastPage(pager, products)
	trimmedProducts := gopager.TrimResultSet(pager, products)

	fmt.Printf("Found %d products (newest first, with lookahead):\n", len(trimmedProducts))
	for _, product := range trimmedProducts {
		fmt.Printf("  %s - Created: %s\n",
			product.Name, product.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	if isLastPage {
		fmt.Println("This is the last page")
	} else {
		fmt.Println("There are more pages available")
	}
}

func seedData(db *gorm.DB) {
	products := []Product{
		{Name: "Laptop Pro", Description: "High-performance laptop", Price: 1299.99, Category: "Electronics", InStock: true, CreatedAt: time.Now().Add(-5 * time.Hour)},
		{Name: "Wireless Mouse", Description: "Ergonomic wireless mouse", Price: 29.99, Category: "Electronics", InStock: true, CreatedAt: time.Now().Add(-4 * time.Hour)},
		{Name: "Office Chair", Description: "Comfortable office chair", Price: 199.99, Category: "Furniture", InStock: false, CreatedAt: time.Now().Add(-3 * time.Hour)},
		{Name: "Coffee Maker", Description: "Automatic coffee maker", Price: 89.99, Category: "Appliances", InStock: true, CreatedAt: time.Now().Add(-2 * time.Hour)},
		{Name: "Desk Lamp", Description: "LED desk lamp", Price: 45.99, Category: "Furniture", InStock: true, CreatedAt: time.Now().Add(-1 * time.Hour)},
		{Name: "Bluetooth Speaker", Description: "Portable Bluetooth speaker", Price: 79.99, Category: "Electronics", InStock: true, CreatedAt: time.Now()},
		{Name: "Notebook Set", Description: "Set of 3 notebooks", Price: 12.99, Category: "Stationery", InStock: true, CreatedAt: time.Now().Add(1 * time.Hour)},
		{Name: "Water Bottle", Description: "Insulated water bottle", Price: 24.99, Category: "Accessories", InStock: false, CreatedAt: time.Now().Add(2 * time.Hour)},
		{Name: "Backpack", Description: "Laptop backpack", Price: 59.99, Category: "Accessories", InStock: true, CreatedAt: time.Now().Add(3 * time.Hour)},
		{Name: "Phone Case", Description: "Protective phone case", Price: 19.99, Category: "Accessories", InStock: true, CreatedAt: time.Now().Add(4 * time.Hour)},
	}

	for _, product := range products {
		db.Create(&product)
	}
}
