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

// WithBaseDomain sets the base domain (default: "wowsqlconnect.com").
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

// Client represents the WowSQL database client.
// All operations communicate directly with PostgREST (/rest/v1).
type Client struct {
	baseURL    string
	apiURL     string
	apiKey     string
	timeout    time.Duration
	httpClient *http.Client
	realtime   *Realtime
}

// NewClient creates a new WowSQL client.
//
// projectURL can be a slug ("myproject"), a domain ("myproject.wowsqlconnect.com"),
// or a full URL ("https://myproject.wowsqlconnect.com").
// All requests are sent directly to the PostgREST endpoint (/rest/v1).
func NewClient(projectURL, apiKey string, opts ...ClientOption) *Client {
	cfg := &clientConfig{
		baseDomain: "wowsqlconnect.com",
		secure:     true,
		timeout:    30 * time.Second,
		verifySSL:  true,
	}
	for _, o := range opts {
		o(cfg)
	}

	baseURL := buildBaseURL(projectURL, cfg.baseDomain, cfg.secure)
	apiURL := baseURL + "/rest/v1"

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

// Close releases resources held by the client.
func (c *Client) Close() {
	if c.realtime != nil {
		c.realtime.Close()
	}
	c.httpClient.CloseIdleConnections()
}

// doRequest performs an HTTP request using the PostgREST API URL.
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, *http.Response, error) {
	url := c.apiURL + path
	return c.doRequestRaw(method, url, body)
}

// doRequestRaw performs an HTTP request to an absolute URL and returns body + response.
func (c *Client) doRequestRaw(method, url string, body interface{}) ([]byte, *http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp, parseError(resp.StatusCode, respBody)
	}

	return respBody, resp, nil
}

// doRequestSimple performs a request and returns only the body bytes.
func (c *Client) doRequestSimple(method, path string, body interface{}) ([]byte, error) {
	b, _, err := c.doRequest(method, path, body)
	return b, err
}

// doRequestWithHeaders performs a request with custom headers and returns body + response.
func (c *Client) doRequestWithHeaders(method, path string, body interface{}, headers map[string]string) ([]byte, *http.Response, error) {
	url := c.apiURL + path
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apikey", c.apiKey)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp, parseError(resp.StatusCode, respBody)
	}

	return respBody, resp, nil
}

// buildBaseURL builds the base URL (without API path) from various input formats.
func buildBaseURL(projectURL, baseDomain string, secure bool) string {
	normalized := strings.TrimSpace(projectURL)

	if strings.HasPrefix(normalized, "http://") || strings.HasPrefix(normalized, "https://") {
		base := strings.TrimSuffix(normalized, "/")
		if strings.Contains(base, "/api") {
			base = strings.Split(base, "/api")[0]
		}
		return base
	}

	protocol := "https"
	if !secure {
		protocol = "http"
	}

	if strings.Contains(normalized, "."+baseDomain) || strings.HasSuffix(normalized, baseDomain) {
		return fmt.Sprintf("%s://%s", protocol, normalized)
	}
	return fmt.Sprintf("%s://%s.%s", protocol, normalized, baseDomain)
}
