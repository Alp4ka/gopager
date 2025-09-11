# Basic Usage Example

This example demonstrates the basic usage of GoPager with DefaultCursor for cursor-based pagination.

## Features Demonstrated

- Basic pagination with single column sorting
- Multi-column sorting
- Lookahead pagination to detect if there are more pages
- Unlimited results pagination
- Next page cursor generation

## Running the Example

```bash
cd examples/basic
go run main.go
```

## What You'll See

1. **Basic Pagination**: Shows users sorted by ID with a limit of 3
2. **Multi-column Sorting**: Shows users sorted by age (DESC) then name (ASC)
3. **Lookahead Pagination**: Demonstrates how to detect if there are more pages
4. **Unlimited Results**: Shows all users without pagination limits

## Key Concepts

- **DefaultCursor**: Uses complex filtering conditions for precise pagination
- **Getters**: Functions that extract values from your structs for cursor generation
- **Lookahead**: Fetches one extra record to determine if there are more pages
- **Multi-column Sorting**: Supports sorting by multiple columns with different directions

