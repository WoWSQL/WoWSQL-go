# WowSQL Go SDK

Official Go SDK for [WowSQL](https://wowsql.com) — MySQL Backend-as-a-Service with S3 Storage and built-in Auth.

[![Go Reference](https://pkg.go.dev/badge/github.com/wowsql/wowsql-go.svg)](https://pkg.go.dev/github.com/wowsql/wowsql-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/wowsql/wowsql-go)](https://goreportcard.com/report/github.com/wowsql/wowsql-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Installation

```bash
go get github.com/wowsql/wowsql-go/wowsql
```

Requires **Go 1.21+**.

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/wowsql/wowsql-go/wowsql"
)

func main() {
    client := wowsql.NewClient(
        "https://your-project.wowsql.com",
        "your-api-key",
    )
    defer client.Close()

    users, err := client.Table("users").
        Select("id", "name", "email").
        Eq("status", "active").
        OrderBy("created_at", wowsql.SortDesc).
        Limit(10).
        Execute()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d users\n", users.Count)
    for _, u := range users.Data {
        fmt.Printf("  %v — %v\n", u["name"], u["email"])
    }
}
```

## Features

### Database
- Full CRUD operations (Create, Read, Update, Delete)
- Fluent query builder with method chaining
- Advanced filtering (`Eq`, `Neq`, `Gt`, `Gte`, `Lt`, `Lte`, `Like`, `IsNull`, `IsNotNull`, `In`, `NotIn`, `Between`, `NotBetween`)
- Logical operators (AND, OR)
- `GROUP BY` with aggregate functions (COUNT, SUM, AVG, MAX, MIN)
- `HAVING` clause for filtering aggregated results
- Multiple `ORDER BY` columns
- Date/time functions in SELECT and filters
- Pagination (`Limit`, `Offset`, `Paginate`)
- Bulk insert and upsert
- Raw SQL queries
- Table schema introspection
- Health check endpoint

### Authentication
- Email/password sign-up and sign-in
- OAuth providers (GitHub, Google, etc.)
- Password reset and forgot-password flows
- OTP (one-time password) and magic link authentication
- Email verification
- Session management with pluggable `TokenStorage`
- User profile updates

### Storage
- Bucket management (create, list, get, update, delete)
- File upload from `io.Reader` or local file path
- File listing with metadata
- File download and public URL generation
- Storage quota and stats
- Configurable MIME type and file-size limits

### Schema Management
- Create, alter, and drop tables programmatically
- Add, drop, rename, and modify columns
- Create indexes (unique, custom name, BTREE/HASH)
- List tables and inspect schemas
- Execute raw DDL

---

## Database Operations

### Initializing the Client

```go
package main

import (
    "time"

    "github.com/wowsql/wowsql-go/wowsql"
)

func main() {
    client := wowsql.NewClient(
        "https://your-project.wowsql.com",
        "your-api-key",
    )
    defer client.Close()

    // With functional options
    client = wowsql.NewClient(
        "https://your-project.wowsql.com",
        "your-api-key",
        wowsql.WithTimeout(60*time.Second),
        wowsql.WithSecure(true),
        wowsql.WithVerifySSL(true),
        wowsql.WithBaseDomain("custom-domain.com"),
    )
    defer client.Close()
}
```

### Select Queries

```go
// Select all columns
all, err := client.Table("users").Select("*").Execute()

// Select specific columns
users, err := client.Table("users").
    Select("id", "name", "email").
    Execute()

// With filters
active, err := client.Table("users").
    Select("*").
    Eq("status", "active").
    Gt("age", "18").
    Execute()

// With ordering and limit
recent, err := client.Table("users").
    Select("*").
    OrderBy("created_at", wowsql.SortDesc).
    Limit(10).
    Execute()

// With pagination via offset
page1, err := client.Table("users").
    Select("*").
    Limit(20).
    Offset(0).
    Execute()

page2, err := client.Table("users").
    Select("*").
    Limit(20).
    Offset(20).
    Execute()

// Pattern matching
gmailUsers, err := client.Table("users").
    Select("*").
    Like("email", "%@gmail.com").
    Execute()
```

### Terminal Methods

The query builder provides several terminal methods for fetching data:

```go
// Execute — returns full QueryResponse with Data and Count
resp, err := client.Table("users").Select("*").Execute()
fmt.Println(resp.Data)  // []map[string]interface{}
fmt.Println(resp.Count) // int

// Get — same as Execute, returns *QueryResponse
resp, err := client.Table("users").Select("*").Get()

// First — returns the first matching row or nil
user, err := client.Table("users").
    Eq("email", "john@example.com").
    First()
if user != nil {
    fmt.Printf("Found: %v\n", user["name"])
}

// Single — returns exactly one row; errors if zero or more than one
user, err := client.Table("users").
    Eq("id", "42").
    Single()

// Count — returns only the row count
n, err := client.Table("users").
    Eq("status", "active").
    Count()
fmt.Printf("%d active users\n", n)

// Paginate — returns a PaginatedResponse with page metadata
page, err := client.Table("users").
    Select("*").
    Paginate(1, 25) // page 1, 25 per page
fmt.Printf("Page %d of %d (total: %d)\n", page.Page, page.TotalPages, page.Total)
for _, row := range page.Data {
    fmt.Println(row["name"])
}
```

### Get by ID

```go
user, err := client.Table("users").GetByID("42")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("User: %v\n", user["name"])
```

### Insert Data

```go
// Insert a single record (Insert and Create are aliases)
result, err := client.Table("users").Insert(map[string]interface{}{
    "name":   "John Doe",
    "email":  "john@example.com",
    "age":    30,
    "status": "active",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Created user with ID: %v\n", result.ID)

// Create is an alias for Insert
result, err = client.Table("users").Create(map[string]interface{}{
    "name":  "Jane Smith",
    "email": "jane@example.com",
})
```

### Bulk Insert

```go
records := []map[string]interface{}{
    {"name": "Alice", "email": "alice@example.com", "age": 28},
    {"name": "Bob", "email": "bob@example.com", "age": 35},
    {"name": "Charlie", "email": "charlie@example.com", "age": 22},
}

results, err := client.Table("users").BulkInsert(records)
if err != nil {
    log.Fatal(err)
}

for _, r := range results {
    fmt.Printf("Inserted ID: %v\n", r.ID)
}
```

### Upsert

```go
row, err := client.Table("users").Upsert(
    map[string]interface{}{
        "email":  "john@example.com",
        "name":   "John Updated",
        "status": "active",
    },
    "email", // conflict column
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Upserted row: %v\n", row)
```

### Update Data

```go
// Update a record by ID
result, err := client.Table("users").Update("1", map[string]interface{}{
    "name": "Jane Smith",
    "age":  26,
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Updated %d row(s)\n", result.AffectedRows)

// Update with query builder conditions
resp, err := client.Table("users").
    Eq("status", "inactive").
    Update(map[string]interface{}{
        "status": "archived",
    })
```

### Delete Data

```go
// Delete a record by ID
result, err := client.Table("users").Delete("1")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Deleted %d row(s)\n", result.AffectedRows)

// Delete with query builder conditions
resp, err := client.Table("users").
    Eq("status", "archived").
    Lt("updated_at", "2024-01-01").
    Delete()
```

### Raw SQL Queries

```go
rows, err := client.Query("SELECT COUNT(*) as total FROM users WHERE age > 18")
if err != nil {
    log.Fatal(err)
}
if len(rows) > 0 {
    fmt.Printf("Adult users: %v\n", rows[0]["total"])
}
```

### Utility Methods

```go
// List all tables
tables, err := client.ListTables()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Tables: %v\n", tables)

// Get table schema
schema, err := client.GetTableSchema("users")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Table: %s (%d columns)\n", schema.Name, len(schema.Columns))
for _, col := range schema.Columns {
    fmt.Printf("  %-20s %s\n", col.Name, col.Type)
}

// Health check
health, err := client.Health()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("API status: %v\n", health["status"])
```

---

## Advanced Query Features

### GROUP BY and Aggregates

GROUP BY supports both simple column names and SQL expressions with functions. All expressions are validated server-side for security.

#### Basic GROUP BY

```go
result, err := client.Table("products").
    Select("category", "COUNT(*) as count", "AVG(price) as avg_price").
    GroupBy("category").
    Execute()

// Multiple columns
result, err = client.Table("sales").
    Select("region", "category", "SUM(amount) as total").
    GroupBy("region", "category").
    Execute()
```

#### GROUP BY with Date/Time Functions

```go
// Daily revenue
result, err := client.Table("orders").
    Select("DATE(created_at) as date", "COUNT(*) as orders", "SUM(total) as revenue").
    GroupBy("DATE(created_at)").
    OrderBy("date", wowsql.SortDesc).
    Execute()

// Monthly breakdown
result, err = client.Table("orders").
    Select(
        "YEAR(created_at) as year",
        "MONTH(created_at) as month",
        "SUM(total) as revenue",
    ).
    GroupBy("YEAR(created_at)", "MONTH(created_at)").
    Execute()

// Weekly stats
result, err = client.Table("orders").
    Select("WEEK(created_at) as week", "COUNT(*) as orders").
    GroupBy("WEEK(created_at)").
    Execute()

// Quarterly report
result, err = client.Table("orders").
    Select("QUARTER(created_at) as quarter", "SUM(total) as revenue").
    GroupBy("QUARTER(created_at)").
    Execute()
```

#### GROUP BY with String Functions

```go
result, err := client.Table("users").
    Select("LEFT(name, 1) as initial", "COUNT(*) as count").
    GroupBy("LEFT(name, 1)").
    Execute()

result, err = client.Table("products").
    Select("UPPER(category) as cat", "COUNT(*) as count").
    GroupBy("UPPER(category)").
    Execute()
```

#### GROUP BY with Mathematical Functions

```go
result, err := client.Table("products").
    Select("ROUND(price, -1) as price_range", "COUNT(*) as count").
    GroupBy("ROUND(price, -1)").
    Execute()

result, err = client.Table("products").
    Select("FLOOR(price / 10) * 10 as tier", "COUNT(*) as count").
    GroupBy("FLOOR(price / 10) * 10").
    Execute()
```

#### Supported Functions in GROUP BY

**Date/Time:** `DATE()`, `YEAR()`, `MONTH()`, `DAY()`, `DAYOFMONTH()`, `DAYOFWEEK()`, `DAYOFYEAR()`, `WEEK()`, `QUARTER()`, `HOUR()`, `MINUTE()`, `SECOND()`, `DATE_FORMAT()`, `TIME()`, `DATE_ADD()`, `DATE_SUB()`, `DATEDIFF()`, `TIMEDIFF()`, `TIMESTAMPDIFF()`, `NOW()`, `CURRENT_TIMESTAMP()`, `CURDATE()`, `CURRENT_DATE()`, `CURTIME()`, `CURRENT_TIME()`, `UNIX_TIMESTAMP()`

**String:** `CONCAT()`, `CONCAT_WS()`, `SUBSTRING()`, `SUBSTR()`, `LEFT()`, `RIGHT()`, `LENGTH()`, `CHAR_LENGTH()`, `UPPER()`, `LOWER()`, `TRIM()`, `LTRIM()`, `RTRIM()`, `REPLACE()`, `LOCATE()`, `POSITION()`

**Mathematical:** `ABS()`, `ROUND()`, `CEIL()`, `CEILING()`, `FLOOR()`, `POW()`, `POWER()`, `SQRT()`, `MOD()`, `RAND()`

### HAVING Clause

HAVING filters aggregated results after GROUP BY. It supports aggregate functions and comparison operators.

```go
// Basic HAVING
result, err := client.Table("products").
    Select("category", "COUNT(*) as count").
    GroupBy("category").
    Having("COUNT(*)", "gt", "10").
    Execute()

// Multiple HAVING conditions (AND logic)
result, err = client.Table("orders").
    Select("DATE(created_at) as date", "SUM(total) as revenue").
    GroupBy("DATE(created_at)").
    Having("SUM(total)", "gt", "1000").
    Having("COUNT(*)", "gte", "5").
    Execute()

// HAVING with different aggregates
result, err = client.Table("products").
    Select("category", "AVG(price) as avg_price", "COUNT(*) as count").
    GroupBy("category").
    Having("AVG(price)", "gt", "100").
    Having("COUNT(*)", "gte", "5").
    Execute()

// HAVING with MAX/MIN
result, err = client.Table("products").
    Select("category", "MAX(price) as max_price", "MIN(price) as min_price").
    GroupBy("category").
    Having("MAX(price)", "gt", "500").
    Execute()

// Top spenders
result, err = client.Table("orders").
    Select("customer_id", "SUM(total) as total_spent").
    GroupBy("customer_id").
    Having("SUM(total)", "gt", "1000").
    OrderBy("total_spent", wowsql.SortDesc).
    Execute()
```

**Supported HAVING Operators:** `eq`, `neq`, `gt`, `gte`, `lt`, `lte`

**Supported Aggregate Functions:** `COUNT(*)`, `COUNT(column)`, `SUM()`, `AVG()`, `MAX()`, `MIN()`, `GROUP_CONCAT()`, `STDDEV()`, `VARIANCE()`

### Multiple ORDER BY

```go
result, err := client.Table("products").
    Select("*").
    OrderBy("category", wowsql.SortAsc).
    Order("price", wowsql.SortDesc).
    Order("created_at", wowsql.SortDesc).
    Execute()
```

### Date/Time Functions in Filters

```go
// Last 7 days
result, err := client.Table("orders").
    Select("*").
    Filter("created_at", wowsql.OpGte, "DATE_SUB(NOW(), INTERVAL 7 DAY)").
    Execute()

// Filter by year/month
result, err = client.Table("orders").
    Select("*").
    Filter("YEAR(created_at)", wowsql.OpEq, "2024").
    Filter("MONTH(created_at)", wowsql.OpEq, "6").
    Execute()
```

### OR Conditions

```go
result, err := client.Table("products").
    Eq("category", "electronics").
    Or("category", "eq", "books").
    Execute()

result, err = client.Table("users").
    Filter("status", wowsql.OpEq, "active").
    Or("role", "eq", "admin").
    Execute()
```

---

## Filter Operators

The SDK provides both shortcut methods and the generic `Filter` method.

### Shortcut Methods

```go
table := client.Table("users")

// Equal
table.Eq("status", "active")

// Not equal
table.Neq("status", "deleted")

// Greater than
table.Gt("age", "18")

// Greater than or equal
table.Gte("age", "18")

// Less than
table.Lt("age", "65")

// Less than or equal
table.Lte("age", "65")

// Pattern matching (SQL LIKE)
table.Like("email", "%@gmail.com")

// Is null
table.IsNull("deleted_at")

// Is not null
table.IsNotNull("email")
```

### Generic Filter Method

```go
// The Filter method accepts a FilterOperator constant and an optional logical operator
table.Filter("status", wowsql.OpEq, "active")
table.Filter("age", wowsql.OpGt, "18")
table.Filter("role", wowsql.OpNeq, "guest")
table.Filter("price", wowsql.OpLte, "99.99")
```

### Collection Operators

```go
// IN
client.Table("products").
    In("category", []interface{}{"electronics", "books", "clothing"}).
    Execute()

// NOT IN
client.Table("products").
    NotIn("status", []interface{}{"deleted", "archived"}).
    Execute()

// BETWEEN
client.Table("products").
    Between("price", 10, 100).
    Execute()

// NOT BETWEEN
client.Table("users").
    NotBetween("age", 18, 65).
    Execute()
```

### FilterOperator Constants

| Constant        | SQL Equivalent   |
|-----------------|------------------|
| `OpEq`          | `=`              |
| `OpNeq`         | `!=` / `<>`      |
| `OpGt`          | `>`              |
| `OpGte`         | `>=`             |
| `OpLt`          | `<`              |
| `OpLte`         | `<=`             |
| `OpLike`        | `LIKE`           |
| `OpIsNull`      | `IS NULL`        |
| `OpIsNotNull`   | `IS NOT NULL`    |
| `OpIn`          | `IN (...)`       |
| `OpNotIn`       | `NOT IN (...)`   |
| `OpBetween`     | `BETWEEN ... AND ...` |
| `OpNotBetween`  | `NOT BETWEEN ... AND ...` |

### SortDirection Constants

| Constant   | Value  |
|------------|--------|
| `SortAsc`  | `asc`  |
| `SortDesc` | `desc` |

---

## Authentication

### Initializing the Auth Client

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/wowsql/wowsql-go/wowsql"
)

func main() {
    auth := wowsql.NewAuthClient(
        "https://your-project.wowsql.com",
        "wowsql_anon_your-key",
    )
    defer auth.Close()

    // With functional options
    auth = wowsql.NewAuthClient(
        "https://your-project.wowsql.com",
        "wowsql_anon_your-key",
        wowsql.AuthWithTimeout(60*time.Second),
        wowsql.AuthWithSecure(true),
        wowsql.AuthWithVerifySSL(true),
        wowsql.AuthWithBaseDomain("custom-domain.com"),
        wowsql.AuthWithTokenStorage(&wowsql.MemoryTokenStorage{}),
    )
    defer auth.Close()
}
```

### Sign Up

```go
result, err := auth.SignUp("user@example.com", "SecurePassword123")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("User ID:       %s\n", result.User.ID)
fmt.Printf("Email:         %s\n", result.User.Email)
fmt.Printf("Access Token:  %s\n", result.Session.AccessToken)

// With optional metadata
result, err = auth.SignUp(
    "user@example.com",
    "SecurePassword123",
    wowsql.WithFullName("Jane Doe"),
    wowsql.WithUserMetadata(map[string]interface{}{
        "referrer":  "landing-page",
        "plan":      "free",
    }),
)
```

### Sign In

```go
result, err := auth.SignIn("user@example.com", "SecurePassword123")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Welcome back, %s\n", result.User.FullName)
fmt.Printf("Access Token: %s\n", result.Session.AccessToken)
fmt.Printf("Refresh Token: %s\n", result.Session.RefreshToken)
```

### Get Current User

```go
// Uses the token from the internal session store
user, err := auth.GetUser()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("ID:             %s\n", user.ID)
fmt.Printf("Email:          %s\n", user.Email)
fmt.Printf("Email Verified: %v\n", user.EmailVerified)
fmt.Printf("Full Name:      %s\n", user.FullName)

// Or pass an explicit token override
user, err = auth.GetUser("eyJhbGciOiJIUzI1NiIs...")
```

### Session Management

```go
// Persist session after sign-in
auth.SetSession(result.Session.AccessToken, result.Session.RefreshToken)

// Retrieve the current session
session := auth.GetSession()
if session != nil {
    fmt.Printf("Access:  %s\n", session.AccessToken)
    fmt.Printf("Refresh: %s\n", session.RefreshToken)
}

// Refresh an expired access token
newResult, err := auth.RefreshToken()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("New access token: %s\n", newResult.Session.AccessToken)

// Refresh with an explicit refresh token
newResult, err = auth.RefreshToken("explicit-refresh-token")

// Clear session
auth.ClearSession()
```

### Custom Token Storage

Implement the `TokenStorage` interface to persist tokens however you like (database, Redis, file, etc.):

```go
type TokenStorage interface {
    GetAccessToken() string
    SetAccessToken(token string)
    GetRefreshToken() string
    SetRefreshToken(token string)
}
```

The SDK ships with `MemoryTokenStorage` for in-process use:

```go
auth := wowsql.NewAuthClient(
    "https://your-project.wowsql.com",
    "wowsql_anon_your-key",
    wowsql.AuthWithTokenStorage(&wowsql.MemoryTokenStorage{}),
)
```

Example custom implementation backed by a file or database:

```go
type RedisTokenStorage struct {
    client *redis.Client
    prefix string
}

func (r *RedisTokenStorage) GetAccessToken() string {
    val, _ := r.client.Get(ctx, r.prefix+":access").Result()
    return val
}

func (r *RedisTokenStorage) SetAccessToken(token string) {
    r.client.Set(ctx, r.prefix+":access", token, 0)
}

func (r *RedisTokenStorage) GetRefreshToken() string {
    val, _ := r.client.Get(ctx, r.prefix+":refresh").Result()
    return val
}

func (r *RedisTokenStorage) SetRefreshToken(token string) {
    r.client.Set(ctx, r.prefix+":refresh", token, 0)
}
```

### OAuth Authentication

```go
// Step 1: Get the authorization URL
oauthResp, err := auth.GetOAuthAuthorizationURL("github", "https://app.example.com/auth/callback")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Redirect user to: %s\n", oauthResp["authorization_url"])

// Step 2: In your callback handler, exchange the code for tokens
result, err := auth.ExchangeOAuthCallback(
    "github",
    "authorization_code_from_query_param",
    "https://app.example.com/auth/callback",
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Logged in as: %s\n", result.User.Email)
fmt.Printf("Access token: %s\n", result.Session.AccessToken)
```

### Password Reset

```go
// Request a password reset email
err := auth.ForgotPassword("user@example.com")
if err != nil {
    log.Fatal(err)
}
fmt.Println("If that email exists, a reset link has been sent.")

// Reset password using the token from the email link
err = auth.ResetPassword("reset-token-from-email", "NewSecurePassword456")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Password reset successfully.")
```

### Change Password

```go
err := auth.ChangePassword("currentPassword", "newPassword123")
if err != nil {
    log.Fatal(err)
}

// With explicit token override
err = auth.ChangePassword("currentPassword", "newPassword123", "access-token-override")
```

### OTP (One-Time Password)

```go
// Send OTP to user's email
err := auth.SendOTP("user@example.com", "login")
if err != nil {
    log.Fatal(err)
}

// Verify OTP
err = auth.VerifyOTP("user@example.com", "123456", "login")
if err != nil {
    log.Fatal(err)
}

// OTP for password reset (pass new password as 4th arg)
err = auth.SendOTP("user@example.com", "password_reset")
err = auth.VerifyOTP("user@example.com", "654321", "password_reset", "NewPassword789")
```

### Magic Link

```go
err := auth.SendMagicLink("user@example.com", "login")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Magic link sent! Check your inbox.")
```

### Email Verification

```go
// Verify email with token from the verification link
err := auth.VerifyEmail("verification-token-from-email")
if err != nil {
    log.Fatal(err)
}

// Resend verification email
err = auth.ResendVerification("user@example.com")
if err != nil {
    log.Fatal(err)
}
```

### Update User Profile

```go
user, err := auth.UpdateUser(
    wowsql.UpdateFullName("Jane Doe-Smith"),
    wowsql.UpdateAvatarURL("https://example.com/avatar.jpg"),
    wowsql.UpdateUsername("janedoe"),
    wowsql.UpdateUserMetadata(map[string]interface{}{
        "theme":    "dark",
        "language": "en",
    }),
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Updated user: %s\n", user.FullName)
```

### Logout

```go
err := auth.Logout()
if err != nil {
    log.Fatal(err)
}

// With explicit token
err = auth.Logout("access-token-to-invalidate")
```

---

## Storage Operations

### Initializing the Storage Client

```go
package main

import (
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "github.com/wowsql/wowsql-go/wowsql"
)

func main() {
    storage := wowsql.NewStorageClient(
        "https://your-project.wowsql.com",
        "your-api-key",
    )
    defer storage.Close()

    // With functional options
    storage = wowsql.NewStorageClient(
        "https://your-project.wowsql.com",
        "your-api-key",
        wowsql.StorageWithTimeout(120*time.Second),
        wowsql.StorageWithSecure(true),
        wowsql.StorageWithVerifySSL(true),
    )
    defer storage.Close()
}
```

### Bucket Management

```go
// Create a bucket
bucket, err := storage.CreateBucket("avatars",
    wowsql.BucketPublic(true),
    wowsql.BucketFileSizeLimit(5*1024*1024), // 5 MB
    wowsql.BucketAllowedMimeTypes([]string{"image/png", "image/jpeg", "image/webp"}),
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Created bucket: %s\n", bucket.Name)

// List all buckets
buckets, err := storage.ListBuckets()
if err != nil {
    log.Fatal(err)
}
for _, b := range buckets {
    fmt.Printf("  %s (public: %v)\n", b.Name, b.Public)
}

// Get bucket details
bucket, err = storage.GetBucket("avatars")
fmt.Printf("Bucket: %s, Created: %s\n", bucket.Name, bucket.CreatedAt)

// Update bucket settings
bucket, err = storage.UpdateBucket("avatars",
    wowsql.BucketPublic(false),
    wowsql.BucketFileSizeLimit(10*1024*1024),
)

// Delete a bucket
err = storage.DeleteBucket("old-bucket")
```

### File Upload

```go
// Upload from an io.Reader
file, err := os.Open("photo.jpg")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

result, err := storage.Upload("avatars", file,
    wowsql.UploadPath("users/42"),
    wowsql.UploadFileName("profile.jpg"),
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Uploaded: %s\n", result.Path)

// Upload from a string/bytes via strings.Reader
data := strings.NewReader(`{"config": true}`)
result, err = storage.Upload("configs", data,
    wowsql.UploadPath("app"),
    wowsql.UploadFileName("settings.json"),
)

// Upload from a local file path (convenience method)
result, err = storage.UploadFromPath(
    "documents/report.pdf", // local path
    "reports",              // bucket name
    "2024/q1/report.pdf",  // remote path (optional)
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Uploaded to: %s\n", result.Path)
```

### File Listing

```go
// List all files in a bucket
files, err := storage.ListFiles("avatars")
if err != nil {
    log.Fatal(err)
}
for _, f := range files {
    fmt.Printf("  %s  %.2f MB  %s\n", f.Key, f.SizeMB, f.LastModified)
}

// List with options (prefix filter)
files, err = storage.ListFiles("reports",
    wowsql.ListFilesPrefix("2024/"),
)
```

### File Download

```go
// Download file content
data, err := storage.Download("reports", "2024/q1/report.pdf")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Downloaded %d bytes\n", len(data))

// Download directly to a local file
err = storage.DownloadToFile("reports", "2024/q1/report.pdf", "./local-report.pdf")
if err != nil {
    log.Fatal(err)
}
```

### Public URL and Deletion

```go
// Get a public URL for a file
url := storage.GetPublicURL("avatars", "users/42/profile.jpg")
fmt.Printf("Public URL: %s\n", url)

// Delete a file
err := storage.DeleteFile("avatars", "users/42/old-photo.jpg")
if err != nil {
    log.Fatal(err)
}
```

### Quota and Stats

```go
// Get storage stats
stats, err := storage.GetStats()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Total files: %d\n", stats["total_files"])
fmt.Printf("Total size:  %v\n", stats["total_size"])

// Get storage quota
quota, err := storage.GetQuota()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Used:      %.2f GB / %.2f GB\n", quota.StorageUsedGB, quota.StorageQuotaGB)
fmt.Printf("Available: %.2f GB\n", quota.StorageAvailableGB)
fmt.Printf("Usage:     %.1f%%\n", quota.UsagePercentage)

// Check quota before uploading
fileInfo, _ := os.Stat("large-file.zip")
if quota.StorageAvailableBytes < fileInfo.Size() {
    fmt.Println("Not enough storage! Upgrade your plan or delete old files.")
} else {
    storage.UploadFromPath("large-file.zip", "backups")
}
```

---

## Schema Management

Schema operations require a **Service Role Key**. Anonymous keys will receive a 403 Forbidden error.

### Initializing the Schema Client

```go
package main

import (
    "fmt"
    "log"
    "os"
    "time"

    "github.com/wowsql/wowsql-go/wowsql"
)

func main() {
    schema := wowsql.NewSchemaClient(
        "https://your-project.wowsql.com",
        os.Getenv("WOWSQL_SERVICE_KEY"),
    )
    defer schema.Close()

    // With functional options
    schema = wowsql.NewSchemaClient(
        "https://your-project.wowsql.com",
        os.Getenv("WOWSQL_SERVICE_KEY"),
        wowsql.SchemaWithTimeout(30*time.Second),
        wowsql.SchemaWithSecure(true),
    )
    defer schema.Close()
}
```

### Create Table

```go
err := schema.CreateTable("products",
    []wowsql.ColumnDefinition{
        {Name: "id", Type: "INT", AutoIncrement: true, NotNull: true},
        {Name: "name", Type: "VARCHAR(255)", NotNull: true},
        {Name: "price", Type: "DECIMAL(10,2)", NotNull: true},
        {Name: "category", Type: "VARCHAR(100)"},
        {Name: "in_stock", Type: "BOOLEAN", Default: "true"},
        {Name: "created_at", Type: "TIMESTAMP", Default: "CURRENT_TIMESTAMP"},
    },
    wowsql.TablePrimaryKey("id"),
    wowsql.TableIndexes("category", "price"),
)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Table 'products' created.")
```

### Alter Table

```go
// Add a column
err := schema.AlterTable("products", "ADD",
    wowsql.AlterColumnName("stock_quantity"),
    wowsql.AlterColumnType("INT"),
    wowsql.AlterDefault("0"),
)

// Modify a column
err = schema.AlterTable("products", "MODIFY",
    wowsql.AlterColumnName("price"),
    wowsql.AlterColumnType("DECIMAL(12,2)"),
    wowsql.AlterNullable(false),
)

// Drop a column
err = schema.AlterTable("products", "DROP",
    wowsql.AlterColumnName("category"),
)

// Rename a column
err = schema.AlterTable("products", "RENAME",
    wowsql.AlterColumnName("name"),
    wowsql.AlterNewColumnName("product_name"),
)
```

### Convenience Methods

```go
// Add a column
err := schema.AddColumn("products", wowsql.ColumnDefinition{
    Name: "description", Type: "TEXT",
})

// Drop a column
err = schema.DropColumn("products", "old_field")

// Rename a column
err = schema.RenameColumn("products", "name", "product_name")

// Modify a column
err = schema.ModifyColumn("products", wowsql.ColumnDefinition{
    Name: "price", Type: "DECIMAL(12,4)", NotNull: true,
})
```

### Create Index

```go
err := schema.CreateIndex("products", []string{"category", "price"},
    wowsql.IndexName("idx_cat_price"),
    wowsql.IndexUnique(true),
    wowsql.IndexUsing("BTREE"),
)
if err != nil {
    log.Fatal(err)
}
```

### Drop Table

```go
// Simple drop
err := schema.DropTable("temp_table")

// Drop with CASCADE
err = schema.DropTable("products", true)
```

### Execute Raw DDL

```go
err := schema.ExecuteSQL(`
    CREATE INDEX idx_product_name ON products(product_name);
`)

err = schema.ExecuteSQL(`
    ALTER TABLE orders
    ADD CONSTRAINT fk_product
    FOREIGN KEY (product_id)
    REFERENCES products(id);
`)
```

### List Tables and Inspect Schemas

```go
tables, err := schema.ListTables()
if err != nil {
    log.Fatal(err)
}
for _, t := range tables {
    fmt.Println(t)
}

tableSchema, err := schema.GetTableSchema("products")
if err != nil {
    log.Fatal(err)
}
for _, col := range tableSchema.Columns {
    fmt.Printf("  %-20s %-15s nullable=%v\n", col.Name, col.Type, col.Nullable)
}
```

### Migration Script Example

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/wowsql/wowsql-go/wowsql"
)

func migrate(schema *wowsql.SchemaClient) error {
    err := schema.CreateTable("users",
        []wowsql.ColumnDefinition{
            {Name: "id", Type: "INT", AutoIncrement: true, NotNull: true},
            {Name: "email", Type: "VARCHAR(255)", NotNull: true, Unique: true},
            {Name: "name", Type: "VARCHAR(255)", NotNull: true},
            {Name: "created_at", Type: "TIMESTAMP", Default: "CURRENT_TIMESTAMP"},
        },
        wowsql.TablePrimaryKey("id"),
        wowsql.TableIndexes("email"),
    )
    if err != nil {
        return fmt.Errorf("create users: %w", err)
    }

    err = schema.CreateTable("posts",
        []wowsql.ColumnDefinition{
            {Name: "id", Type: "INT", AutoIncrement: true, NotNull: true},
            {Name: "user_id", Type: "INT", NotNull: true},
            {Name: "title", Type: "VARCHAR(255)", NotNull: true},
            {Name: "body", Type: "TEXT"},
            {Name: "published", Type: "BOOLEAN", Default: "false"},
            {Name: "created_at", Type: "TIMESTAMP", Default: "CURRENT_TIMESTAMP"},
        },
        wowsql.TablePrimaryKey("id"),
        wowsql.TableIndexes("user_id"),
    )
    if err != nil {
        return fmt.Errorf("create posts: %w", err)
    }

    err = schema.ExecuteSQL(`
        ALTER TABLE posts
        ADD CONSTRAINT fk_user
        FOREIGN KEY (user_id) REFERENCES users(id);
    `)
    if err != nil {
        return fmt.Errorf("add FK: %w", err)
    }

    return nil
}

func main() {
    schema := wowsql.NewSchemaClient(
        os.Getenv("WOWSQL_PROJECT_URL"),
        os.Getenv("WOWSQL_SERVICE_KEY"),
    )
    defer schema.Close()

    if err := migrate(schema); err != nil {
        log.Fatal(err)
    }
    fmt.Println("Migration completed successfully.")
}
```

---

## API Keys

WowSQL uses **unified authentication** — the same API keys work for database operations, authentication, storage, and schema management.

| Operation | Recommended Key | Alternative Key | Client |
|-----------|----------------|-----------------|--------|
| **Database** (CRUD) | Service Role Key (`wowsql_service_...`) | Anonymous Key (`wowsql_anon_...`) | `Client` |
| **Authentication** (sign-up, OAuth) | Anonymous Key (`wowsql_anon_...`) | Service Role Key (`wowsql_service_...`) | `AuthClient` |
| **Storage** (upload, download) | Service Role Key (`wowsql_service_...`) | Anonymous Key (`wowsql_anon_...`) | `StorageClient` |
| **Schema** (DDL) | Service Role Key (`wowsql_service_...`) | *(none)* | `SchemaClient` |

### Where to Find Your Keys

All keys are in: **WowSQL Dashboard > Settings > API Keys** (or **Authentication > PROJECT KEYS**).

1. **Anonymous Key** (`wowsql_anon_...`) — safe to expose in client-side code (browser, mobile).
2. **Service Role Key** (`wowsql_service_...`) — **never** expose in client code. Server-side only.

### Environment Variables

```go
package main

import (
    "os"

    "github.com/wowsql/wowsql-go/wowsql"
)

func main() {
    projectURL := os.Getenv("WOWSQL_PROJECT_URL")
    serviceKey := os.Getenv("WOWSQL_SERVICE_ROLE_KEY")
    anonKey    := os.Getenv("WOWSQL_ANON_KEY")

    // Server-side database client (full access)
    db := wowsql.NewClient(projectURL, serviceKey)
    defer db.Close()

    // Client-side auth (sign-up, login, OAuth)
    auth := wowsql.NewAuthClient(projectURL, anonKey)
    defer auth.Close()

    // Storage
    storage := wowsql.NewStorageClient(projectURL, serviceKey)
    defer storage.Close()

    // Schema management (requires service key)
    schema := wowsql.NewSchemaClient(projectURL, serviceKey)
    defer schema.Close()
}
```

### Security Best Practices

1. **Never expose** the Service Role Key in client-side code or public repositories.
2. **Use the Anonymous Key** for client-side auth flows and public database access.
3. **Store keys in environment variables** — never hard-code them.
4. **Rotate keys** immediately if compromised.
5. **Use service keys only in backend code** (API servers, CLI tools, migration scripts).

---

## Error Handling

The SDK uses idiomatic Go error handling. All errors implement the `error` interface. Use `errors.As` for typed error inspection.

### Error Types

| Error Type | When It Occurs |
|------------|---------------|
| `WOWSQLError` | Base error for all SDK errors |
| `AuthenticationError` | Invalid or expired API key / token |
| `NotFoundError` | Table or record not found |
| `RateLimitError` | API rate limit exceeded |
| `NetworkError` | Connection timeout or DNS failure |
| `StorageError` | General storage operation failure |
| `StorageLimitExceededError` | Upload exceeds available quota |
| `SchemaPermissionError` | Schema operation with non-service key |

### Database Error Handling

```go
import (
    "errors"
    "fmt"
    "log"

    "github.com/wowsql/wowsql-go/wowsql"
)

users, err := client.Table("users").Select("*").Execute()
if err != nil {
    var authErr     *wowsql.AuthenticationError
    var notFoundErr *wowsql.NotFoundError
    var rateErr     *wowsql.RateLimitError
    var netErr      *wowsql.NetworkError

    switch {
    case errors.As(err, &authErr):
        fmt.Printf("Auth error: %s\n", authErr.Message)
    case errors.As(err, &notFoundErr):
        fmt.Printf("Not found: %s\n", notFoundErr.Message)
    case errors.As(err, &rateErr):
        fmt.Printf("Rate limited — retry after a moment\n")
    case errors.As(err, &netErr):
        fmt.Printf("Network issue: %s\n", netErr.Error())
    default:
        log.Fatalf("Unexpected error: %v", err)
    }
}
```

### Storage Error Handling

```go
_, err := storage.UploadFromPath("huge-backup.zip", "backups")
if err != nil {
    var limitErr   *wowsql.StorageLimitExceededError
    var storageErr *wowsql.StorageError

    switch {
    case errors.As(err, &limitErr):
        fmt.Printf("Storage full: %s\n", limitErr.Message)
        fmt.Printf("  Required:  %d bytes\n", limitErr.RequiredBytes)
        fmt.Printf("  Available: %d bytes\n", limitErr.AvailableBytes)
    case errors.As(err, &storageErr):
        fmt.Printf("Storage error: %s\n", storageErr.Message)
    default:
        log.Fatal(err)
    }
}
```

### Schema Error Handling

```go
err := schema.CreateTable("test",
    []wowsql.ColumnDefinition{{Name: "id", Type: "INT"}},
)
if err != nil {
    var permErr *wowsql.SchemaPermissionError
    if errors.As(err, &permErr) {
        fmt.Printf("Permission denied: %s\n", permErr.Message)
        fmt.Println("Use a SERVICE ROLE KEY for schema operations.")
    } else {
        log.Fatal(err)
    }
}
```

### Auth Error Handling

```go
result, err := auth.SignIn("user@example.com", "wrong-password")
if err != nil {
    var authErr *wowsql.AuthenticationError
    if errors.As(err, &authErr) {
        fmt.Println("Invalid credentials.")
    } else {
        log.Fatal(err)
    }
}
```

---

## Configuration

All clients use the **functional options** pattern for clean, extensible configuration.

### Client Options

```go
client := wowsql.NewClient(
    "https://your-project.wowsql.com",
    "your-api-key",
    wowsql.WithTimeout(60*time.Second),    // HTTP timeout (default: 30s)
    wowsql.WithSecure(true),               // Use HTTPS (default: true)
    wowsql.WithVerifySSL(true),            // Verify TLS certs (default: true)
    wowsql.WithBaseDomain("custom.com"),   // Override base domain
)
```

### AuthClient Options

```go
auth := wowsql.NewAuthClient(
    "https://your-project.wowsql.com",
    "your-anon-key",
    wowsql.AuthWithTimeout(60*time.Second),
    wowsql.AuthWithSecure(true),
    wowsql.AuthWithVerifySSL(true),
    wowsql.AuthWithBaseDomain("custom.com"),
    wowsql.AuthWithTokenStorage(&wowsql.MemoryTokenStorage{}),
)
```

### StorageClient Options

```go
storage := wowsql.NewStorageClient(
    "https://your-project.wowsql.com",
    "your-api-key",
    wowsql.StorageWithTimeout(120*time.Second),
    wowsql.StorageWithSecure(true),
    wowsql.StorageWithVerifySSL(true),
)
```

### SchemaClient Options

```go
schema := wowsql.NewSchemaClient(
    "https://your-project.wowsql.com",
    "your-service-key",
    wowsql.SchemaWithTimeout(30*time.Second),
    wowsql.SchemaWithSecure(true),
    wowsql.SchemaWithVerifySSL(true),
)
```

---

## Examples

### Blog Application

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/wowsql/wowsql-go/wowsql"
)

func main() {
    client := wowsql.NewClient(
        os.Getenv("WOWSQL_PROJECT_URL"),
        os.Getenv("WOWSQL_SERVICE_ROLE_KEY"),
    )
    defer client.Close()

    // Create a post
    post, err := client.Table("posts").Insert(map[string]interface{}{
        "title":     "Getting Started with WowSQL",
        "content":   "WowSQL makes backend development effortless...",
        "author_id": 1,
        "published": true,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created post ID: %v\n", post.ID)

    // Fetch recent published posts
    resp, err := client.Table("posts").
        Select("id", "title", "created_at").
        Eq("published", "true").
        OrderBy("created_at", wowsql.SortDesc).
        Limit(10).
        Execute()
    if err != nil {
        log.Fatal(err)
    }
    for _, p := range resp.Data {
        fmt.Printf("  [%v] %v\n", p["id"], p["title"])
    }

    // Paginate comments for a post
    comments, err := client.Table("comments").
        Select("id", "body", "author_name", "created_at").
        Eq("post_id", fmt.Sprintf("%v", post.ID)).
        OrderBy("created_at", wowsql.SortAsc).
        Paginate(1, 20)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Comments: page %d/%d (%d total)\n",
        comments.Page, comments.TotalPages, comments.Total)
}
```

### File Upload with Avatar

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/wowsql/wowsql-go/wowsql"
)

func main() {
    client := wowsql.NewClient(
        os.Getenv("WOWSQL_PROJECT_URL"),
        os.Getenv("WOWSQL_SERVICE_ROLE_KEY"),
    )
    defer client.Close()

    storage := wowsql.NewStorageClient(
        os.Getenv("WOWSQL_PROJECT_URL"),
        os.Getenv("WOWSQL_SERVICE_ROLE_KEY"),
    )
    defer storage.Close()

    userID := "42"

    // Upload avatar
    remotePath := fmt.Sprintf("users/%s/avatar.jpg", userID)
    _, err := storage.UploadFromPath("avatar.jpg", "avatars", remotePath)
    if err != nil {
        log.Fatal(err)
    }

    // Save the public URL in the user record
    url := storage.GetPublicURL("avatars", remotePath)
    _, err = client.Table("users").Update(userID, map[string]interface{}{
        "avatar_url": url,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Avatar updated: %s\n", url)
}
```

### Analytics Dashboard Query

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/wowsql/wowsql-go/wowsql"
)

func main() {
    client := wowsql.NewClient(
        os.Getenv("WOWSQL_PROJECT_URL"),
        os.Getenv("WOWSQL_SERVICE_ROLE_KEY"),
    )
    defer client.Close()

    // Daily revenue for the last 30 days
    revenue, err := client.Table("orders").
        Select("DATE(created_at) as date", "COUNT(*) as orders", "SUM(total) as revenue").
        Filter("created_at", wowsql.OpGte, "DATE_SUB(NOW(), INTERVAL 30 DAY)").
        Eq("status", "completed").
        GroupBy("DATE(created_at)").
        Having("COUNT(*)", "gte", "1").
        OrderBy("date", wowsql.SortDesc).
        Execute()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Date           | Orders | Revenue")
    fmt.Println("---------------|--------|--------")
    for _, row := range revenue.Data {
        fmt.Printf("%-15v| %-7v| $%v\n", row["date"], row["orders"], row["revenue"])
    }

    // Top categories
    categories, err := client.Table("products").
        Select("category", "COUNT(*) as count", "AVG(price) as avg_price").
        GroupBy("category").
        Having("COUNT(*)", "gte", "5").
        OrderBy("count", wowsql.SortDesc).
        Limit(10).
        Execute()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nTop Categories:")
    for _, row := range categories.Data {
        fmt.Printf("  %v: %v products (avg $%v)\n",
            row["category"], row["count"], row["avg_price"])
    }
}
```

### Full Auth Flow

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/wowsql/wowsql-go/wowsql"
)

func main() {
    auth := wowsql.NewAuthClient(
        os.Getenv("WOWSQL_PROJECT_URL"),
        os.Getenv("WOWSQL_ANON_KEY"),
        wowsql.AuthWithTokenStorage(&wowsql.MemoryTokenStorage{}),
    )
    defer auth.Close()

    // 1. Sign up
    signUp, err := auth.SignUp("alice@example.com", "StrongPass!456",
        wowsql.WithFullName("Alice Johnson"),
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Signed up: %s (ID: %s)\n", signUp.User.Email, signUp.User.ID)

    // 2. Session is stored automatically via MemoryTokenStorage

    // 3. Fetch current user
    user, err := auth.GetUser()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Current user: %s, verified: %v\n", user.Email, user.EmailVerified)

    // 4. Update profile
    updated, err := auth.UpdateUser(
        wowsql.UpdateUsername("alicej"),
        wowsql.UpdateAvatarURL("https://example.com/alice.jpg"),
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Username set to: %s\n", updated.Username)

    // 5. Refresh token
    refreshed, err := auth.RefreshToken()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("New access token: %s...\n", refreshed.Session.AccessToken[:20])

    // 6. Logout
    err = auth.Logout()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Logged out.")
}
```

---

## Models Reference

| Model | Key Fields |
|-------|-----------|
| `QueryResponse` | `Data []map[string]interface{}`, `Count int` |
| `CreateResponse` | `ID interface{}`, `AffectedRows int` |
| `UpdateResponse` | `AffectedRows int` |
| `DeleteResponse` | `AffectedRows int` |
| `PaginatedResponse` | `Data`, `Page`, `PerPage`, `Total`, `TotalPages` |
| `TableSchema` | `Name string`, `Columns []ColumnInfo` |
| `ColumnInfo` | `Name`, `Type`, `Nullable`, `Default`, `Key` |
| `ColumnDefinition` | `Name`, `Type`, `NotNull`, `Unique`, `AutoIncrement`, `Default` |
| `AuthUser` | `ID`, `Email`, `FullName`, `Username`, `AvatarURL`, `EmailVerified`, `Metadata` |
| `AuthSession` | `AccessToken`, `RefreshToken`, `ExpiresAt` |
| `AuthResponse` | `User *AuthUser`, `Session *AuthSession` |
| `StorageBucket` | `Name`, `Public`, `FileSizeLimit`, `AllowedMimeTypes`, `CreatedAt` |
| `StorageFile` | `Key`, `Size`, `SizeMB`, `SizeGB`, `LastModified`, `ContentType` |
| `StorageQuota` | `StorageUsedGB`, `StorageQuotaGB`, `StorageAvailableGB`, `StorageAvailableBytes`, `UsagePercentage` |
| `FileUploadResult` | `Path`, `Size`, `ContentType` |
| `FilterExpression` | `Column`, `Operator`, `Value`, `LogicalOp` |
| `HavingFilter` | `Column`, `Operator`, `Value` |
| `OrderByItem` | `Column`, `Direction` |

---

## Troubleshooting

**Error: "Invalid API key for project"**
- Verify you are using the correct key type for the operation.
- Database operations accept Service Role Key or Anonymous Key.
- Schema operations require Service Role Key exclusively.
- Confirm there are no trailing spaces in the key.

**Error: "Authentication failed"**
- Check that the project URL matches your dashboard.
- Ensure the key has not been revoked or expired.
- For auth operations, use the Anonymous Key on the client side.

**Error: "Rate limit exceeded"**
- Back off and retry after a short delay.
- Consider caching frequently-accessed data.

**Error: "Permission denied" (schema operations)**
- Schema operations require a Service Role Key (`wowsql_service_...`).
- Anonymous keys will always return 403 for DDL.

---

## Requirements

- **Go** 1.21+
- A WowSQL project with valid API keys

---

## Links

- [Documentation](https://wowsql.com/docs)
- [Website](https://wowsql.com)
- [Discord](https://discord.gg/wowsql)
- [Issues](https://github.com/wowsql/wowsql/issues)
- [Go Package Reference](https://pkg.go.dev/github.com/wowsql/wowsql-go/wowsql)

## License

MIT License — see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome. Please open an issue or submit a pull request.

## Support

- Email: support@wowsql.com
- Discord: https://discord.gg/wowsql
- Documentation: https://wowsql.com/docs

