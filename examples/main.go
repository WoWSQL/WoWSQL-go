package main

import (
	"bytes"
	"fmt"
	"log"

	"github.com/wowsql/wowsql-go/wowsql"
)

const storageBucket = "default"

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
	updated, err := client.Table("users").Update(newUser.ID, map[string]interface{}{
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
	deleted, err := client.Table("users").Delete(newUser.ID)
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

	// 1. Get storage quota / stats
	fmt.Println("1. Get storage quota")
	quota, err := storage.GetQuota()
	if err != nil {
		log.Fatalf("Failed to get quota: %v", err)
	}
	fmt.Printf("Used: %.2f GB\n", quota.StorageUsedGB)
	fmt.Printf("Available: %.2f GB\n", quota.StorageAvailableGB)
	fmt.Printf("Total: %.2f GB\n", quota.StorageQuotaGB)
	usagePct := quota.UsagePercentage
	if usagePct == 0 && quota.StorageQuotaGB > 0 {
		usagePct = (quota.StorageUsedGB / quota.StorageQuotaGB) * 100
	}
	fmt.Printf("Usage: %.1f%%\n\n", usagePct)

	// 2. Upload file
	fmt.Println("2. Upload file")
	fileData := []byte("Hello, WOWSQL!")
	uploadResult, err := storage.Upload(storageBucket, bytes.NewReader(fileData),
		WOWSQL.UploadPath("uploads/test.txt"),
	)
	if err != nil {
		log.Fatalf("Failed to upload file: %v", err)
	}
	fmt.Printf("Uploaded path: %s\n", uploadResult.Path)
	if uploadResult.PublicURL != "" {
		fmt.Printf("URL: %s\n\n", uploadResult.PublicURL)
	} else {
		fmt.Printf("Public URL: %s\n\n", storage.GetPublicURL(storageBucket, "uploads/test.txt"))
	}

	// 3. List files
	fmt.Println("3. List files")
	files, err := storage.ListFiles(storageBucket, WOWSQL.ListFilesPrefix("uploads/"))
	if err != nil {
		log.Fatalf("Failed to list files: %v", err)
	}
	fmt.Printf("Found %d files:\n", len(files))
	for _, file := range files {
		key := file.Path
		if file.Key != "" {
			key = file.Key
		}
		fmt.Printf("  - %s (%d bytes)\n", key, file.Size)
	}
	fmt.Println()

	// 4. Public URL for a file
	fmt.Println("4. Public URL")
	publicURL := storage.GetPublicURL(storageBucket, "uploads/test.txt")
	fmt.Printf("URL: %s\n\n", publicURL)

	// 5. File metadata (from list — SDK has no separate GetFileInfo)
	fmt.Println("5. File metadata (from list)")
	var meta *WOWSQL.StorageFile
	for i := range files {
		p := files[i].Path
		if files[i].Key != "" {
			p = files[i].Key
		}
		if p == "uploads/test.txt" {
			meta = &files[i]
			break
		}
	}
	if meta != nil {
		fmt.Printf("Path: %s\n", meta.Path)
		fmt.Printf("Size: %d bytes\n", meta.Size)
		if meta.LastModified != "" {
			fmt.Printf("Modified: %s\n", meta.LastModified)
		}
	} else {
		fmt.Println("(file not found in list — upload may be required first)")
	}
	fmt.Println()

	// 6. Check if file exists (via list)
	fmt.Println("6. Check if file exists")
	exists := false
	for i := range files {
		p := files[i].Path
		if files[i].Key != "" {
			p = files[i].Key
		}
		if p == "uploads/test.txt" {
			exists = true
			break
		}
	}
	fmt.Printf("File exists: %v\n\n", exists)

	// 7. Download file (binary contents)
	fmt.Println("7. Download file")
	downloaded, err := storage.Download(storageBucket, "uploads/test.txt")
	if err != nil {
		log.Fatalf("Failed to download file: %v", err)
	}
	fmt.Printf("Downloaded %d bytes\n\n", len(downloaded))

	// 8. Delete file
	fmt.Println("8. Delete file")
	_, err = storage.DeleteFile(storageBucket, "uploads/test.txt")
	if err != nil {
		log.Fatalf("Failed to delete file: %v", err)
	}
	fmt.Println("File deleted")

	// 9. Delete multiple files
	fmt.Println("9. Delete multiple files")
	for _, path := range []string{"uploads/file1.txt", "uploads/file2.txt"} {
		_, err := storage.DeleteFile(storageBucket, path)
		if err != nil {
			log.Printf("delete %s: %v (ignored in demo)", path, err)
		}
	}
	fmt.Println("Multiple delete attempts completed")

	// 10. Check API health
	fmt.Println("10. Check API health")
	health, err := client.Health()
	if err != nil {
		log.Fatalf("Failed to check health: %v", err)
	}
	fmt.Printf("Status: %v\n\n", health["status"])

	fmt.Println("✅ All operations completed successfully!")
}
