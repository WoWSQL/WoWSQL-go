package WOWSQL

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ── Models ──────────────────────────────────────────────────────

// AuthUser represents an authenticated user.
type AuthUser struct {
	ID            string                 `json:"id"`
	Email         string                 `json:"email"`
	FullName      string                 `json:"full_name,omitempty"`
	AvatarURL     string                 `json:"avatar_url,omitempty"`
	EmailVerified bool                   `json:"email_verified"`
	UserMetadata  map[string]interface{} `json:"user_metadata"`
	AppMetadata   map[string]interface{} `json:"app_metadata"`
	CreatedAt     string                 `json:"created_at,omitempty"`
}

// AuthSession represents session tokens.
type AuthSession struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// AuthResponse combines user (if available) with session tokens.
type AuthResponse struct {
	Session *AuthSession `json:"session,omitempty"`
	User    *AuthUser    `json:"user,omitempty"`
}

// ── Token storage ───────────────────────────────────────────────

// TokenStorage is the interface for persisting tokens.
type TokenStorage interface {
	GetAccessToken() string
	SetAccessToken(token string)
	GetRefreshToken() string
	SetRefreshToken(token string)
}

// MemoryTokenStorage is the default in-memory token storage.
type MemoryTokenStorage struct {
	accessToken  string
	refreshToken string
}

func (m *MemoryTokenStorage) GetAccessToken() string        { return m.accessToken }
func (m *MemoryTokenStorage) SetAccessToken(token string)   { m.accessToken = token }
func (m *MemoryTokenStorage) GetRefreshToken() string       { return m.refreshToken }
func (m *MemoryTokenStorage) SetRefreshToken(token string)  { m.refreshToken = token }

// ── Auth client options ─────────────────────────────────────────

// AuthClientOption configures the AuthClient.
type AuthClientOption func(*authClientConfig)

type authClientConfig struct {
	baseDomain   string
	secure       bool
	timeout      time.Duration
	verifySSL    bool
	tokenStorage TokenStorage
}

// AuthWithBaseDomain sets the base domain.
func AuthWithBaseDomain(domain string) AuthClientOption {
	return func(c *authClientConfig) { c.baseDomain = domain }
}

// AuthWithSecure toggles HTTPS.
func AuthWithSecure(secure bool) AuthClientOption {
	return func(c *authClientConfig) { c.secure = secure }
}

// AuthWithTimeout sets the HTTP timeout.
func AuthWithTimeout(d time.Duration) AuthClientOption {
	return func(c *authClientConfig) { c.timeout = d }
}

// AuthWithVerifySSL sets SSL certificate verification.
func AuthWithVerifySSL(verify bool) AuthClientOption {
	return func(c *authClientConfig) { c.verifySSL = verify }
}

// AuthWithTokenStorage sets a custom token storage implementation.
func AuthWithTokenStorage(storage TokenStorage) AuthClientOption {
	return func(c *authClientConfig) { c.tokenStorage = storage }
}

// ── OAuth response ──────────────────────────────────────────────

// OAuthAuthorizeResponse describes the authorize URL payload.
type OAuthAuthorizeResponse struct {
	AuthorizationURL    string `json:"authorization_url"`
	Provider            string `json:"provider"`
	RedirectURI         string `json:"redirect_uri"`
	BackendCallbackURL  string `json:"backend_callback_url,omitempty"`
	FrontendRedirectURI string `json:"frontend_redirect_uri,omitempty"`
}

// ── Update user options ─────────────────────────────────────────

// UpdateUserOption configures an UpdateUser call.
type UpdateUserOption func(map[string]interface{})

// UpdateFullName sets the full name field.
func UpdateFullName(name string) UpdateUserOption {
	return func(m map[string]interface{}) { m["full_name"] = name }
}

// UpdateAvatarURL sets the avatar URL field.
func UpdateAvatarURL(url string) UpdateUserOption {
	return func(m map[string]interface{}) { m["avatar_url"] = url }
}

