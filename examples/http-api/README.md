# HTTP API Example

This example demonstrates how to integrate GoPager with HTTP APIs, showing both GET and POST request handling with cursor-based pagination.

## Features Demonstrated

- RESTful API with cursor-based pagination
- Support for both GET and POST requests
- Custom sorting parameters
- Error handling and validation
- HTML documentation endpoint
- JSON request/response format

## Running the Example

```bash
cd examples/http-api
go run main.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### GET /users
Get paginated users with query parameters.

**Query Parameters:**
- `limit` (int): Number of items per page (default: 10)
- `startToken` (string): Cursor token for pagination
- `sort` (string): Sort specification (e.g., "age desc,name asc")

**Example:**
```bash
curl "http://localhost:8080/users?limit=5&sort=age desc"
```

### POST /users
Get paginated users with JSON request body.

**Request Body:**
```json
{
  "limit": 3,
  "startToken": "base64_encoded_cursor",
  "sort": ["age desc", "name asc"]
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"limit":3,"sort":["age desc"]}'
```

### GET /users/count
Get total number of users.

**Example:**
```bash
curl http://localhost:8080/users/count
```

### GET /
HTML documentation page with API reference.

## Response Format

All paginated endpoints return responses in this format:

```json
{
  "items": [
    {
      "id": 1,
      "name": "Alice Johnson",
      "email": "alice@example.com",
      "age": 25,
      "city": "New York",
      "created_at": "2024-01-01T12:00:00Z"
    }
  ],
  "nextPageToken": "base64_encoded_cursor",
  "hasMore": true,
  "total": 100
}
```

## Error Handling

The API returns appropriate HTTP status codes and error messages:

- `400 Bad Request`: Invalid parameters or malformed JSON
- `405 Method Not Allowed`: Unsupported HTTP method
- `500 Internal Server Error`: Database or server errors

Error response format:
```json
{
  "error": "Bad Request",
  "message": "Invalid sort parameter: unknown column 'invalid_field'"
}
```

## Testing the API

1. **Start the server:**
   ```bash
   go run main.go
   ```

2. **Get first page:**
   ```bash
   curl "http://localhost:8080/users?limit=3"
   ```

3. **Use the nextPageToken from the response:**
   ```bash
   curl "http://localhost:8080/users?limit=3&startToken=<token_from_previous_response>"
   ```

4. **Test custom sorting:**
   ```bash
   curl "http://localhost:8080/users?limit=3&sort=age desc,name asc"
   ```

5. **Test POST request:**
   ```bash
   curl -X POST http://localhost:8080/users \
     -H "Content-Type: application/json" \
     -d '{"limit":2,"sort":["city asc","age desc"]}'
   ```

## Key Implementation Details

- **Column Mapping**: Maps external field names to database columns
- **Sort Parsing**: Parses sort strings into OrderBy structures
- **Lookahead**: Uses lookahead to detect if there are more pages
- **Error Handling**: Comprehensive error handling with proper HTTP status codes
- **Flexibility**: Supports both query parameters and JSON body requests

