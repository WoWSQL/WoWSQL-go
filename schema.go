package WOWSQL

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// ColumnDefinition represents a column definition for table creation
type ColumnDefinition struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	AutoIncrement *bool   `json:"auto_increment,omitempty"`
	Unique        *bool   `json:"unique,omitempty"`
	Nullable      *bool   `json:"nullable,omitempty"`
	Default       *string `json:"default,omitempty"`
}

// CreateTableRequest represents a request to create a table
type CreateTableRequest struct {
	TableName  string             `json:"table_name"`
	Columns    []ColumnDefinition `json:"columns"`
	PrimaryKey *string            `json:"primary_key,omitempty"`
	Indexes    []string           `json:"indexes,omitempty"`
}

// AlterTableRequest represents a request to alter a table
type AlterTableRequest struct {
	TableName     string  `json:"table_name"`
	Operation     string  `json:"operation"` // add_column, drop_column, modify_column, rename_column
	ColumnName    *string `json:"column_name,omitempty"`
	ColumnType    *string `json:"column_type,omitempty"`
	NewColumnName *string `json:"new_column_name,omitempty"`
	Nullable      *bool   `json:"nullable,omitempty"`
	Default       *string `json:"default,omitempty"`
}

// SchemaResponse represents a schema operation response
type SchemaResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	Table        string `json:"table,omitempty"`
	Operation    string `json:"operation,omitempty"`
	RowsAffected int    `json:"rows_affected,omitempty"`
	Warning      string `json:"warning,omitempty"`
}

// SchemaClient handles schema management operations
// ⚠️ IMPORTANT: Requires SERVICE ROLE key, not anonymous key!
type SchemaClient struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

// NewSchemaClient creates a new schema management client
//
// ⚠️ IMPORTANT: Requires SERVICE ROLE key, not anonymous key!
func NewSchemaClient(projectURL, serviceKey string) *SchemaClient {
	return &SchemaClient{
		baseURL:    projectURL,
		serviceKey: serviceKey,
		httpClient: &http.Client{},
	}
}

// CreateTable creates a new table in the database
//
// Example:
//
//	trueVal := true
//	falseVal := false
//	err := schema.CreateTable(CreateTableRequest{
//	    TableName: "users",
//	    Columns: []ColumnDefinition{
//	        {Name: "id", Type: "INT", AutoIncrement: &trueVal},
//	        {Name: "email", Type: "VARCHAR(255)", Unique: &trueVal, Nullable: &falseVal},
//	    },
//	    PrimaryKey: strPtr("id"),
//	    Indexes: []string{"email"},
//	})
func (c *SchemaClient) CreateTable(req CreateTableRequest) (*SchemaResponse, error) {
	url := fmt.Sprintf("%s/api/v2/schema/tables", c.baseURL)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.serviceKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("schema operations require a SERVICE ROLE key. You are using an anonymous key which cannot modify database schema")
	}

	if resp.StatusCode != 200 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("failed to create table: %v", errorResp["detail"])
	}

	var result SchemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// AlterTable alters an existing table
//
// Example:
//
//	err := schema.AlterTable(AlterTableRequest{
//	    TableName: "users",
//	    Operation: "add_column",
//	    ColumnName: strPtr("phone"),
//	    ColumnType: strPtr("VARCHAR(20)"),
//	})
func (c *SchemaClient) AlterTable(req AlterTableRequest) (*SchemaResponse, error) {
	url := fmt.Sprintf("%s/api/v2/schema/tables/%s", c.baseURL, req.TableName)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.serviceKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("schema operations require a SERVICE ROLE key")
	}

	if resp.StatusCode != 200 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("failed to alter table: %v", errorResp["detail"])
	}

	var result SchemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DropTable drops a table from the database
//
// ⚠️ WARNING: This operation cannot be undone!
func (c *SchemaClient) DropTable(tableName string, cascade bool) (*SchemaResponse, error) {
	url := fmt.Sprintf("%s/api/v2/schema/tables/%s?cascade=%t", c.baseURL, tableName, cascade)

	httpReq, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.serviceKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("schema operations require a SERVICE ROLE key")
	}

	if resp.StatusCode != 200 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("failed to drop table: %v", errorResp["detail"])
	}

	var result SchemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ExecuteSQL executes raw SQL for schema operations
//
// Example:
//
//	err := schema.ExecuteSQL(`
//	    CREATE TABLE products (
//	        id INT PRIMARY KEY AUTO_INCREMENT,
//	        name VARCHAR(255) NOT NULL
//	    )
//	`)
func (c *SchemaClient) ExecuteSQL(sql string) (*SchemaResponse, error) {
	url := fmt.Sprintf("%s/api/v2/schema/execute", c.baseURL)

	jsonData, err := json.Marshal(map[string]string{"sql": sql})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.serviceKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("schema operations require a SERVICE ROLE key")
	}

	if resp.StatusCode != 200 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("failed to execute SQL: %v", errorResp["detail"])
	}

	var result SchemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}
