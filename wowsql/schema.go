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

// ── Schema client options ───────────────────────────────────────

// SchemaClientOption configures the SchemaClient.
type SchemaClientOption func(*schemaClientConfig)

type schemaClientConfig struct {
	baseDomain string
	secure     bool
	timeout    time.Duration
	verifySSL  bool
}

// SchemaWithBaseDomain sets the base domain.
func SchemaWithBaseDomain(domain string) SchemaClientOption {
	return func(c *schemaClientConfig) { c.baseDomain = domain }
}

// SchemaWithSecure toggles HTTPS.
func SchemaWithSecure(secure bool) SchemaClientOption {
	return func(c *schemaClientConfig) { c.secure = secure }
}

// SchemaWithTimeout sets the HTTP timeout.
func SchemaWithTimeout(d time.Duration) SchemaClientOption {
	return func(c *schemaClientConfig) { c.timeout = d }
}

// SchemaWithVerifySSL enables/disables SSL verification.
func SchemaWithVerifySSL(verify bool) SchemaClientOption {
	return func(c *schemaClientConfig) { c.verifySSL = verify }
}

// ── SchemaClient ────────────────────────────────────────────────

// SchemaClient handles schema operations.
// Requires a SERVICE ROLE key, not an anonymous key.
type SchemaClient struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

// NewSchemaClient creates a new schema client.
//
// projectURL can be a slug, domain, or full URL.
// serviceKey must be a service role key (wowsql_service_...).
func NewSchemaClient(projectURL, serviceKey string, opts ...SchemaClientOption) *SchemaClient {
	cfg := &schemaClientConfig{
		baseDomain: "wowsqlconnect.com",
		secure:     true,
		timeout:    30 * time.Second,
		verifySSL:  true,
	}
	for _, o := range opts {
		o(cfg)
	}

	baseURL := buildSchemaBaseURL(projectURL, cfg.baseDomain, cfg.secure)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !cfg.verifySSL {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &SchemaClient{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		httpClient: &http.Client{
			Timeout:   cfg.timeout,
			Transport: transport,
		},
	}
}

// ── Table operations ────────────────────────────────────────────

// CreateTable creates a new table.
// TablePrimaryKey is required; that column must use PostgreSQL type UUID.
func (s *SchemaClient) CreateTable(tableName string, columns []ColumnDefinition, opts ...CreateTableOption) (map[string]interface{}, error) {
	cfg := &createTableConfig{}
	for _, o := range opts {
		o(cfg)
	}
	if cfg.primaryKey == "" {
		return nil, fmt.Errorf("wowsql: TablePrimaryKey is required; primary key column must be UUID")
	}
	if err := validateUUIDPrimaryKey(cfg.primaryKey, columns); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"table_name":  tableName,
		"columns":     columns,
		"primary_key": cfg.primaryKey,
	}
	if len(cfg.indexes) > 0 {
		body["indexes"] = cfg.indexes
	}

	return s.doRequest("POST", "/api/v2/schema/tables", body)
}

func validateUUIDPrimaryKey(pk string, columns []ColumnDefinition) error {
	for _, c := range columns {
		if c.Name != pk {
			continue
		}
		t := strings.TrimSpace(c.Type)
		if t == "" {
			return fmt.Errorf("wowsql: primary key column must have type UUID")
		}
		first := strings.ToUpper(strings.Fields(t)[0])
		if first == "UUID" {
			return nil
		}
		return fmt.Errorf("wowsql: primary key column must have type UUID, got %q", c.Type)
	}
	return fmt.Errorf("wowsql: primary key column %q not found in columns", pk)
}

// CreateTableOption configures a CreateTable call.
type CreateTableOption func(*createTableConfig)

type createTableConfig struct {
	primaryKey string
	indexes    []string
}

// TablePrimaryKey sets the primary key column.
func TablePrimaryKey(column string) CreateTableOption {
	return func(c *createTableConfig) { c.primaryKey = column }
}

// TableIndexes sets columns to index.
func TableIndexes(columns ...string) CreateTableOption {
	return func(c *createTableConfig) { c.indexes = columns }
}