// UpdateUsername sets the username field.
func UpdateUsername(username string) UpdateUserOption {
	return func(m map[string]interface{}) { m["username"] = username }
}

// UpdateUserMetadata sets the user metadata.
func UpdateUserMetadata(meta map[string]interface{}) UpdateUserOption {
	return func(m map[string]interface{}) { m["user_metadata"] = meta }
}

// ── Internal request/response types ─────────────────────────────

type signUpRequest struct {
	Email        string                 `json:"email"`
	Password     string                 `json:"password"`
	FullName     *string                `json:"full_name,omitempty"`
	UserMetadata map[string]interface{} `json:"user_metadata,omitempty"`
}

type authAPIResponse struct {
	User         *AuthUser `json:"user"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
}

// ── AuthClient ──────────────────────────────────────────────────

// AuthClient handles project-level authentication endpoints.
type AuthClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	storage    TokenStorage
}

// NewAuthClient constructs a new project auth client.
//
// projectURL can be a slug, domain, or full URL.
// apiKey should be the unified API key (anon or service).
func NewAuthClient(projectURL, apiKey string, opts ...AuthClientOption) *AuthClient {
	cfg := &authClientConfig{
		baseDomain:   "wowsqlconnect.com",
		secure:       true,
		timeout:      30 * time.Second,
		verifySSL:    true,
		tokenStorage: &MemoryTokenStorage{},
	}
	for _, o := range opts {
		o(cfg)
	}

	base := buildAuthBaseURL(projectURL, cfg.baseDomain, cfg.secure)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !cfg.verifySSL {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &AuthClient{
		baseURL: base,
		apiKey:  apiKey,
		storage: cfg.tokenStorage,
		httpClient: &http.Client{
			Timeout:   cfg.timeout,
			Transport: transport,
		},
	}
}

// ── Public API ──────────────────────────────────────────────────

// SignUp registers a new end user for the project.
func (c *AuthClient) SignUp(email, password string, options ...func(*signUpRequest)) (*AuthResponse, error) {
	payload := &signUpRequest{
		Email:    email,
		Password: password,
	}
	for _, opt := range options {
		opt(payload)
	}

	body, err := c.doRequest("POST", "/signup", payload, nil)
	if err != nil {
		return nil, err
	}

	var resp authAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse signup response: %w", err)
	}

	session := c.persistSession(&resp)
	user := normalizeUser(resp.User)

	return &AuthResponse{Session: session, User: user}, nil
}

// WithFullName sets the optional full name for SignUp.
func WithFullName(fullName string) func(*signUpRequest) {
	return func(req *signUpRequest) { req.FullName = &fullName }
}

// WithUserMetadata sets optional metadata for SignUp.
func WithUserMetadata(metadata map[string]interface{}) func(*signUpRequest) {
	return func(req *signUpRequest) { req.UserMetadata = metadata }
}

// SignIn authenticates an existing user.
func (c *AuthClient) SignIn(email, password string) (*AuthResponse, error) {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}

	body, err := c.doRequest("POST", "/login", payload, nil)
	if err != nil {
		return nil, err
	}

	var resp authAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", err)
	}

	session := c.persistSession(&resp)
	return &AuthResponse{Session: session, User: nil}, nil
}

// GetUser fetches the current user profile.
func (c *AuthClient) GetUser(tokenOverride ...string) (*AuthUser, error) {
	token := c.resolveAccessToken(tokenOverride...)
	if token == "" {
		return nil, &WOWSQLError{Message: "access token is required to fetch user profile"}
	}

	body, err := c.doRequest("GET", "/me", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w", err)
	}
	return mapToAuthUser(raw), nil
}

// GetOAuthAuthorizationURL requests the provider authorization URL.
func (c *AuthClient) GetOAuthAuthorizationURL(provider string, redirectURI ...string) (map[string]string, error) {
	if provider == "" {
		return nil, &WOWSQLError{Message: "provider is required and cannot be empty"}
	}

	path := fmt.Sprintf("/oauth/%s", strings.TrimSpace(provider))
	if len(redirectURI) > 0 && redirectURI[0] != "" {
		path += "?frontend_redirect_uri=" + url.QueryEscape(redirectURI[0])
	}

	body, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse oauth response: %w", err)
	}

	result := map[string]string{
		"authorization_url":    stringFromMap(data, "authorization_url"),
		"provider":             stringFromMap(data, "provider"),
		"backend_callback_url": stringFromMap(data, "backend_callback_url"),
		"frontend_redirect_uri": stringFromMap(data, "frontend_redirect_uri"),
	}
	return result, nil
}

// ExchangeOAuthCallback exchanges an OAuth callback code for access tokens.
func (c *AuthClient) ExchangeOAuthCallback(provider, code string, redirectURI ...string) (*AuthResponse, error) {
	payload := map[string]interface{}{
		"code": code,
	}
	if len(redirectURI) > 0 && redirectURI[0] != "" {
		payload["redirect_uri"] = redirectURI[0]
	}

	body, err := c.doRequest("POST", fmt.Sprintf("/oauth/%s/callback", provider), payload, nil)
	if err != nil {
		return nil, err
	}

	var resp authAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse oauth callback response: %w", err)
	}

	session := c.persistSession(&resp)
	user := normalizeUser(resp.User)
	return &AuthResponse{Session: session, User: user}, nil
}

// ForgotPassword requests a password reset email.
func (c *AuthClient) ForgotPassword(email string) (map[string]interface{}, error) {
	return c.postSimple("/forgot-password", map[string]interface{}{"email": email})
}

// ResetPassword resets password with a token.
func (c *AuthClient) ResetPassword(token, newPassword string) (map[string]interface{}, error) {
	return c.postSimple("/reset-password", map[string]interface{}{
		"token":        token,
		"new_password": newPassword,
	})
}

// SendOTP sends an OTP code to the user's email.
// Purpose must be "login", "signup", or "password_reset".
func (c *AuthClient) SendOTP(email, purpose string) (map[string]interface{}, error) {
	if purpose != "login" && purpose != "signup" && purpose != "password_reset" {
		return nil, fmt.Errorf("purpose must be 'login', 'signup', or 'password_reset'")
	}
	return c.postSimple("/otp/send", map[string]interface{}{
		"email":   email,
		"purpose": purpose,
	})
}

// VerifyOTP verifies OTP and completes authentication.
// For password_reset purpose, newPassword is required.
func (c *AuthClient) VerifyOTP(email, otp, purpose string, newPassword ...string) (*AuthResponse, error) {
	if purpose != "login" && purpose != "signup" && purpose != "password_reset" {
		return nil, fmt.Errorf("purpose must be 'login', 'signup', or 'password_reset'")
	}

	payload := map[string]interface{}{
		"email":   email,
		"otp":     otp,
		"purpose": purpose,
	}

	if purpose == "password_reset" {
		if len(newPassword) == 0 || newPassword[0] == "" {
			return nil, fmt.Errorf("newPassword is required for password_reset purpose")
		}
		payload["new_password"] = newPassword[0]
	} else if len(newPassword) > 0 && newPassword[0] != "" {
		payload["new_password"] = newPassword[0]
	}

	body, err := c.doRequest("POST", "/otp/verify", payload, nil)
	if err != nil {
		return nil, err
	}

	if purpose == "password_reset" {
		return &AuthResponse{Session: &AuthSession{}, User: nil}, nil
	}

	var resp authAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse verify OTP response: %w", err)
	}

	session := c.persistSession(&resp)
	user := normalizeUser(resp.User)
	return &AuthResponse{Session: session, User: user}, nil
}

// SendMagicLink sends a magic link to the user's email.
// Purpose must be "login", "signup", or "email_verification".
func (c *AuthClient) SendMagicLink(email, purpose string) (map[string]interface{}, error) {
	if purpose != "login" && purpose != "signup" && purpose != "email_verification" {
		return nil, fmt.Errorf("purpose must be 'login', 'signup', or 'email_verification'")
	}
	return c.postSimple("/magic-link/send", map[string]interface{}{
		"email":   email,
		"purpose": purpose,
	})
}

// VerifyEmail verifies an email using a token.
func (c *AuthClient) VerifyEmail(token string) (map[string]interface{}, error) {
	return c.postSimple("/verify-email", map[string]interface{}{"token": token})
}

// ResendVerification resends the verification email.
func (c *AuthClient) ResendVerification(email string) (map[string]interface{}, error) {
	return c.postSimple("/resend-verification", map[string]interface{}{"email": email})
}

// Logout invalidates the current session.
func (c *AuthClient) Logout(tokenOverride ...string) (map[string]interface{}, error) {
	token := c.resolveAccessToken(tokenOverride...)
	if token == "" {
		return nil, &WOWSQLError{Message: "access token is required. Call SignIn first."}
	}

	body, err := c.doRequest("POST", "/logout", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if err != nil {
		return nil, err
	}

	c.ClearSession()

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]interface{}{"success": true}, nil
	}
	return result, nil
}

// RefreshToken exchanges a refresh token for new access + refresh tokens.
func (c *AuthClient) RefreshToken(refreshTokenOverride ...string) (*AuthResponse, error) {
	token := c.storage.GetRefreshToken()
	if len(refreshTokenOverride) > 0 && refreshTokenOverride[0] != "" {
		token = refreshTokenOverride[0]
	}
	if token == "" {
		return nil, &WOWSQLError{Message: "refresh token is required. Call SignIn first."}
	}

	body, err := c.doRequest("POST", "/refresh-token", map[string]interface{}{
		"refresh_token": token,
	}, nil)
	if err != nil {
		return nil, err
	}

	var resp authAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	session := c.persistSession(&resp)
	return &AuthResponse{Session: session, User: nil}, nil
}

// ChangePassword changes the authenticated user's password.
func (c *AuthClient) ChangePassword(currentPassword, newPassword string, tokenOverride ...string) (map[string]interface{}, error) {
	token := c.resolveAccessToken(tokenOverride...)
	if token == "" {
		return nil, &WOWSQLError{Message: "access token is required. Call SignIn first."}
	}

	body, err := c.doRequest("POST", "/change-password", map[string]interface{}{
		"current_password": currentPassword,
		"new_password":     newPassword,
	}, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]interface{}{"success": true}, nil
	}
	return result, nil
}

// UpdateUser updates the authenticated user's profile.
func (c *AuthClient) UpdateUser(opts ...UpdateUserOption) (*AuthUser, error) {
	token := c.resolveAccessToken()
	if token == "" {
		return nil, &WOWSQLError{Message: "access token is required. Call SignIn first."}
	}

	payload := make(map[string]interface{})
	for _, opt := range opts {
		opt(payload)
	}
	if len(payload) == 0 {
		return nil, &WOWSQLError{Message: "at least one field to update is required"}
	}

	body, err := c.doRequest("PATCH", "/me", payload, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse update user response: %w", err)
	}
	return mapToAuthUser(raw), nil
}

// ── Session management ──────────────────────────────────────────

// GetSession returns the currently stored tokens.
func (c *AuthClient) GetSession() *AuthSession {
	return &AuthSession{
		AccessToken:  c.storage.GetAccessToken(),
		RefreshToken: c.storage.GetRefreshToken(),
		TokenType:    "bearer",
	}
}

// SetSession stores access and refresh tokens.
func (c *AuthClient) SetSession(accessToken, refreshToken string) {
	c.storage.SetAccessToken(accessToken)
	c.storage.SetRefreshToken(refreshToken)
}

// ClearSession removes all stored tokens.
func (c *AuthClient) ClearSession() {
	c.storage.SetAccessToken("")
	c.storage.SetRefreshToken("")
}

// Close releases resources.
func (c *AuthClient) Close() {
	c.httpClient.CloseIdleConnections()
}

// ── Internals ───────────────────────────────────────────────────

func (c *AuthClient) resolveAccessToken(overrides ...string) string {
	if len(overrides) > 0 && overrides[0] != "" {
		return overrides[0]
	}
	return c.storage.GetAccessToken()
}

func (c *AuthClient) persistSession(resp *authAPIResponse) *AuthSession {
	session := &AuthSession{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
	}
	if session.TokenType == "" {
		session.TokenType = "bearer"
	}
	c.storage.SetAccessToken(session.AccessToken)
	c.storage.SetRefreshToken(session.RefreshToken)
	return session
}

func (c *AuthClient) postSimple(path string, payload interface{}) (map[string]interface{}, error) {
	body, err := c.doRequest("POST", path, payload, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return result, nil
}

func (c *AuthClient) doRequest(method, path string, body interface{}, headers map[string]string) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequest(method, reqURL, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseError(resp.StatusCode, bodyBytes)
	}

	return bodyBytes, nil
}

func buildAuthBaseURL(projectURL, baseDomain string, secure bool) string {
	if baseDomain == "" {
		baseDomain = "wowsqlconnect.com"
	}

	normalized := strings.TrimSpace(projectURL)

	if strings.HasPrefix(normalized, "http://") || strings.HasPrefix(normalized, "https://") {
		normalized = strings.TrimSuffix(normalized, "/")
		if strings.HasSuffix(normalized, "/api") {
			normalized = strings.TrimSuffix(normalized, "/api")
		}
		return normalized + "/api/auth"
	}

	protocol := "https"
	if !secure {
		protocol = "http"
	}

	if strings.Contains(normalized, "."+baseDomain) || strings.HasSuffix(normalized, baseDomain) {
		normalized = fmt.Sprintf("%s://%s", protocol, normalized)
	} else {
		normalized = fmt.Sprintf("%s://%s.%s", protocol, normalized, baseDomain)
	}

	normalized = strings.TrimSuffix(normalized, "/")
	if strings.HasSuffix(normalized, "/api") {
		normalized = strings.TrimSuffix(normalized, "/api")
	}

	return normalized + "/api/auth"
}

// ── User normalization helpers ──────────────────────────────────

func normalizeUser(user *AuthUser) *AuthUser {
	if user == nil {
		return nil
	}
	if user.UserMetadata == nil {
		user.UserMetadata = map[string]interface{}{}
	}
	if user.AppMetadata == nil {
		user.AppMetadata = map[string]interface{}{}
	}
	return user
}

func mapToAuthUser(m map[string]interface{}) *AuthUser {
	u := &AuthUser{
		ID:           stringFromMap(m, "id"),
		Email:        stringFromMap(m, "email"),
		FullName:     firstString(m, "full_name", "fullName"),
		AvatarURL:    firstString(m, "avatar_url", "avatarUrl"),
		UserMetadata: mapFromMap(m, "user_metadata", "userMetadata"),
		AppMetadata:  mapFromMap(m, "app_metadata", "appMetadata"),
		CreatedAt:    firstString(m, "created_at", "createdAt"),
	}
	if v, ok := m["email_verified"]; ok {
		u.EmailVerified = boolFromInterface(v)
	} else if v, ok := m["emailVerified"]; ok {
		u.EmailVerified = boolFromInterface(v)
	}
	return u
}

func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func firstString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if s := stringFromMap(m, k); s != "" {
			return s
		}
	}
	return ""
}

func mapFromMap(m map[string]interface{}, keys ...string) map[string]interface{} {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if mp, ok := v.(map[string]interface{}); ok {
				return mp
			}
		}
	}
	return map[string]interface{}{}
}

func boolFromInterface(v interface{}) bool {
	switch b := v.(type) {
	case bool:
		return b
	case float64:
		return b != 0
	case string:
		return b == "true" || b == "1"
	}
	return false
}
