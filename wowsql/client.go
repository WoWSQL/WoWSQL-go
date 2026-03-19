package WOWSQL

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ClientOption configures the Client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	baseDomain string
	secure     bool
	timeout    time.Duration
	verifySSL  bool
}

// WithBaseDomain sets the base domain (default: "wowsql.com").
func WithBaseDomain(domain string) ClientOption {
	return func(c *clientConfig) { c.baseDomain = domain }
}

// WithSecure toggles HTTPS (default: true).
func WithSecure(secure bool) ClientOption {
	return func(c *clientConfig) { c.secure = secure }
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) { c.timeout = d }
}

// WithVerifySSL enables or disables SSL certificate verification.
func WithVerifySSL(verify bool) ClientOption {
	return func(c *clientConfig) { c.verifySSL = verify }
}

// Client represents the WOWSQL database client.
type Client struct {
	baseURL    string
	apiURL     string
	apiKey     string
	timeout    time.Duration
	httpClient *http.Client
}

// NewClient creates a new WOWSQL client.
//
// projectURL can be a slug ("myproject"), a domain ("myproject.wowsql.com"),
// or a full URL ("https://myproject.wowsql.com" or "https://myproject.wowsql.com/api").
func NewClient(projectURL, apiKey string, opts ...ClientOption) *Client {
	cfg := &clientConfig{
		baseDomain: "wowsql.com",
		secure:     true,
		timeout:    30 * time.Second,
		verifySSL:  true,
	}
	for _, o := range opts {
		o(cfg)
	}

	baseURL, apiURL := buildClientURLs(projectURL, cfg.baseDomain, cfg.secure)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !cfg.verifySSL {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &Client{
		baseURL: baseURL,
		apiURL:  apiURL,
		apiKey:  apiKey,
		timeout: cfg.timeout,
		httpClient: &http.Client{
			Timeout:   cfg.timeout,
			Transport: transport,
		},
	}
}

// Table returns a new Table instance for the given table name.
func (c *Client) Table(tableName string) *Table {
	return &Table{
		client:    c,
		tableName: tableName,
	}
}

// Schema returns a new SchemaClient for schema management operations.
// Requires a SERVICE ROLE key, not an anonymous key.
func (c *Client) Schema() *SchemaClient {
	return NewSchemaClient(c.baseURL, c.apiKey)
}

// ListTables lists all tables in the database.
func (c *Client) ListTables() ([]string, error) {
	resp, err := c.doRequest("GET", "/tables", nil)
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

// GetTableSchema gets the schema information for a table.
func (c *Client) GetTableSchema(tableName string) (*TableSchema, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/tables/%s/schema", tableName), nil)
	if err != nil {
		return nil, err
	}

	var schema TableSchema
	if err := json.Unmarshal(resp, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &schema, nil
}

// Query executes a raw SQL query (read-only).
func (c *Client) Query(sql string) ([]map[string]interface{}, error) {
	body := map[string]interface{}{
		"sql": sql,
	}

	resp, err := c.doRequestRaw("POST", c.baseURL+"/api/v1/query", body)
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

// Health checks the API health.
func (c *Client) Health() (map[string]interface{}, error) {
	resp, err := c.doRequestRaw("GET", c.baseURL+"/api/v1/health", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// Close releases resources held by the client.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

// doRequest performs an HTTP request using the v2 API URL.
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	url := c.apiURL + path
	return c.doRequestRaw(method, url, body)
}

// doRequestRaw performs an HTTP request to an absolute URL.
func (c *Client) doRequestRaw(method, url string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

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

// buildClientURLs builds the base URL and v2 API URL from various input formats.
func buildClientURLs(projectURL, baseDomain string, secure bool) (baseURL, apiURL string) {
	normalized := strings.TrimSpace(projectURL)

	if strings.HasPrefix(normalized, "http://") || strings.HasPrefix(normalized, "https://") {
		baseURL = strings.TrimSuffix(normalized, "/")
		if strings.Contains(baseURL, "/api") {
			baseURL = strings.Split(baseURL, "/api")[0]
		}
		apiURL = baseURL + "/api/v2"
		return
	}

	protocol := "https"
	if !secure {
		protocol = "http"
	}

	if strings.Contains(normalized, "."+baseDomain) || strings.HasSuffix(normalized, baseDomain) {
		baseURL = fmt.Sprintf("%s://%s", protocol, normalized)
	} else {
		baseURL = fmt.Sprintf("%s://%s.%s", protocol, normalized, baseDomain)
	}

	apiURL = baseURL + "/api/v2"
	return
}