// AlterTable modifies an existing table.
func (s *SchemaClient) AlterTable(tableName, operation string, opts ...AlterTableOption) (map[string]interface{}, error) {
	cfg := &alterTableConfig{}
	for _, o := range opts {
		o(cfg)
	}

	body := map[string]interface{}{
		"table_name": tableName,
		"operation":  operation,
	}
	if cfg.columnName != "" {
		body["column_name"] = cfg.columnName
	}
	if cfg.columnType != "" {
		body["column_type"] = cfg.columnType
	}
	if cfg.newColumnName != "" {
		body["new_column_name"] = cfg.newColumnName
	}
	if cfg.nullable != nil {
		body["nullable"] = *cfg.nullable
	}
	if cfg.defaultVal != "" {
		body["default"] = cfg.defaultVal
	}

	return s.doRequest("PATCH", fmt.Sprintf("/api/v2/schema/tables/%s", tableName), body)
}

// AlterTableOption configures an AlterTable call.
type AlterTableOption func(*alterTableConfig)

type alterTableConfig struct {
	columnName    string
	columnType    string
	newColumnName string
	nullable      *bool
	defaultVal    string
}

// AlterColumnName sets the column name.
func AlterColumnName(name string) AlterTableOption {
	return func(c *alterTableConfig) { c.columnName = name }
}

// AlterColumnType sets the column type.
func AlterColumnType(typ string) AlterTableOption {
	return func(c *alterTableConfig) { c.columnType = typ }
}

// AlterNewColumnName sets the new column name (for rename).
func AlterNewColumnName(name string) AlterTableOption {
	return func(c *alterTableConfig) { c.newColumnName = name }
}

// AlterNullable sets whether the column is nullable.
func AlterNullable(nullable bool) AlterTableOption {
	return func(c *alterTableConfig) { c.nullable = &nullable }
}

// AlterDefault sets the column default value.
func AlterDefault(def string) AlterTableOption {
	return func(c *alterTableConfig) { c.defaultVal = def }
}

