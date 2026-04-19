package WOWSQL

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ── Storage client options ──────────────────────────────────────

// StorageClientOption configures the StorageClient.
type StorageClientOption func(*storageClientConfig)

type storageClientConfig struct {
	baseDomain string
	secure     bool
	timeout    time.Duration
	verifySSL  bool
}

// StorageWithBaseDomain sets the base domain.
func StorageWithBaseDomain(domain string) StorageClientOption {
	return func(c *storageClientConfig) { c.baseDomain = domain }
}

// StorageWithSecure toggles HTTPS.
func StorageWithSecure(secure bool) StorageClientOption {
	return func(c *storageClientConfig) { c.secure = secure }
}

// StorageWithTimeout sets the HTTP timeout.
func StorageWithTimeout(d time.Duration) StorageClientOption {
	return func(c *storageClientConfig) { c.timeout = d }
}

// StorageWithVerifySSL enables/disables SSL verification.
func StorageWithVerifySSL(verify bool) StorageClientOption {
	return func(c *storageClientConfig) { c.verifySSL = verify }
}

// ── Bucket create options ───────────────────────────────────────

// BucketOption configures a CreateBucket call.
type BucketOption func(map[string]interface{})

// BucketPublic sets the bucket public flag.
func BucketPublic(public bool) BucketOption {
	return func(m map[string]interface{}) { m["public"] = public }
}

// BucketFileSizeLimit sets the maximum file size in bytes.
func BucketFileSizeLimit(limit int64) BucketOption {
	return func(m map[string]interface{}) { m["file_size_limit"] = limit }
}

// BucketAllowedMimeTypes restricts allowed MIME types.
func BucketAllowedMimeTypes(types []string) BucketOption {
	return func(m map[string]interface{}) { m["allowed_mime_types"] = types }
}

// ── Upload options ──────────────────────────────────────────────

// UploadOption configures an Upload call.
type UploadOption func(*uploadConfig)

type uploadConfig struct {
	path     string
	fileName string
}

// UploadPath sets the file path inside the bucket.
func UploadPath(path string) UploadOption {
	return func(c *uploadConfig) { c.path = path }
}

// UploadFileName overrides the file name.
func UploadFileName(name string) UploadOption {
	return func(c *uploadConfig) { c.fileName = name }
}

// ── List files options ──────────────────────────────────────────

// ListFilesOption configures a ListFiles call.
type ListFilesOption func(url.Values)

// ListFilesPrefix filters by path prefix.
func ListFilesPrefix(prefix string) ListFilesOption {
	return func(v url.Values) { v.Set("prefix", prefix) }
}

// ListFilesLimit sets the max number of files returned.
func ListFilesLimit(limit int) ListFilesOption {
	return func(v url.Values) { v.Set("limit", fmt.Sprintf("%d", limit)) }
}

// ListFilesOffset sets the offset for pagination.
func ListFilesOffset(offset int) ListFilesOption {
	return func(v url.Values) { v.Set("offset", fmt.Sprintf("%d", offset)) }
}

// ── StorageClient ───────────────────────────────────────────────

// StorageClient provides PostgreSQL-native file storage.
type StorageClient struct {
	baseURL     string
	projectSlug string
	apiKey      string
	httpClient  *http.Client
}

