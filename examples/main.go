package main

import (
	"fmt"
	"log"

	"github.com/wowsql/wowsql-go/WOWSQL"
)

func main() {
	// Initialize database client
	client := WOWSQL.NewClient(
		"https://your-project.wowsql.com",
		"your-api-key",
	)

	// Initialize storage client
	storage := WOWSQL.NewStorageClient(
		"https://your-project.wowsql.com",
		"your-api-key",
	)

	fmt.Println("=== DATABASE OPERATIONS ===")

	// 1. List all tables
	fmt.Println("1. List all tables")
	tables, err := client.ListTables()
	if err != nil {
		log.Fatalf("Failed to list tables: %v", err)
	}
	fmt.Printf("Tables: %v\n\n", tables)

	// 2. Get table schema
	fmt.Println("2. Get table schema")
	schema, err := client.GetTableSchema("users")
	if err != nil {
		log.Fatalf("Failed to get schema: %v", err)
	}
	fmt.Printf("Columns: %d\n", len(schema.Columns))
	fmt.Printf("Row count: %v\n\n", schema.RowCount)

	// 3. Select all users
	fmt.Println("3. Select all users")
	allUsers, err := client.Table("users").Select("*").Get()
	if err != nil {
		log.Fatalf("Failed to get users: %v", err)
	}
	fmt.Printf("Found %d users\n\n", allUsers.Count)

	// 4. Select with filters
	fmt.Println("4. Select active users")
	activeUsers, err := client.Table("users").
		Select("id", "name", "email").
		Eq("status", "active").
		Limit(10).
		Execute()
	if err != nil {
		log.Fatalf("Failed to get active users: %v", err)
	}
	fmt.Printf("Active users: %d\n\n", activeUsers.Count)

	// 5. Insert new user
	fmt.Println("5. Insert new user")
	newUser, err := client.Table("users").Insert(map[string]interface{}{
		"name":   "John Doe",
		"email":  "john@example.com",
		"age":    30,
		"status": "active",
	})
	if err != nil {
		log.Fatalf("Failed to insert user: %v", err)
	}
	fmt.Printf("New user ID: %v\n\n", newUser.ID)

	// 6. Update user
	fmt.Println("6. Update user")
	updated, err := client.Table("users").UpdateByID(newUser.ID, map[string]interface{}{
		"name": "John Smith",
	})
	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}
	fmt.Printf("Updated %d row(s)\n\n", updated.AffectedRows)

	// 7. Complex query
	fmt.Println("7. Complex query")
	results, err := client.Table("users").
		Select("id", "name", "email").
		Gt("age", 18).
		Like("email", "%@example.com").
		OrderBy("created_at", WOWSQL.SortDesc).
		Limit(5).
		Execute()
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
	fmt.Printf("Results: %d\n\n", results.Count)

	// 8. Get first result
	fmt.Println("8. Get first user")
	firstUser, err := client.Table("users").
		Select("*").
		Eq("email", "john@example.com").
		First()
	if err != nil {
		log.Fatalf("Failed to get first user: %v", err)
	}
	if firstUser != nil {
		fmt.Printf("User: %v\n\n", firstUser["name"])
	}

	// 9. Delete user
	fmt.Println("9. Delete user")
	deleted, err := client.Table("users").DeleteByID(newUser.ID)
	if err != nil {
		log.Fatalf("Failed to delete user: %v", err)
	}
	fmt.Printf("Deleted %d row(s)\n\n", deleted.AffectedRows)

	// 10. Raw SQL query
	fmt.Println("10. Raw SQL query")
	sqlResults, err := client.Query("SELECT COUNT(*) as count FROM users WHERE age > 18")
	if err != nil {
		log.Fatalf("Failed to execute SQL: %v", err)
	}
	if len(sqlResults) > 0 {
		fmt.Printf("Count: %v\n\n", sqlResults[0]["count"])
	}

	fmt.Println("=== STORAGE OPERATIONS ===")

	// 1. Get storage quota
	fmt.Println("1. Get storage quota")
	quota, err := storage.GetQuota()
	if err != nil {
		log.Fatalf("Failed to get quota: %v", err)
	}
	fmt.Printf("Used: %.2f GB\n", quota.StorageUsedGB)
	fmt.Printf("Available: %.2f GB\n", quota.StorageAvailableGB)
	fmt.Printf("Total: %.2f GB\n", quota.StorageQuotaGB)
	fmt.Printf("Usage: %.1f%%\n\n", quota.UsagePercentage)

	// 2. Upload file
	fmt.Println("2. Upload file")
	fileData := []byte("Hello, WOWSQL!")
	uploadResult, err := storage.Upload(fileData, "uploads/test.txt", "text/plain", nil)
	if err != nil {
		log.Fatalf("Failed to upload file: %v", err)
	}
	fmt.Printf("Uploaded: %s\n", uploadResult.Key)
	fmt.Printf("URL: %s\n\n", uploadResult.URL)

	// 3. List files
	fmt.Println("3. List files")
	files, err := storage.ListFiles("uploads/", 0)
	if err != nil {
		log.Fatalf("Failed to list files: %v", err)
	}
	fmt.Printf("Found %d files:\n", len(files))
	for _, file := range files {
		fmt.Printf("  - %s (%d bytes)\n", file.Key, file.Size)
	}
	fmt.Println()

	// 4. Get file info
	fmt.Println("4. Get file info")
	fileInfo, err := storage.GetFileInfo("uploads/test.txt")
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}
	fmt.Printf("Key: %s\n", fileInfo.Key)
	fmt.Printf("Size: %d bytes\n", fileInfo.Size)
	fmt.Printf("Modified: %s\n\n", fileInfo.LastModified)

	// 5. Check if file exists
	fmt.Println("5. Check if file exists")
	exists, err := storage.FileExists("uploads/test.txt")
	if err != nil {
		log.Fatalf("Failed to check file: %v", err)
	}
	fmt.Printf("File exists: %v\n\n", exists)

	// 6. Download file (get presigned URL)
	fmt.Println("6. Download file")
	downloadURL, err := storage.Download("uploads/test.txt", 3600)
	if err != nil {
		log.Fatalf("Failed to get download URL: %v", err)
	}
	fmt.Printf("Download URL: %s\n\n", downloadURL)

	// 7. Delete file
	fmt.Println("7. Delete file")
	err = storage.DeleteFile("uploads/test.txt")
	if err != nil {
		log.Fatalf("Failed to delete file: %v", err)
	}
	fmt.Println("File deleted")

	// 8. Delete multiple files
	fmt.Println("8. Delete multiple files")
	err = storage.DeleteFiles([]string{
		"uploads/file1.txt",
		"uploads/file2.txt",
	})
	if err != nil {
		log.Fatalf("Failed to delete files: %v", err)
	}
	fmt.Println("Multiple files deleted")

	// 9. Check API health
	fmt.Println("9. Check API health")
	health, err := client.Health()
	if err != nil {
		log.Fatalf("Failed to check health: %v", err)
	}
	fmt.Printf("Status: %v\n\n", health["status"])

	fmt.Println("✅ All operations completed successfully!")
}