// DropTable deletes a table.
func (s *SchemaClient) DropTable(tableName string, cascade ...bool) (map[string]interface{}, error) {
	cascadeVal := false
	if len(cascade) > 0 {
		cascadeVal = cascade[0]
	}
	path := fmt.Sprintf("/api/v2/schema/tables/%s?cascade=%t", tableName, cascadeVal)

	req, err := http.NewRequest("DELETE", s.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == 403 {
		return nil, &SchemaPermissionError{
			WOWSQLError: WOWSQLError{
				Message:    "Schema operations require a SERVICE ROLE key. You are using an anonymous key which cannot modify database schema.",
				StatusCode: 403,
			},
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, s.parseSchemaError(resp.StatusCode, respBody, "drop table")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return result, nil
}

// ExecuteSQL executes raw DDL SQL.
func (s *SchemaClient) ExecuteSQL(sql string) (map[string]interface{}, error) {
	return s.doRequest("POST", "/api/v2/schema/execute", map[string]string{"sql": sql})
}

// ── Convenience methods ─────────────────────────────────────────

// AddColumn adds a column to an existing table.
func (s *SchemaClient) AddColumn(tableName, columnName, columnType string, opts ...AlterTableOption) (map[string]interface{}, error) {
	allOpts := append([]AlterTableOption{
		AlterColumnName(columnName),
		AlterColumnType(columnType),
	}, opts...)
	return s.AlterTable(tableName, "add_column", allOpts...)
}

// DropColumn drops a column from a table.
func (s *SchemaClient) DropColumn(tableName, columnName string) (map[string]interface{}, error) {
	return s.AlterTable(tableName, "drop_column", AlterColumnName(columnName))
}

// RenameColumn renames a column.
func (s *SchemaClient) RenameColumn(tableName, oldName, newName string) (map[string]interface{}, error) {
	return s.AlterTable(tableName, "rename_column",
		AlterColumnName(oldName),
		AlterNewColumnName(newName),
	)
}

// ModifyColumn changes a column's type, nullability, or default.
func (s *SchemaClient) ModifyColumn(tableName, columnName string, opts ...AlterTableOption) (map[string]interface{}, error) {
	allOpts := append([]AlterTableOption{AlterColumnName(columnName)}, opts...)
	return s.AlterTable(tableName, "modify_column", allOpts...)
}

// CreateIndex creates an index on a table.
func (s *SchemaClient) CreateIndex(tableName string, columns []string, opts ...CreateIndexOption) (map[string]interface{}, error) {
	cfg := &createIndexConfig{}
	for _, o := range opts {
		o(cfg)
	}

	idxName := cfg.name
	if idxName == "" {
		idxName = fmt.Sprintf("idx_%s_%s", tableName, strings.Join(columns, "_"))
	}

	uniqueKW := ""
	if cfg.unique {
		uniqueKW = "UNIQUE "
	}
	usingKW := ""
	if cfg.using != "" {
		usingKW = fmt.Sprintf(" USING %s", cfg.using)
	}

	quotedCols := make([]string, len(columns))
	for i, c := range columns {
		quotedCols[i] = fmt.Sprintf(`"%s"`, c)
	}
	colList := strings.Join(quotedCols, ", ")

	sql := fmt.Sprintf(`CREATE %sINDEX IF NOT EXISTS "%s" ON "%s"%s (%s)`,
		uniqueKW, idxName, tableName, usingKW, colList)

	return s.ExecuteSQL(sql)
}

// CreateIndexOption configures a CreateIndex call.
type CreateIndexOption func(*createIndexConfig)

type createIndexConfig struct {
	unique bool
	name   string
	using  string
}

// IndexUnique creates a UNIQUE index.
func IndexUnique(unique bool) CreateIndexOption {
	return func(c *createIndexConfig) { c.unique = unique }
}

// IndexName sets a custom index name.
func IndexName(name string) CreateIndexOption {
	return func(c *createIndexConfig) { c.name = name }
}

// IndexUsing sets the index method (btree, hash, gin, gist).
func IndexUsing(method string) CreateIndexOption {
	return func(c *createIndexConfig) { c.using = method }
}

// ListTables lists all tables via the v2 REST API.
func (s *SchemaClient) ListTables() ([]string, error) {
	result, err := s.doRequest("GET", "/api/v2/tables", nil)
	if err != nil {
		return nil, err
	}
	if tables, ok := result["tables"]; ok {
		if arr, ok := tables.([]interface{}); ok {
			strs := make([]string, 0, len(arr))
			for _, v := range arr {
				if s, ok := v.(string); ok {
					strs = append(strs, s)
				}
			}
			return strs, nil
		}
	}
	return []string{}, nil
}

// GetTableSchema gets column-level schema information for a table.
func (s *SchemaClient) GetTableSchema(tableName string) (map[string]interface{}, error) {
	return s.doRequest("GET", fmt.Sprintf("/api/v2/tables/%s/schema", tableName), nil)
}

// Close releases resources.
func (s *SchemaClient) Close() {
	s.httpClient.CloseIdleConnections()
}

// ── Internals ───────────────────────────────────────────────────

func (s *SchemaClient) doRequest(method, path string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	reqURL := s.baseURL + path
	req, err := http.NewRequest(method, reqURL, bodyReader)
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == 403 {
		return nil, &SchemaPermissionError{
			WOWSQLError: WOWSQLError{
				Message:    "Schema operations require a SERVICE ROLE key. You are using an anonymous key which cannot modify database schema.",
				StatusCode: 403,
			},
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, s.parseSchemaError(resp.StatusCode, respBody, "schema operation")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return result, nil
}

func (s *SchemaClient) parseSchemaError(statusCode int, body []byte, operation string) error {
	var errorResp map[string]interface{}
	_ = json.Unmarshal(body, &errorResp)

	if detail, ok := errorResp["detail"].(string); ok {
		return &WOWSQLError{
			Message:    fmt.Sprintf("failed to %s: %s", operation, detail),
			StatusCode: statusCode,
			Response:   errorResp,
		}
	}
	return &WOWSQLError{
		Message:    fmt.Sprintf("failed to %s: status %d", operation, statusCode),
		StatusCode: statusCode,
		Response:   errorResp,
	}
}

func buildSchemaBaseURL(projectURL, baseDomain string, secure bool) string {
	normalized := strings.TrimSpace(projectURL)

	if strings.HasPrefix(normalized, "http://") || strings.HasPrefix(normalized, "https://") {
		normalized = strings.TrimSuffix(normalized, "/")
		if strings.Contains(normalized, "/api") {
			normalized = strings.Split(normalized, "/api")[0]
		}
		return normalized
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
