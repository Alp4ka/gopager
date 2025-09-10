# GoPager Examples

This directory contains comprehensive examples demonstrating different use cases and features of the GoPager library.

## Examples Overview

### 1. [Basic Usage](./basic/)
Demonstrates fundamental GoPager features with DefaultCursor:
- Basic pagination with single and multi-column sorting
- Lookahead pagination to detect if there are more pages
- Unlimited results pagination
- Next page cursor generation

**Best for:** Getting started with GoPager, understanding core concepts

### 2. [Pseudo Cursor](./pseudo-cursor/)
Shows how to use PseudoCursor for simple offset-based pagination:
- Basic pseudo cursor pagination with OFFSET/LIMIT
- Pseudo cursor with database filtering
- Multi-page navigation simulation
- Pseudo cursor with lookahead functionality

**Best for:** Simple pagination needs, small datasets, page-based navigation

### 3. [HTTP API](./http-api/)
Complete HTTP API integration example:
- RESTful API with cursor-based pagination
- Support for both GET and POST requests
- Custom sorting parameters
- Error handling and validation
- HTML documentation endpoint

**Best for:** Building web APIs, understanding production patterns

### 4. [Advanced Features](./advanced/)
Demonstrates advanced features and production best practices:
- Multi-table pagination with complex joins
- Dynamic sorting with column mapping
- Comprehensive error handling and validation
- Custom business rule validation
- Performance optimization techniques

**Best for:** Production applications, complex use cases, performance optimization

### 5. [Performance Benchmark](./benchmark/)
Performance analysis and benchmarking tools:
- Performance comparison between cursor types
- Memory usage analysis
- Scalability testing
- Go benchmark functions
- Performance optimization tips

**Best for:** Performance analysis, choosing the right pagination strategy

## Quick Start

1. **Choose an example** based on your use case
2. **Navigate to the example directory**
3. **Run the example**:
   ```bash
   cd examples/basic
   go run main.go
   ```
4. **Read the example's README** for detailed explanations
5. **Adapt the code** to your specific needs

## Example Selection Guide

| Use Case | Recommended Example | Why |
|----------|-------------------|-----|
| Learning GoPager | [Basic](./basic/) | Covers all fundamental concepts |
| Simple pagination | [Pseudo Cursor](./pseudo-cursor/) | Easy to understand and implement |
| Building APIs | [HTTP API](./http-api/) | Complete API integration example |
| Production app | [Advanced](./advanced/) | Best practices and optimization |
| Performance analysis | [Benchmark](./benchmark/) | Performance testing and optimization |

## Common Patterns

### Basic Pagination
```go
pager := gopager.NewCursorPager[*gopager.DefaultCursor]().
    WithLimit(10).
    WithSort(
        gopager.OrderBy{Column: "id", Direction: gopager.DirectionASC},
    )
```

### HTTP API Integration
```go
type PaginationRequest struct {
    gopager.RawCursorPager `json:",inline"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req PaginationRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    pager, err := req.Decode(orderBy...)
    // ... handle pagination
}
```

### Error Handling
```go
pager, err := gopager.DecodeCursorPager(limit, startToken, orderBy...)
if err != nil {
    // Handle different error types
    switch {
    case strings.Contains(err.Error(), "failed to decode base64"):
        // Invalid cursor token
    case strings.Contains(err.Error(), "cursor column number mismatch"):
        // Cursor/ordering mismatch
    default:
        // Other errors
    }
}
```

## Running All Examples

To run all examples and see their output:

```bash
# Run basic example
cd examples/basic && go run main.go

# Run pseudo cursor example
cd examples/pseudo-cursor && go run main.go

# Run HTTP API example (starts server on :8080)
cd examples/http-api && go run main.go

# Run advanced example
cd examples/advanced && go run main.go

# Run benchmark example
cd examples/benchmark && go run main.go
```

## Dependencies

All examples use the same core dependencies:
- Go 1.24+
- GORM v1.25.5+
- SQLite driver (for in-memory testing)
- GoPager library

## Contributing

When adding new examples:
1. Create a new directory with a descriptive name
2. Include a `main.go` file with the example code
3. Add a `go.mod` file with proper dependencies
4. Write a comprehensive `README.md` explaining the example
5. Update this main README to include the new example

## Questions?

If you have questions about any example or need help adapting them to your use case, please refer to the main GoPager documentation or create an issue in the repository.
