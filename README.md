# WowSQL Go SDK

Official Go client for WowSQL. All data operations communicate directly with PostgREST over HTTPS. No legacy API layers.

## Requirements

- Go 1.19 or later

## Installation

```bash
go get github.com/wowsql/sdk-go
```

## Quick Start

```go
package main

import (
    "fmt"
    wowsql "github.com/wowsql/sdk-go"
)

func main() {
    client := wowsql.NewClient("myproject", "wowsql_anon_...")

    result, err := client.Table("users").Get()
    if err != nil {
        panic(err)
    }
    fmt.Println(result.Data)
}
```

## Configuration

```go
client := wowsql.NewClient(
    "myproject",
    "wowsql_anon_...",
    wowsql.WithBaseDomain("wowsqlconnect.com"), // default
    wowsql.WithSecure(true),                    // default: true
    wowsql.WithTimeout(30 * time.Second),       // default: 30s
    wowsql.WithVerifySSL(true),                 // default: true
)
```

### projectURL formats

```go
wowsql.NewClient("myproject", apiKey)                                    // slug
wowsql.NewClient("myproject.wowsqlconnect.com", apiKey)                 // domain
wowsql.NewClient("https://myproject.wowsqlconnect.com", apiKey)         // full URL
```

## API Keys

| Key Type | Prefix | Usage |
|----------|--------|-------|
| Anonymous | `wowsql_anon_...` | Client-side, respects Row-Level Security |
| Service Role | `wowsql_service_...` | Server-side, bypasses RLS |

## Database Operations

### Querying Records

```go
// Get all records
result, err := client.Table("products").Get()
// result.Data  []map[string]interface{}
// result.Total int (from Content-Range header)

// Select specific columns
result, err := client.Table("products").
    Select("id", "name", "price").
    Get()

// Filter with equality
result, err := client.Table("products").
    Eq("category", "electronics").
    Get()

// Multiple filters (ANDed)
result, err := client.Table("products").
    Gte("price", 100).
    Lte("price", 500).
    Eq("in_stock", true).
    Get()

// OR filter
result, err := client.Table("products").
    Eq("status", "active").
    Or("status", wowsql.OpEq, "featured").
    Get()
```

### Filter Operators

| Constant | Description |
|----------|-------------|
| `OpEq` | Equal (`col=eq.val`) |
| `OpNeq` | Not equal |
| `OpGt` | Greater than |
| `OpGte` | Greater or equal |
| `OpLt` | Less than |
| `OpLte` | Less or equal |
| `OpLike` | LIKE pattern (case-sensitive) |
| `OpILike` | ILIKE pattern (case-insensitive) |
| `OpIsNull` | IS NULL |
| `OpIsNotNull` | IS NOT NULL |
| `OpIn` | IN list |
| `OpNotIn` | NOT IN list |
| `OpBetween` | BETWEEN range |
| `OpNotBetween` | NOT BETWEEN |

### Ordering and Pagination

```go
// Order ascending
result, err := client.Table("products").
    OrderBy("price", wowsql.SortAsc).
    Get()

// Order descending
result, err := client.Table("products").
    OrderBy("created_at", wowsql.SortDesc).
    Get()

// Limit and offset
result, err := client.Table("products").
    Limit(20).
    Offset(40).
    Get()

// Paginate
page, err := client.Table("products").Paginate(2, 20)
// page.Data, page.Page, page.PerPage, page.Total, page.TotalPages
```

### Single Record Fetch

```go
// By primary key
user, err := client.Table("users").GetByID("uuid-here")

// First matching record
user, err := client.Table("users").
    Eq("email", "user@example.com").
    First()

// Exactly one record
user, err := client.Table("users").
    Eq("email", "user@example.com").
    Single()
```

### Creating Records

```go
// Single record
resp, err := client.Table("users").Create(map[string]interface{}{
    "email": "new@example.com",
    "name":  "New User",
})
fmt.Println(resp.ID)

// Insert (alias for Create)
resp, err := client.Table("users").Insert(data)

// Bulk insert
results, err := client.Table("users").BulkInsert([]map[string]interface{}{
    {"email": "a@example.com"},
    {"email": "b@example.com"},
})
```

### Updating Records

```go
// Update by primary key
resp, err := client.Table("users").Update("user-uuid", map[string]interface{}{
    "name": "Updated Name",
})
fmt.Println(resp.AffectedRows)
```

### Deleting Records

```go
// Delete by primary key
resp, err := client.Table("users").Delete("user-uuid")
fmt.Println(resp.AffectedRows)
```

### Upsert

```go
resp, err := client.Table("users").Upsert(map[string]interface{}{
    "id":   "existing-uuid",
    "name": "Updated",
}, "id")
```

