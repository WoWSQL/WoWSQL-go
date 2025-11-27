package WOWSQL

import "encoding/json"

// QueryResponse represents a query response
type QueryResponse struct {
	Data  []map[string]interface{} `json:"data"`
	Count int                      `json:"count"`
	Total *int                     `json:"total,omitempty"`
	Error *string                  `json:"error,omitempty"`
}

// CreateResponse represents a create operation response
type CreateResponse struct {
	ID           interface{} `json:"id"`
	AffectedRows int         `json:"affected_rows"`
	Success      bool        `json:"success"`
}

// UpdateResponse represents an update operation response
type UpdateResponse struct {
	AffectedRows int  `json:"affected_rows"`
	Success      bool `json:"success"`
}

// DeleteResponse represents a delete operation response
type DeleteResponse struct {
	AffectedRows int  `json:"affected_rows"`
	Success      bool `json:"success"`
}

// TableSchema represents table schema information
type TableSchema struct {
	Name       string       `json:"name"`
	Columns    []ColumnInfo `json:"columns"`
	PrimaryKey *string      `json:"primary_key,omitempty"`
	RowCount   *int         `json:"row_count,omitempty"`
}

// ColumnInfo represents column information
type ColumnInfo struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Nullable bool        `json:"nullable"`
	Default  interface{} `json:"default,omitempty"`
}

// StorageQuota represents storage quota information
type StorageQuota struct {
	StorageQuotaGB       float64 `json:"storage_quota_gb"`
	StorageUsedGB        float64 `json:"storage_used_gb"`
	StorageExpansionGB   float64 `json:"storage_expansion_gb"`
	StorageAvailableGB   float64 `json:"storage_available_gb"`
	UsagePercentage      float64 `json:"usage_percentage"`
	CanExpandStorage     bool    `json:"can_expand_storage"`
	IsEnterprise         bool    `json:"is_enterprise"`
	PlanName             string  `json:"plan_name"`
	StorageQuotaBytes    int64   `json:"-"`
	StorageUsedBytes     int64   `json:"-"`
	StorageAvailableBytes int64   `json:"-"`
}

// UnmarshalJSON implements custom unmarshaling for StorageQuota
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

	// Calculate byte values
	sq.StorageQuotaBytes = int64(sq.StorageQuotaGB * 1024 * 1024 * 1024)
	sq.StorageUsedBytes = int64(sq.StorageUsedGB * 1024 * 1024 * 1024)
	sq.StorageAvailableBytes = int64(sq.StorageAvailableGB * 1024 * 1024 * 1024)

	return nil
}

// StorageFile represents file information
type StorageFile struct {
	Key          string  `json:"key"`
	Size         int64   `json:"size"`
	LastModified string  `json:"last_modified"`
	ContentType  *string `json:"content_type,omitempty"`
	ETag         *string `json:"etag,omitempty"`
}

// FileUploadResult represents file upload result
type FileUploadResult struct {
	Key     string `json:"key"`
	Size    int64  `json:"size"`
	URL     string `json:"url"`
	Success bool   `json:"success"`
}

