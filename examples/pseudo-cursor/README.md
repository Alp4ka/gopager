# Pseudo Cursor Example

This example demonstrates the usage of GoPager with PseudoCursor for simple offset-based pagination.

## When to Use PseudoCursor

- Simple pagination scenarios where you don't need complex cursor logic
- When you want to use traditional OFFSET/LIMIT pagination
- For smaller datasets where performance is not critical
- When you need simple page navigation (page 1, 2, 3, etc.)

## Features Demonstrated

- Basic pseudo cursor pagination with OFFSET/LIMIT
- Pseudo cursor with database filtering
- Multi-page navigation simulation
- Pseudo cursor with lookahead functionality

## Running the Example

```bash
cd examples/pseudo-cursor
go run main.go
```

## What You'll See

1. **Basic Pseudo Cursor**: Shows products with simple offset-based pagination
2. **Filtered Pagination**: Shows only in-stock products with pseudo cursor
3. **Multi-page Navigation**: Simulates navigating through multiple pages
4. **Lookahead with Pseudo Cursor**: Demonstrates lookahead functionality

## Key Concepts

- **PseudoCursor**: Uses OFFSET for pagination instead of complex filtering
- **Offset-based**: Traditional pagination using OFFSET and LIMIT
- **Simple Navigation**: Easy to implement page-based navigation
- **Performance Trade-offs**: Simpler but potentially slower for large datasets

## PseudoCursor vs DefaultCursor

| Feature | PseudoCursor | DefaultCursor |
|---------|--------------|---------------|
| Complexity | Simple | Complex |
| Performance | Good for small datasets | Excellent for large datasets |
| Use Case | Page-based navigation | Cursor-based navigation |
| Implementation | OFFSET/LIMIT | Complex WHERE conditions |
| Consistency | May have issues with concurrent inserts | Consistent results |
