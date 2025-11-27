package WOWSQL

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// ColumnDefinition represents a column in a table
type ColumnDefinition struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	AutoIncrement bool   `json:"auto_increment,omitempty"`
	Unique        bool   `json:"unique,omitempty"`
	Nullable      bool   `json:"nullable,omitempty"`
	Default       string `json:"default,omitempty"`
}

// CreateTableOptions contains options for creating a table
type CreateTableOptions struct {
	TableName  string             `json:"table_name"`
	Columns    []ColumnDefinition `json:"columns"`
	PrimaryKey string             `json:"primary_key,omitempty"`
	Indexes    []string           `json:"indexes,omitempty"`
}

// AlterTableOptions contains options for altering a table
type AlterTableOptions struct {
	TableName     string `json:"table_name"`
	Operation     string `json:"operation"` // add_column, drop_column, modify_column, rename_column
	ColumnName    string `json:"column_name,omitempty"`
	ColumnType    string `json:"column_type,omitempty"`
	NewColumnName string `json:"new_column_name,omitempty"`
	Nullable      *bool  `json:"nullable,omitempty"`
	Default       string `json:"default,omitempty"`
}

// SchemaClient handles schema operations
type SchemaClient struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

// NewSchemaClient creates a new schema client
// ⚠️ IMPORTANT: Requires SERVICE ROLE key, not anonymous key!
func NewSchemaClient(projectURL, serviceKey string) *SchemaClient {
	return &SchemaClient{
		baseURL:    projectURL,
		serviceKey: serviceKey,
		httpClient: &http.Client{},
	}
}

// CreateTable creates a new table
func (s *SchemaClient) CreateTable(options CreateTableOptions) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v2/schema/tables", s.baseURL)

	jsonData, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("schema operations require a SERVICE ROLE key. You are using an anonymous key which cannot modify database schema")
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		if detail, ok := errorResp["detail"].(string); ok {
			return nil, fmt.Errorf("failed to create table: %s", detail)
		}
		return nil, fmt.Errorf("failed to create table: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// AlterTable modifies an existing table
func (s *SchemaClient) AlterTable(options AlterTableOptions) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v2/schema/tables/%s", s.baseURL, options.TableName)

	jsonData, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("schema operations require a SERVICE ROLE key")
	}

	if resp.StatusCode != 200 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		if detail, ok := errorResp["detail"].(string); ok {
			return nil, fmt.Errorf("failed to alter table: %s", detail)
		}
		return nil, fmt.Errorf("failed to alter table: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// DropTable deletes a table
// ⚠️ WARNING: This operation cannot be undone!
func (s *SchemaClient) DropTable(tableName string, cascade bool) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v2/schema/tables/%s?cascade=%t", s.baseURL, tableName, cascade)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("schema operations require a SERVICE ROLE key")
	}

	if resp.StatusCode != 200 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		if detail, ok := errorResp["detail"].(string); ok {
			return nil, fmt.Errorf("failed to drop table: %s", detail)
		}
		return nil, fmt.Errorf("failed to drop table: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// ExecuteSQL executes raw SQL for schema operations
func (s *SchemaClient) ExecuteSQL(sql string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v2/schema/execute", s.baseURL)

	payload := map[string]string{"sql": sql}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("schema operations require a SERVICE ROLE key")
	}

	if resp.StatusCode != 200 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		if detail, ok := errorResp["detail"].(string); ok {
			return nil, fmt.Errorf("failed to execute SQL: %s", detail)
		}
		return nil, fmt.Errorf("failed to execute SQL: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