// NewStorageClient creates a new storage client.
//
// projectURL can be a slug, domain, or full URL.
func NewStorageClient(projectURL, apiKey string, opts ...StorageClientOption) *StorageClient {
	cfg := &storageClientConfig{
		baseDomain: "wowsqlconnect.com",
		secure:     true,
		timeout:    60 * time.Second,
		verifySSL:  true,
	}
	for _, o := range opts {
		o(cfg)
	}

	baseURL, slug := buildStorageURLs(projectURL, cfg.baseDomain, cfg.secure)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !cfg.verifySSL {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &StorageClient{
		baseURL:     baseURL,
		projectSlug: slug,
		apiKey:      apiKey,
		httpClient: &http.Client{
			Timeout:   cfg.timeout,
			Transport: transport,
		},
	}
}

// ── Bucket methods ──────────────────────────────────────────────

// CreateBucket creates a new storage bucket.
func (s *StorageClient) CreateBucket(name string, opts ...BucketOption) (*StorageBucket, error) {
	body := map[string]interface{}{"name": name}
	for _, o := range opts {
		o(body)
	}

	resp, err := s.doJSON("POST", s.bucketsPath(), body)
	if err != nil {
		return nil, err
	}

	var bucket StorageBucket
	if err := json.Unmarshal(resp, &bucket); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &bucket, nil
}

// ListBuckets lists all buckets in the project.
func (s *StorageClient) ListBuckets() ([]StorageBucket, error) {
	resp, err := s.doJSON("GET", s.bucketsPath(), nil)
	if err != nil {
		return nil, err
	}

	var buckets []StorageBucket
	if err := json.Unmarshal(resp, &buckets); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return buckets, nil
}

// GetBucket gets a specific bucket by name.
func (s *StorageClient) GetBucket(name string) (*StorageBucket, error) {
	resp, err := s.doJSON("GET", s.bucketsPath()+"/"+name, nil)
	if err != nil {
		return nil, err
	}

	var bucket StorageBucket
	if err := json.Unmarshal(resp, &bucket); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &bucket, nil
}

// UpdateBucket updates bucket settings.
func (s *StorageClient) UpdateBucket(name string, opts ...BucketOption) (*StorageBucket, error) {
	body := make(map[string]interface{})
	for _, o := range opts {
		o(body)
	}

	resp, err := s.doJSON("PATCH", s.bucketsPath()+"/"+name, body)
	if err != nil {
		return nil, err
	}

	var bucket StorageBucket
	if err := json.Unmarshal(resp, &bucket); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &bucket, nil
}

// DeleteBucket deletes a bucket and all its files.
func (s *StorageClient) DeleteBucket(name string) (map[string]interface{}, error) {
	resp, err := s.doJSON("DELETE", s.bucketsPath()+"/"+name, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return result, nil
}

// ── File methods ────────────────────────────────────────────────

// Upload uploads a file to a bucket from an io.Reader.
func (s *StorageClient) Upload(bucketName string, reader io.Reader, opts ...UploadOption) (*StorageFile, error) {
	cfg := &uploadConfig{}
	for _, o := range opts {
		o(cfg)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload data: %w", err)
	}

	name := cfg.fileName
	if name == "" && cfg.path != "" {
		name = filepath.Base(cfg.path)
	}
	if name == "" {
		name = "file"
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	uploadPath := fmt.Sprintf("%s/%s/files", s.bucketsPath(), bucketName)

	// Add folder param if path contains directory separators
	if cfg.path != "" && strings.Contains(cfg.path, "/") {
		folder := cfg.path[:strings.LastIndex(cfg.path, "/")]
		uploadPath += "?folder=" + url.QueryEscape(folder)
	}

	reqURL := s.baseURL + uploadPath
	req, err := http.NewRequest("POST", reqURL, &buf)
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

	var file StorageFile
	if err := json.Unmarshal(respBody, &file); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &file, nil
}

// UploadFromPath uploads a file from a local filesystem path.
func (s *StorageClient) UploadFromPath(filePath, bucketName string, remotePath ...string) (*StorageFile, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	rp := filepath.Base(filePath)
	if len(remotePath) > 0 && remotePath[0] != "" {
		rp = remotePath[0]
	}

	return s.Upload(bucketName, f,
		UploadPath(rp),
		UploadFileName(filepath.Base(filePath)),
	)
}

// ListFiles lists files in a bucket.
func (s *StorageClient) ListFiles(bucketName string, opts ...ListFilesOption) ([]StorageFile, error) {
	params := url.Values{}
	params.Set("limit", "100")
	params.Set("offset", "0")
	for _, o := range opts {
		o(params)
	}

	path := fmt.Sprintf("%s/%s/files?%s", s.bucketsPath(), bucketName, params.Encode())
	resp, err := s.doJSON("GET", path, nil)
	if err != nil {
		return nil, err
	}

	// API may return a list or an object with files key
	var files []StorageFile
	if json.Unmarshal(resp, &files) == nil {
		return files, nil
	}

	var wrapper struct {
		Files []StorageFile `json:"files"`
		Data  []StorageFile `json:"data"`
	}
	if err := json.Unmarshal(resp, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(wrapper.Files) > 0 {
		return wrapper.Files, nil
	}
	return wrapper.Data, nil
}

// Download downloads a file and returns its binary contents.
func (s *StorageClient) Download(bucketName, filePath string) ([]byte, error) {
	reqURL := fmt.Sprintf("%s/api/v1/storage/projects/%s/files/%s/%s",
		s.baseURL, s.projectSlug, bucketName, filePath)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, &StorageError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, parseStorageError(resp.StatusCode, body)
	}

	return io.ReadAll(resp.Body)
}

// DownloadToFile downloads a file and saves it to a local path.
func (s *StorageClient) DownloadToFile(bucketName, filePath, localPath string) error {
	content, err := s.Download(bucketName, filePath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(localPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	return os.WriteFile(localPath, content, 0o644)
}

// DeleteFile deletes a file from a bucket.
func (s *StorageClient) DeleteFile(bucketName, filePath string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/v1/storage/projects/%s/files/%s/%s",
		s.projectSlug, bucketName, filePath)

	resp, err := s.doJSON("DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return result, nil
}

// ── Utilities ───────────────────────────────────────────────────

// GetPublicURL returns the public URL for a file in a public bucket.
func (s *StorageClient) GetPublicURL(bucketName, filePath string) string {
	return fmt.Sprintf("%s/api/v1/storage/projects/%s/files/%s/%s",
		s.baseURL, s.projectSlug, bucketName, filePath)
}

// GetStats returns storage statistics for the project.
func (s *StorageClient) GetStats() (*StorageQuota, error) {
	path := fmt.Sprintf("/api/v1/storage/projects/%s/stats", s.projectSlug)
	resp, err := s.doJSON("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var quota StorageQuota
	if err := json.Unmarshal(resp, &quota); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &quota, nil
}

// GetQuota is a backward-compat alias for GetStats.
func (s *StorageClient) GetQuota() (*StorageQuota, error) {
	return s.GetStats()
}

// Close releases resources.
func (s *StorageClient) Close() {
	s.httpClient.CloseIdleConnections()
}

// ── Internals ───────────────────────────────────────────────────

func (s *StorageClient) bucketsPath() string {
	return fmt.Sprintf("/api/v1/storage/projects/%s/buckets", s.projectSlug)
}

func (s *StorageClient) doJSON(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	reqURL := s.baseURL + path
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		reqURL = path
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
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

func buildStorageURLs(projectURL, baseDomain string, secure bool) (baseURL string, slug string) {
	normalized := strings.TrimSpace(projectURL)

	if strings.HasPrefix(normalized, "http://") || strings.HasPrefix(normalized, "https://") {
		baseURL = strings.TrimSuffix(normalized, "/")
		if strings.Contains(baseURL, "/api") {
			baseURL = strings.Split(baseURL, "/api")[0]
		}
		parsed, err := url.Parse(baseURL)
		if err == nil {
			host := parsed.Hostname()
			parts := strings.Split(host, ".")
			if len(parts) > 0 {
				slug = parts[0]
			}
		}
		return
	}

	protocol := "https"
	if !secure {
		protocol = "http"
	}

	if strings.Contains(normalized, "."+baseDomain) || strings.HasSuffix(normalized, baseDomain) {
		baseURL = fmt.Sprintf("%s://%s", protocol, normalized)
		slug = strings.Split(normalized, ".")[0]
	} else {
		baseURL = fmt.Sprintf("%s://%s.%s", protocol, normalized, baseDomain)
		slug = normalized
	}
	return
}

// formatBytes formats bytes to human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
