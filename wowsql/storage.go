package WOWSQL

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// StorageClient represents the S3 storage client
type StorageClient struct {
	projectURL     string
	apiKey         string
	httpClient     *http.Client
	autoCheckQuota bool
}

// NewStorageClient creates a new storage client
func NewStorageClient(projectURL, apiKey string) *StorageClient {
	return &StorageClient{
		projectURL:     projectURL,
		apiKey:         apiKey,
		autoCheckQuota: true,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewStorageClientWithOptions creates a new storage client with options
func NewStorageClientWithOptions(projectURL, apiKey string, timeout time.Duration, autoCheckQuota bool) *StorageClient {
	return &StorageClient{
		projectURL:     projectURL,
		apiKey:         apiKey,
		autoCheckQuota: autoCheckQuota,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetQuota retrieves storage quota information
func (s *StorageClient) GetQuota() (*StorageQuota, error) {
	resp, err := s.doRequest("GET", "/api/v1/storage/quota", nil)
	if err != nil {
		return nil, err
	}

	var quota StorageQuota
	if err := json.Unmarshal(resp, &quota); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &quota, nil
}

// Upload uploads a file to storage
func (s *StorageClient) Upload(fileData []byte, key string, contentType string, checkQuota *bool) (*FileUploadResult, error) {
	shouldCheck := s.autoCheckQuota
	if checkQuota != nil {
		shouldCheck = *checkQuota
	}

	// Check quota if enabled
	if shouldCheck {
		quota, err := s.GetQuota()
		if err != nil {
			return nil, err
		}

		if quota.StorageAvailableBytes < int64(len(fileData)) {
			return nil, &StorageLimitExceededError{
				Message:        fmt.Sprintf("Storage limit exceeded. Need %s, but only %s available.", formatBytes(int64(len(fileData))), formatBytes(quota.StorageAvailableBytes)),
				RequiredBytes:  int64(len(fileData)),
				AvailableBytes: quota.StorageAvailableBytes,
			}
		}
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add key field
	if err := writer.WriteField("key", key); err != nil {
		return nil, fmt.Errorf("failed to write key field: %w", err)
	}

	// Add content type if provided
	if contentType != "" {
		if err := writer.WriteField("content_type", contentType); err != nil {
			return nil, fmt.Errorf("failed to write content_type field: %w", err)
		}
	}

	// Add file
	part, err := writer.CreateFormFile("file", key)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Make request
	url := s.projectURL + "/api/v1/storage/upload"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, &StorageError{Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseStorageError(resp.StatusCode, respBody)
	}

	var result FileUploadResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// Download gets a presigned URL for downloading a file
func (s *StorageClient) Download(key string, expiresIn int) (string, error) {
	url := fmt.Sprintf("/api/v1/storage/download?key=%s&expires_in=%d", key, expiresIn)
	resp, err := s.doRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.URL, nil
}

// ListFiles lists files in storage
func (s *StorageClient) ListFiles(prefix string, limit int) ([]StorageFile, error) {
	url := "/api/v1/storage/list"
	if prefix != "" || limit > 0 {
		url += "?"
		if prefix != "" {
			url += fmt.Sprintf("prefix=%s", prefix)
		}
		if limit > 0 {
			if prefix != "" {
				url += "&"
			}
			url += fmt.Sprintf("limit=%d", limit)
		}
	}

	resp, err := s.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Files []StorageFile `json:"files"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Files, nil
}

// DeleteFile deletes a single file
func (s *StorageClient) DeleteFile(key string) error {
	body := map[string]interface{}{
		"key": key,
	}

	_, err := s.doRequest("DELETE", "/api/v1/storage/delete", body)
	return err
}

// DeleteFiles deletes multiple files
func (s *StorageClient) DeleteFiles(keys []string) error {
	body := map[string]interface{}{
		"keys": keys,
	}

	_, err := s.doRequest("DELETE", "/api/v1/storage/delete-batch", body)
	return err
}

// GetFileInfo gets information about a file
func (s *StorageClient) GetFileInfo(key string) (*StorageFile, error) {
	url := fmt.Sprintf("/api/v1/storage/info?key=%s", key)
	resp, err := s.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var file StorageFile
	if err := json.Unmarshal(resp, &file); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &file, nil
}

// FileExists checks if a file exists
func (s *StorageClient) FileExists(key string) (bool, error) {
	_, err := s.GetFileInfo(key)
	if err != nil {
		if _, ok := err.(*NotFoundError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// doRequest performs an HTTP request
func (s *StorageClient) doRequest(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := s.projectURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, &StorageError{Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseStorageError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

// formatBytes formats bytes to human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

