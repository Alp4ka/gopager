package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Alp4ka/gopager"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Order represents an order in our e-commerce system
type Order struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null" json:"user_id"`
	ProductID  uint      `gorm:"not null" json:"product_id"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// User represents a user
type User struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"not null" json:"name"`
}

// Product represents a product
type Product struct {
	ID    uint    `gorm:"primaryKey" json:"id"`
	Name  string  `gorm:"not null" json:"name"`
	Price float64 `json:"price"`
}

// OrderWithDetails represents an order with joined data
type OrderWithDetails struct {
	Order
	UserName     string  `json:"user_name"`
	ProductName  string  `json:"product_name"`
	ProductPrice float64 `json:"product_price"`
}

func main() {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schemas
	err = db.AutoMigrate(&Order{}, &User{}, &Product{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed test data
	seedData(db)

	// Example 1: Complex multi-table pagination with joins
	fmt.Println("=== Example 1: Multi-table Pagination with Joins ===")
	multiTablePaginationExample(db)

	// Example 2: Dynamic sorting with column mapping
	fmt.Println("\n=== Example 2: Dynamic Sorting with Column Mapping ===")
	dynamicSortingExample(db)

	// Example 3: Error handling and validation
	fmt.Println("\n=== Example 3: Error Handling and Validation ===")
	errorHandlingExample(db)

	// Example 4: Custom cursor validation
	fmt.Println("\n=== Example 4: Custom Cursor Validation ===")
	customValidationExample(db)

	// Example 5: Performance optimization with indexes
	fmt.Println("\n=== Example 5: Performance Optimization ===")
	performanceOptimizationExample(db)
}

func multiTablePaginationExample(db *gorm.DB) {
	// Create a complex query with joins
	baseQuery := db.Table("orders").
		Select("orders.*, users.name as user_name, products.name as product_name, products.price as product_price").
		Joins("LEFT JOIN users ON orders.user_id = users.id").
		Joins("LEFT JOIN products ON orders.product_id = products.id").
		Where("orders.status = ?", "completed")

	// Create pager with multi-column sorting
	pager := gopager.NewCursorPager[*gopager.DefaultCursor]().
		WithLimit(4).
		WithSort(
			gopager.OrderBy{Column: "orders.created_at", Direction: gopager.DirectionDESC},
			gopager.OrderBy{Column: "orders.id", Direction: gopager.DirectionASC},
		)

	// Apply pagination
	query, err := pager.Paginate(baseQuery)
	if err != nil {
		log.Fatal("Failed to apply pagination:", err)
	}

	// Execute query
	var orders []OrderWithDetails
	result := query.Find(&orders)
	if result.Error != nil {
		log.Fatal("Failed to execute query:", result.Error)
	}

	fmt.Printf("Found %d completed orders:\n", len(orders))
	for _, order := range orders {
		fmt.Printf("  Order #%d: %s bought %s (qty: %d, $%.2f) on %s\n",
			order.ID, order.UserName, order.ProductName, order.Quantity,
			order.TotalPrice, order.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	// Generate next page cursor
	getters := gopager.Getters[OrderWithDetails]{
		"orders.created_at": func(o OrderWithDetails) any { return o.CreatedAt },
		"orders.id":         func(o OrderWithDetails) any { return o.ID },
	}

	_, nextCursor, err := gopager.NextPageCursor(pager, orders, getters)
	if err != nil {
		log.Fatal("Failed to generate next cursor:", err)
	}

	if nextCursor != nil {
		fmt.Printf("Next page token: %s\n", nextCursor.String())
	}
}

func dynamicSortingExample(db *gorm.DB) {
	// Simulate dynamic sort parameters from user input
	sortParams := []string{"total_price desc", "created_at asc", "id asc"}

	// Define column mapping for complex queries
	columnMapping := gopager.ColumnMapping{
		"id":          "orders.id",
		"user_id":     "orders.user_id",
		"product_id":  "orders.product_id",
		"quantity":    "orders.quantity",
		"total_price": "orders.total_price",
		"status":      "orders.status",
		"created_at":  "orders.created_at",
		"updated_at":  "orders.updated_at",
	}

	// Parse sort parameters
	orderBy, err := gopager.ParseSort(sortParams, columnMapping)
	if err != nil {
		log.Fatal("Failed to parse sort parameters:", err)
	}

	// Create pager with dynamic sorting
	pager := gopager.NewCursorPager[*gopager.DefaultCursor]().
		WithLimit(3).
		WithSort(orderBy...)

	// Apply pagination
	query, err := pager.Paginate(db.Model(&Order{}))
	if err != nil {
		log.Fatal("Failed to apply pagination:", err)
	}

	var orders []Order
	result := query.Find(&orders)
	if result.Error != nil {
		log.Fatal("Failed to execute query:", result.Error)
	}

	fmt.Printf("Found %d orders (sorted by total_price DESC, created_at ASC):\n", len(orders))
	for _, order := range orders {
		fmt.Printf("  Order #%d: $%.2f, Status: %s, Created: %s\n",
			order.ID, order.TotalPrice, order.Status, order.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	// Generate next page cursor
	getters := gopager.Getters[Order]{
		"orders.total_price": func(o Order) any { return o.TotalPrice },
		"orders.created_at":  func(o Order) any { return o.CreatedAt },
		"orders.id":          func(o Order) any { return o.ID },
	}

	_, nextCursor, err := gopager.NextPageCursor(pager, orders, getters)
	if err != nil {
		log.Fatal("Failed to generate next cursor:", err)
	}

	if nextCursor != nil {
		fmt.Printf("Next page token: %s\n", nextCursor.String())
	}
}

func errorHandlingExample(db *gorm.DB) {
	// Test various error conditions
	fmt.Println("Testing error conditions:")

	// 1. Invalid cursor token
	fmt.Println("\n1. Testing invalid cursor token:")
	_, err := gopager.DecodeCursor("invalid_base64_token")
	if err != nil {
		fmt.Printf("   Expected error: %v\n", err)
	}

	// 2. Invalid sort parameters
	fmt.Println("\n2. Testing invalid sort parameters:")
	invalidSortParams := []string{"invalid_column asc", "id desc"}
	columnMapping := gopager.ColumnMapping{
		"id": "orders.id",
	}

	_, err = gopager.ParseSort(invalidSortParams, columnMapping)
	if err != nil {
		fmt.Printf("   Expected error: %v\n", err)
	}

	// 3. Cursor/ordering mismatch
	fmt.Println("\n3. Testing cursor/ordering mismatch:")
	// Create a cursor with one column
	cursor := gopager.NewDefaultCursor(
		gopager.CursorElement{
			Column:   "id",
			Value:    1,
			Operator: gopager.OperatorGT,
		},
	)

	// Try to use it with different ordering
	pager := gopager.NewCursorPager[*gopager.DefaultCursor]().
		WithCursor(cursor).
		WithSort(
			gopager.OrderBy{Column: "id", Direction: gopager.DirectionASC},
			gopager.OrderBy{Column: "created_at", Direction: gopager.DirectionDESC},
		)

	// 4. Invalid limit
	fmt.Println("\n4. Testing invalid limit:")
	pager = gopager.NewCursorPager[*gopager.DefaultCursor]().
		WithLimit(-5) // Invalid negative limit

	_, err = pager.Paginate(db.Model(&Order{}))
	if err != nil {
		fmt.Printf("   Expected error: %v\n", err)
	}
}

func customValidationExample(db *gorm.DB) {
	// Create a custom validation scenario
	fmt.Println("Testing custom validation scenarios:")

	// 1. Validate cursor with business rules
	fmt.Println("\n1. Testing business rule validation:")

	// Create a cursor that might violate business rules
	cursor := gopager.NewDefaultCursor(
		gopager.CursorElement{
			Column:   "created_at",
			Value:    time.Now().Add(-30 * 24 * time.Hour), // 30 days ago
			Operator: gopager.OperatorGT,
		},
	)

	// Check if cursor is too old (business rule: don't allow cursors older than 7 days)
	if !cursor.IsEmpty() {
		elements := cursor.GetElements()
		if len(elements) > 0 {
			createdAt, ok := elements[0].Value.(time.Time)
			if ok && time.Since(createdAt) > 7*24*time.Hour {
				fmt.Printf("   Business rule violation: Cursor is too old (%v)\n", time.Since(createdAt))
			} else {
				fmt.Printf("   Cursor is valid: %v\n", createdAt.Format("2006-01-02 15:04:05"))
			}
		}
	}

	// 2. Validate pagination parameters
	fmt.Println("\n2. Testing pagination parameter validation:")

	// Test various limit scenarios
	limits := []int{-1, 0, 5, 100, 1000}
	for _, limit := range limits {
		normalized := gopager.NormalizeLimit(limit)
		fmt.Printf("   Limit %d -> normalized to %d\n", limit, normalized)
	}
}

func performanceOptimizationExample(db *gorm.DB) {
	// Demonstrate performance considerations
	fmt.Println("Performance optimization considerations:")

	// 1. Use appropriate indexes
	fmt.Println("\n1. Creating indexes for better performance:")

	// Create indexes on commonly queried columns
	db.Exec("CREATE INDEX idx_orders_status ON orders(status)")
	db.Exec("CREATE INDEX idx_orders_created_at ON orders(created_at)")
	db.Exec("CREATE INDEX idx_orders_user_id ON orders(user_id)")

	fmt.Println("   Created indexes on status, created_at, and user_id columns")

	// 2. Use lookahead efficiently
	fmt.Println("\n2. Using lookahead efficiently:")

	pager := gopager.NewCursorPager[*gopager.DefaultCursor]().
		WithLimit(5).
		WithLookahead(). // Only use when you need to know if there are more pages
		WithSort(
			gopager.OrderBy{Column: "id", Direction: gopager.DirectionASC},
		)

	query, err := pager.Paginate(db.Model(&Order{}))
	if err != nil {
		log.Fatal("Failed to apply pagination:", err)
	}

	var orders []Order
	result := query.Find(&orders)
	if result.Error != nil {
		log.Fatal("Failed to execute query:", result.Error)
	}

	// Check if lookahead detected more pages
	isLastPage := gopager.IsLastPage(pager, orders)
	trimmedOrders := gopager.TrimResultSet(pager, orders)

	fmt.Printf("   Fetched %d orders (lookahead enabled)\n", len(trimmedOrders))
	fmt.Printf("   Is last page: %v\n", isLastPage)

	// 3. Avoid unlimited queries in production
	fmt.Println("\n3. Avoiding unlimited queries:")

	// This is fine for small datasets or admin interfaces
	unlimitedPager := gopager.NewCursorPager[*gopager.DefaultCursor]().
		WithUnlimited().
		WithSort(
			gopager.OrderBy{Column: "id", Direction: gopager.DirectionASC},
		)

	query, err = unlimitedPager.Paginate(db.Model(&Order{}))
	if err != nil {
		log.Fatal("Failed to apply pagination:", err)
	}

	var allOrders []Order
	result = query.Find(&allOrders)
	if result.Error != nil {
		log.Fatal("Failed to execute query:", result.Error)
	}

	fmt.Printf("   Unlimited query returned %d orders (use with caution in production)\n", len(allOrders))
}

func seedData(db *gorm.DB) {
	// Create users
	users := []User{
		{Name: "Alice Johnson"},
		{Name: "Bob Smith"},
		{Name: "Charlie Brown"},
		{Name: "Diana Prince"},
	}
	for _, user := range users {
		db.Create(&user)
	}

	// Create products
	products := []Product{
		{Name: "Laptop", Price: 999.99},
		{Name: "Mouse", Price: 29.99},
		{Name: "Keyboard", Price: 79.99},
		{Name: "Monitor", Price: 299.99},
		{Name: "Headphones", Price: 149.99},
	}
	for _, product := range products {
		db.Create(&product)
	}

	// Create orders
	orders := []Order{
		{UserID: 1, ProductID: 1, Quantity: 1, TotalPrice: 999.99, Status: "completed", CreatedAt: time.Now().Add(-5 * time.Hour)},
		{UserID: 2, ProductID: 2, Quantity: 2, TotalPrice: 59.98, Status: "completed", CreatedAt: time.Now().Add(-4 * time.Hour)},
		{UserID: 1, ProductID: 3, Quantity: 1, TotalPrice: 79.99, Status: "pending", CreatedAt: time.Now().Add(-3 * time.Hour)},
		{UserID: 3, ProductID: 4, Quantity: 1, TotalPrice: 299.99, Status: "completed", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{UserID: 2, ProductID: 5, Quantity: 1, TotalPrice: 149.99, Status: "completed", CreatedAt: time.Now().Add(-1 * time.Hour)},
		{UserID: 4, ProductID: 1, Quantity: 1, TotalPrice: 999.99, Status: "completed", CreatedAt: time.Now()},
		{UserID: 1, ProductID: 2, Quantity: 3, TotalPrice: 89.97, Status: "completed", CreatedAt: time.Now().Add(1 * time.Hour)},
		{UserID: 3, ProductID: 3, Quantity: 2, TotalPrice: 159.98, Status: "pending", CreatedAt: time.Now().Add(2 * time.Hour)},
	}
	for _, order := range orders {
		db.Create(&order)
	}
}
