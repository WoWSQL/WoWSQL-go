package WOWSQL

import "encoding/json"

// ── Query / filter types ────────────────────────────────────────

// HavingFilter represents a HAVING clause condition.
type HavingFilter struct {
	Column   string      `json:"column"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// ── Response types ──────────────────────────────────────────────

// QueryResponse represents a query response.
type QueryResponse struct {
	Data   []map[string]interface{} `json:"data"`
	Count  int                      `json:"count"`
	Total  *int                     `json:"total,omitempty"`
	Limit  *int                     `json:"limit,omitempty"`
	Offset *int                     `json:"offset,omitempty"`
	Error  *string                  `json:"error,omitempty"`
}

// CreateResponse represents a create operation response.
type CreateResponse struct {
	ID           interface{} `json:"id"`
	AffectedRows int         `json:"affected_rows"`
	Success      bool        `json:"success"`
	Message      string      `json:"message,omitempty"`
}

// UpdateResponse represents an update operation response.
type UpdateResponse struct {
	AffectedRows int    `json:"affected_rows"`
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
}

// DeleteResponse represents a delete operation response.
type DeleteResponse struct {
	AffectedRows int    `json:"affected_rows"`
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
}

// PaginatedResponse represents a paginated result set.
type PaginatedResponse struct {
	Data       []map[string]interface{} `json:"data"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"per_page"`
	Total      int                      `json:"total"`
	TotalPages int                      `json:"total_pages"`
}

// ── Schema types ────────────────────────────────────────────────

// TableSchema represents table schema information.
type TableSchema struct {
	Name       string       `json:"name"`
	Table      string       `json:"table"`
	Columns    []ColumnInfo `json:"columns"`
	PrimaryKey *string      `json:"primary_key,omitempty"`
	RowCount   *int         `json:"row_count,omitempty"`
}

// ColumnInfo represents column information.
type ColumnInfo struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Nullable bool        `json:"nullable"`
	Key      string      `json:"key,omitempty"`
	Default  interface{} `json:"default,omitempty"`
	Extra    string      `json:"extra,omitempty"`
}

// ── Storage types ───────────────────────────────────────────────

// StorageBucket represents a storage bucket.
type StorageBucket struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Public           bool     `json:"public"`
	FileSizeLimit    *int64   `json:"file_size_limit,omitempty"`
	AllowedMimeTypes []string `json:"allowed_mime_types,omitempty"`
	CreatedAt        string   `json:"created_at,omitempty"`
	ObjectCount      int      `json:"object_count"`
	TotalSize        int64    `json:"total_size"`
}

// StorageFile represents file information in a bucket.
type StorageFile struct {
	ID        string                 `json:"id"`
	BucketID  string                 `json:"bucket_id"`
	Name      string                 `json:"name"`
	Path      string                 `json:"path"`
	MimeType  string                 `json:"mime_type,omitempty"`
	Size      int64                  `json:"size"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt string                 `json:"created_at,omitempty"`
	PublicURL string                 `json:"public_url,omitempty"`

	// Legacy fields for backward compat
	Key          string  `json:"key,omitempty"`
	LastModified string  `json:"last_modified,omitempty"`
	ContentType  *string `json:"content_type,omitempty"`
	ETag         *string `json:"etag,omitempty"`
}

// SizeMB returns the file size in megabytes.
func (f *StorageFile) SizeMB() float64 {
	return float64(f.Size) / (1024 * 1024)
}

// SizeGB returns the file size in gigabytes.
func (f *StorageFile) SizeGB() float64 {
	return float64(f.Size) / (1024 * 1024 * 1024)
}

// StorageQuota represents storage quota / statistics information.
type StorageQuota struct {
	TotalFiles     int                `json:"total_files"`
	TotalSizeBytes int64              `json:"total_size_bytes"`
	TotalSizeGB    float64            `json:"total_size_gb"`
	FileTypes      map[string]float64 `json:"file_types,omitempty"`

	// Legacy fields
	StorageQuotaGB        float64 `json:"storage_quota_gb,omitempty"`
	StorageUsedGB         float64 `json:"storage_used_gb,omitempty"`
	StorageExpansionGB    float64 `json:"storage_expansion_gb,omitempty"`
	StorageAvailableGB    float64 `json:"storage_available_gb,omitempty"`
	UsagePercentage       float64 `json:"usage_percentage,omitempty"`
	CanExpandStorage      bool    `json:"can_expand_storage,omitempty"`
	IsEnterprise          bool    `json:"is_enterprise,omitempty"`
	PlanName              string  `json:"plan_name,omitempty"`
	StorageQuotaBytes     int64   `json:"-"`
	StorageUsedBytes      int64   `json:"-"`
	StorageAvailableBytes int64   `json:"-"`
}

// UnmarshalJSON implements custom unmarshaling for StorageQuota.
func (sq *StorageQuota) UnmarshalJSON(data []byte) error {
	type Alias StorageQuota
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(sq),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	sq.StorageQuotaBytes = int64(sq.StorageQuotaGB * 1024 * 1024 * 1024)
	sq.StorageUsedBytes = int64(sq.StorageUsedGB * 1024 * 1024 * 1024)
	sq.StorageAvailableBytes = int64(sq.StorageAvailableGB * 1024 * 1024 * 1024)

	return nil
}

// FileUploadResult represents file upload result (legacy).
type FileUploadResult struct {
	Key     string `json:"key"`
	Size    int64  `json:"size"`
	URL     string `json:"url"`
	Success bool   `json:"success"`
}

// ── Column definition (schema) ──────────────────────────────────

// ColumnDefinition represents a column when creating/altering tables.
type ColumnDefinition struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	AutoIncrement bool   `json:"auto_increment,omitempty"`
	Unique        bool   `json:"unique,omitempty"`
	Nullable      bool   `json:"nullable,omitempty"`
	Default       string `json:"default,omitempty"`
}