### Aggregates

```go
// Count
total, err := client.Table("orders").Eq("status", "active").Count()

// Sum
revenue, err := client.Table("orders").Eq("status", "paid").Sum("amount")

// Average
avgPrice, err := client.Table("products").Eq("category", "electronics").Avg("price")
```

### Group By

```go
result, err := client.Table("orders").
    Select("status", "count(*)").
    GroupBy("status").
    Get()
```

## Authentication

```go
auth := wowsql.NewAuthClient("myproject", "wowsql_anon_...")
```

### Email / Password

```go
// Sign up
resp, err := auth.SignUp(wowsql.SignUpRequest{
    Email:    "user@example.com",
    Password: "secure-password",
})
session := resp.Session

// Sign in
resp, err := auth.SignIn(wowsql.SignInRequest{
    Email:    "user@example.com",
    Password: "secure-password",
})

// Get current user
user, err := auth.GetUser(session.AccessToken)

// Sign out
err = auth.Logout(session.AccessToken)
```

### OAuth

```go
// Step 1 — get redirect URL
result, err := auth.GetOAuthAuthorizationURL("google",
    wowsql.WithFrontendRedirectURI("https://myapp.com/auth/callback"))
// Redirect user to result.AuthorizationURL

// Step 2 — exchange code for tokens
resp, err := auth.ExchangeOAuthCallback("google", code, "")
session := resp.Session
```

### Token Management

```go
// Refresh tokens
resp, err := auth.RefreshToken(session.RefreshToken)

// Password reset
err = auth.ForgotPassword("user@example.com")
err = auth.ResetPassword(token, "new-password")
```

## File Storage

```go
storage := wowsql.NewStorageClient("myproject", "wowsql_anon_...")

// Create bucket
bucket, err := storage.CreateBucket("avatars",
    wowsql.BucketPublic(true))

// Upload file
data, _ := os.ReadFile("photo.jpg")
result, err := storage.Upload("avatars", data,
    wowsql.UploadPath("users/profile.jpg"))

// Get public URL
url := storage.GetPublicURL("avatars", "users/profile.jpg")

// List files
files, err := storage.ListFiles("avatars",
    wowsql.ListFilesPrefix("users/"))

// Download
fileData, err := storage.Download("avatars", "users/profile.jpg")

// Delete
err = storage.DeleteFile("avatars", "users/profile.jpg")
```

## Schema Management

Requires a service role key.

```go
schema := wowsql.NewSchemaClient("myproject", "wowsql_service_...")

// Create table
err := schema.CreateTable("products", []wowsql.ColumnDefinition{
    {Name: "id",         Type: "UUID",         AutoIncrement: true},
    {Name: "name",       Type: "VARCHAR(255)", Nullable: false},
    {Name: "price",      Type: "DECIMAL(10,2)"},
    {Name: "created_at", Type: "TIMESTAMPTZ",  Default: "CURRENT_TIMESTAMP"},
})

// Add column
err = schema.AddColumn("products", "sku", "VARCHAR(100)", true, "")

// Drop column
err = schema.DropColumn("products", "old_column")

// List tables
tables, err := schema.ListTables()

// Get table schema
info, err := schema.GetTableSchema("products")

// Execute raw SQL
err = schema.ExecuteSQL("CREATE INDEX ON products(name)")

// Drop table
err = schema.DropTable("products", false)
```

## Error Handling

```go
result, err := client.Table("orders").Eq("id", "uuid").Get()
if err != nil {
    switch e := err.(type) {
    case *wowsql.WOWSQLError:
        fmt.Printf("Error %d: %s\n", e.StatusCode, e.Message)
    case *wowsql.AuthenticationError:
        fmt.Println("Authentication failed")
    case *wowsql.NotFoundError:
        fmt.Println("Record not found")
    case *wowsql.NetworkError:
        fmt.Printf("Network error: %v\n", e.Err)
    }
}
```

## Architecture

All requests are sent directly to the PostgREST endpoint (`/rest/v1`). SDK filter expressions are translated to PostgREST query parameters. Total counts are read from the `Content-Range` response header.

| Operation | HTTP Method | PostgREST Path |
|-----------|-------------|----------------|
| GET | `GET` | `/{table}?col=op.val` |
| CREATE | `POST` | `/{table}` with `Prefer: return=representation` |
| UPDATE | `PATCH` | `/{table}?id=eq.{id}` |
| DELETE | `DELETE` | `/{table}?id=eq.{id}` |
| UPSERT | `POST` | `/{table}` with `Prefer: resolution=merge-duplicates` |
| COUNT | `GET` | `/{table}?limit=0` with `Prefer: count=exact` |

## License

MIT
