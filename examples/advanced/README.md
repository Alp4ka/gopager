# Advanced Features Example

This example demonstrates advanced features and best practices for using GoPager in production applications.

## Features Demonstrated

- Multi-table pagination with complex joins
- Dynamic sorting with column mapping
- Comprehensive error handling and validation
- Custom business rule validation
- Performance optimization techniques
- Production-ready patterns

## Running the Example

```bash
cd examples/advanced
go run main.go
```

## Advanced Features Covered

### 1. Multi-table Pagination with Joins

Demonstrates how to paginate complex queries with multiple table joins:

```go
baseQuery := db.Table("orders").
    Select("orders.*, users.name as user_name, products.name as product_name").
    Joins("LEFT JOIN users ON orders.user_id = users.id").
    Joins("LEFT JOIN products ON orders.product_id = products.id").
    Where("orders.status = ?", "completed")
```

### 2. Dynamic Sorting with Column Mapping

Shows how to handle user-provided sort parameters safely:

```go
columnMapping := gopager.ColumnMapping{
    "id":           "orders.id",
    "total_price":  "orders.total_price",
    "created_at":   "orders.created_at",
}

orderBy, err := gopager.ParseSort(sortParams, columnMapping)
```

### 3. Error Handling and Validation

Comprehensive error handling for various scenarios:

- Invalid cursor tokens
- Malformed sort parameters
- Cursor/ordering mismatches
- Invalid pagination parameters

### 4. Custom Business Rule Validation

Demonstrates how to implement custom validation:

```go
// Check if cursor is too old (business rule)
if time.Since(createdAt) > 7*24*time.Hour {
    return errors.New("cursor is too old")
}
```

### 5. Performance Optimization

Best practices for production performance:

- Creating appropriate database indexes
- Using lookahead efficiently
- Avoiding unlimited queries in production
- Optimizing query patterns

## Key Production Patterns

### Database Indexes

Always create indexes on columns used for:
- Sorting (ORDER BY)
- Filtering (WHERE clauses)
- Cursor generation

```sql
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_orders_user_id ON orders(user_id);
```

### Error Handling Strategy

1. **Validate input early** - Check parameters before processing
2. **Use specific error messages** - Help developers debug issues
3. **Log errors appropriately** - Don't expose internal details
4. **Return appropriate HTTP status codes** - Follow REST conventions

### Security Considerations

1. **Column mapping** - Prevent SQL injection through column names
2. **Input validation** - Validate all user inputs
3. **Cursor validation** - Check cursor age and validity
4. **Rate limiting** - Implement rate limiting for pagination endpoints

### Performance Best Practices

1. **Use DefaultCursor for large datasets** - Better performance than OFFSET
2. **Limit page sizes** - Prevent excessive data transfer
3. **Use lookahead sparingly** - Only when you need to know if there are more pages
4. **Monitor query performance** - Use database query analysis tools
5. **Consider caching** - Cache frequently accessed data

## Common Pitfalls to Avoid

1. **Forgetting unique columns in sorting** - Always include a unique column
2. **Not handling empty result sets** - Always check for empty results
3. **Ignoring cursor validation** - Validate cursors before use
4. **Using unlimited queries** - Set reasonable limits
5. **Not testing edge cases** - Test with various data scenarios

## Monitoring and Debugging

### Query Performance

Monitor these metrics:
- Query execution time
- Number of rows scanned
- Index usage
- Memory consumption

### Pagination Metrics

Track these pagination-specific metrics:
- Average page size
- Cursor generation time
- Error rates by error type
- Most common sort patterns

### Debugging Tips

1. **Log generated SQL** - Use GORM's debug mode
2. **Validate cursors** - Check cursor contents
3. **Test with edge cases** - Empty results, single results, etc.
4. **Monitor memory usage** - Large result sets can cause issues

