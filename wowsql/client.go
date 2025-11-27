package WOWSQL

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents the WOWSQL database client
type Client struct {
	projectURL string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new WOWSQL client
func NewClient(projectURL, apiKey string) *Client {
	return &Client{
		projectURL: projectURL,
		apiKey:     apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithTimeout creates a new WOWSQL client with custom timeout
func NewClientWithTimeout(projectURL, apiKey string, timeout time.Duration) *Client {
	return &Client{
		projectURL: projectURL,
		apiKey:     apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Table returns a new Table instance for the given table name
func (c *Client) Table(tableName string) *Table {
	return &Table{
		client:    c,
		tableName: tableName,
	}
}

// Schema returns a new SchemaClient for schema management operations
// ⚠️ IMPORTANT: Requires SERVICE ROLE key, not anonymous key!
func (c *Client) Schema() *SchemaClient {
	return NewSchemaClient(c.projectURL, c.apiKey)
}

// ListTables lists all tables in the database
func (c *Client) ListTables() ([]string, error) {
	resp, err := c.doRequest("GET", "/api/v1/tables", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Tables []string `json:"tables"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Tables, nil
}

// GetTableSchema gets the schema information for a table
func (c *Client) GetTableSchema(tableName string) (*TableSchema, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/v1/tables/%s/schema", tableName), nil)
	if err != nil {
		return nil, err
	}

	var schema TableSchema
	if err := json.Unmarshal(resp, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &schema, nil
}

// Query executes a raw SQL query (read-only)
func (c *Client) Query(sql string) ([]map[string]interface{}, error) {
	body := map[string]interface{}{
		"sql": sql,
	}

	resp, err := c.doRequest("POST", "/api/v1/query", body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Data, nil
}

// Health checks the API health
func (c *Client) Health() (map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/v1/health", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// doRequest performs an HTTP request
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := c.projectURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

