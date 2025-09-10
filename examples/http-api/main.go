package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	City      string    `json:"city"`
	CreatedAt time.Time `json:"created_at"`
}

// PaginationRequest represents the incoming pagination request
type PaginationRequest struct {
	gopager.RawCursorPager `json:",inline"`
	Sort                   []string `json:"sort,omitempty"` // Optional custom sorting
}

// PaginationResponse represents the paginated response
type PaginationResponse[T any] struct {
	Items         []T                    `json:"items"`
	NextPageToken *gopager.DefaultCursor `json:"nextPageToken,omitempty"`
	HasMore       bool                   `json:"hasMore"`
	Total         int64                  `json:"total,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
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

	// Setup routes
	http.HandleFunc("/users", getUsersHandler(db))
	http.HandleFunc("/users/count", getUsersCountHandler(db))
	http.HandleFunc("/", indexHandler)

	fmt.Println("HTTP API Example Server starting on :8080")
	fmt.Println("Available endpoints:")
	fmt.Println("  GET  /users - Get paginated users")
	fmt.Println("  POST /users - Get paginated users with custom parameters")
	fmt.Println("  GET  /users/count - Get total user count")
	fmt.Println("  GET  / - API documentation")
	fmt.Println("\nExample requests:")
	fmt.Println("  curl http://localhost:8080/users?limit=5")
	fmt.Println("  curl -X POST http://localhost:8080/users -d '{\"limit\":3,\"sort\":[\"age desc\",\"name asc\"]}'")
	fmt.Println("  curl http://localhost:8080/users?limit=2&startToken=<token_from_previous_response>")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getUsersHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req PaginationRequest
		var err error

		// Handle both GET and POST requests
		if r.Method == "GET" {
			// Parse query parameters for GET request
			req.Limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
			req.StartToken = r.URL.Query().Get("startToken")

			// Parse sort parameter (comma-separated)
			if sortParam := r.URL.Query().Get("sort"); sortParam != "" {
				req.Sort = []string{sortParam}
			}
		} else if r.Method == "POST" {
			// Parse JSON body for POST request
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				sendError(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
		} else {
			sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Set default limit if not provided
		if req.Limit <= 0 {
			req.Limit = 10
		}

		// Parse sorting
		var orderBy []gopager.OrderBy
		if len(req.Sort) > 0 {
			// Parse custom sort parameters
			columnMapping := gopager.ColumnMapping{
				"id":         "id",
				"name":       "name",
				"email":      "email",
				"age":        "age",
				"city":       "city",
				"created_at": "created_at",
			}

			orderBy, err = gopager.ParseSort(req.Sort, columnMapping)
			if err != nil {
				sendError(w, "Invalid sort parameter: "+err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			// Default sorting
			orderBy = []gopager.OrderBy{
				{Column: "id", Direction: gopager.DirectionASC},
			}
		}

		// Decode the cursor pager
		pager, err := req.Decode(orderBy...)
		if err != nil {
			sendError(w, "Invalid pagination parameters: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Enable lookahead to detect if there are more pages
		pager = pager.WithLookahead()

		// Apply pagination to query
		query, err := pager.Paginate(db.Model(&User{}))
		if err != nil {
			sendError(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Execute query
		var users []User
		result := query.Find(&users)
		if result.Error != nil {
			sendError(w, "Query error: "+result.Error.Error(), http.StatusInternalServerError)
			return
		}

		// Check if this is the last page
		isLastPage := gopager.IsLastPage(pager, users)
		trimmedUsers := gopager.TrimResultSet(pager, users)

		// Generate next page cursor if not last page
		var nextCursor *gopager.DefaultCursor
		if !isLastPage {
			getters := gopager.Getters[User]{
				"id":         func(u User) any { return u.ID },
				"name":       func(u User) any { return u.Name },
				"email":      func(u User) any { return u.Email },
				"age":        func(u User) any { return u.Age },
				"city":       func(u User) any { return u.City },
				"created_at": func(u User) any { return u.CreatedAt },
			}

			_, nextCursor, err = gopager.NextPageCursor(pager, trimmedUsers, getters)
			if err != nil {
				sendError(w, "Cursor generation error: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Send response
		response := PaginationResponse[User]{
			Items:         trimmedUsers,
			NextPageToken: nextCursor,
			HasMore:       nextCursor != nil,
		}

		json.NewEncoder(w).Encode(response)
	}
}

func getUsersCountHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var count int64
		result := db.Model(&User{}).Count(&count)
		if result.Error != nil {
			sendError(w, "Database error: "+result.Error.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]int64{"total": count}
		json.NewEncoder(w).Encode(response)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	html := `
<!DOCTYPE html>
<html>
<head>
    <title>GoPager HTTP API Example</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .method { font-weight: bold; color: #0066cc; }
        .example { background: #e8f4f8; padding: 10px; margin: 5px 0; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>GoPager HTTP API Example</h1>
    
    <h2>Available Endpoints</h2>
    
    <div class="endpoint">
        <span class="method">GET</span> /users - Get paginated users
        <div class="example">
            <strong>Query Parameters:</strong><br>
            • limit (int): Number of items per page (default: 10)<br>
            • startToken (string): Cursor token for pagination<br>
            • sort (string): Sort specification (e.g., "age desc,name asc")<br><br>
            <strong>Example:</strong><br>
            <code>curl "http://localhost:8080/users?limit=5&sort=age desc"</code>
        </div>
    </div>
    
    <div class="endpoint">
        <span class="method">POST</span> /users - Get paginated users with JSON body
        <div class="example">
            <strong>Request Body:</strong><br>
            <code>{"limit": 3, "startToken": "...", "sort": ["age desc", "name asc"]}</code><br><br>
            <strong>Example:</strong><br>
            <code>curl -X POST http://localhost:8080/users -d '{"limit":3,"sort":["age desc"]}'</code>
        </div>
    </div>
    
    <div class="endpoint">
        <span class="method">GET</span> /users/count - Get total user count
        <div class="example">
            <strong>Example:</strong><br>
            <code>curl http://localhost:8080/users/count</code>
        </div>
    </div>
    
    <h2>Response Format</h2>
    <div class="example">
        <pre>{
  "items": [...],
  "nextPageToken": "base64_encoded_cursor",
  "hasMore": true,
  "total": 100
}</pre>
    </div>
    
    <h2>Sorting Options</h2>
    <div class="example">
        Available fields: id, name, email, age, city, created_at<br>
        Directions: asc, desc<br>
        Examples: "age desc", "name asc", "created_at desc"
    </div>
</body>
</html>`

	fmt.Fprint(w, html)
}

func sendError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	json.NewEncoder(w).Encode(response)
}

func seedData(db *gorm.DB) {
	users := []User{
		{Name: "Alice Johnson", Email: "alice@example.com", Age: 25, City: "New York", CreatedAt: time.Now().Add(-24 * time.Hour)},
		{Name: "Bob Smith", Email: "bob@example.com", Age: 30, City: "Los Angeles", CreatedAt: time.Now().Add(-23 * time.Hour)},
		{Name: "Charlie Brown", Email: "charlie@example.com", Age: 25, City: "Chicago", CreatedAt: time.Now().Add(-22 * time.Hour)},
		{Name: "Diana Prince", Email: "diana@example.com", Age: 28, City: "Miami", CreatedAt: time.Now().Add(-21 * time.Hour)},
		{Name: "Eve Wilson", Email: "eve@example.com", Age: 32, City: "Seattle", CreatedAt: time.Now().Add(-20 * time.Hour)},
		{Name: "Frank Miller", Email: "frank@example.com", Age: 27, City: "Boston", CreatedAt: time.Now().Add(-19 * time.Hour)},
		{Name: "Grace Lee", Email: "grace@example.com", Age: 29, City: "San Francisco", CreatedAt: time.Now().Add(-18 * time.Hour)},
		{Name: "Henry Davis", Email: "henry@example.com", Age: 31, City: "Denver", CreatedAt: time.Now().Add(-17 * time.Hour)},
		{Name: "Ivy Chen", Email: "ivy@example.com", Age: 26, City: "Portland", CreatedAt: time.Now().Add(-16 * time.Hour)},
		{Name: "Jack Wilson", Email: "jack@example.com", Age: 33, City: "Austin", CreatedAt: time.Now().Add(-15 * time.Hour)},
		{Name: "Kate Brown", Email: "kate@example.com", Age: 24, City: "Nashville", CreatedAt: time.Now().Add(-14 * time.Hour)},
		{Name: "Leo Garcia", Email: "leo@example.com", Age: 35, City: "Phoenix", CreatedAt: time.Now().Add(-13 * time.Hour)},
	}

	for _, user := range users {
		db.Create(&user)
	}
}
