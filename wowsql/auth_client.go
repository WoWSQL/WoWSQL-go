package WOWSQL

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AuthConfig configures the project auth client.
// UNIFIED AUTHENTICATION: Uses the same API keys (anon/service) as database operations.
type AuthConfig struct {
	ProjectURL   string
	BaseDomain   string
	Secure       bool
	Timeout      time.Duration
	// Unified API key - Anonymous Key (wowsql_anon_...) for client-side,
	// or Service Role Key (wowsql_service_...) for server-side.
	// UNIFIED AUTHENTICATION: Same key works for both auth and database operations.
	APIKey string
	// Deprecated: Use APIKey instead. Kept for backward compatibility.
	PublicAPIKey string
}

// AuthClient handles project-level authentication endpoints.
// UNIFIED AUTHENTICATION: Uses the same API keys (anon/service) as database operations.
type AuthClient struct {
	baseURL     string
	httpClient  *http.Client
	apiKey      string // Unified API key (anon or service)
	publicKey   string // Deprecated: same as apiKey, kept for backward compatibility
	accessToken string
	refreshToken string
}

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

// AuthResult combines user (if available) with session tokens.
type AuthResult struct {
	User    *AuthUser
	Session AuthSession
}

// OAuthAuthorizeResponse describes the authorize URL payload.
type OAuthAuthorizeResponse struct {
	AuthorizationURL    string `json:"authorization_url"`
	Provider            string `json:"provider"`
	RedirectURI         string `json:"redirect_uri"`
	BackendCallbackURL  string `json:"backend_callback_url,omitempty"`
	FrontendRedirectURI string `json:"frontend_redirect_uri,omitempty"`
}

type signUpRequest struct {
	Email        string                 `json:"email"`
	Password     string                 `json:"password"`
	FullName     *string                `json:"full_name,omitempty"`
	UserMetadata map[string]interface{} `json:"user_metadata,omitempty"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User         *AuthUser `json:"user"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// NewAuthClient constructs a new project auth client.
// UNIFIED AUTHENTICATION: Uses the same API keys (anon/service) as database operations.
func NewAuthClient(config AuthConfig) *AuthClient {
	base := buildAuthBaseURL(config.ProjectURL, config.BaseDomain, config.Secure)
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// UNIFIED AUTHENTICATION: Support both APIKey (new) and PublicAPIKey (deprecated)
	unifiedKey := config.APIKey
	if unifiedKey == "" {
		unifiedKey = config.PublicAPIKey
	}

	return &AuthClient{
		baseURL:   base,
		apiKey:    unifiedKey,
		publicKey: unifiedKey, // Keep for backward compatibility
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SignUp registers a new end user for the project.
func (c *AuthClient) SignUp(email, password string, options ...func(*signUpRequest)) (*AuthResult, error) {
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

	var resp authResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse signup response: %w", err)
	}

	session := AuthSession{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
	}
	c.persistSession(session)

	return &AuthResult{
		User:    resp.User,
		Session: session,
	}, nil
}

// WithFullName sets the optional full name for SignUp.
func WithFullName(fullName string) func(*signUpRequest) {
	return func(req *signUpRequest) {
		req.FullName = &fullName
	}
}

// WithUserMetadata sets optional metadata for SignUp.
func WithUserMetadata(metadata map[string]interface{}) func(*signUpRequest) {
	return func(req *signUpRequest) {
		req.UserMetadata = metadata
	}
}

// SignIn authenticates an existing user.
func (c *AuthClient) SignIn(email, password string) (*AuthResult, error) {
	payload := loginRequest{
		Email:    email,
		Password: password,
	}

	body, err := c.doRequest("POST", "/login", payload, nil)
	if err != nil {
		return nil, err
	}

	var resp loginResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", err)
	}

	session := AuthSession{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
	}
	c.persistSession(session)

	return &AuthResult{
		User:    nil,
		Session: session,
	}, nil
}

// GetUser fetches the current user profile using the stored access token.
func (c *AuthClient) GetUser(tokenOverride ...string) (*AuthUser, error) {
	token := c.accessToken
	if len(tokenOverride) > 0 && tokenOverride[0] != "" {
		token = tokenOverride[0]
	}
	if token == "" {
		return nil, &WOWSQLError{Message: "access token is required to fetch user profile"}
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	body, err := c.doRequest("GET", "/me", nil, headers)
	if err != nil {
		return nil, err
	}

	var user AuthUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w", err)
	}

	return &user, nil
}

// GetOAuthAuthorizationURL requests the provider authorization URL.
func (c *AuthClient) GetOAuthAuthorizationURL(provider, redirectURL string) (*OAuthAuthorizeResponse, error) {
	path := fmt.Sprintf("/oauth/%s?frontend_redirect_uri=%s", provider, url.QueryEscape(redirectURL))
	body, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var resp OAuthAuthorizeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse oauth response: %w", err)
	}

	return &resp, nil
}

// ExchangeOAuthCallback exchanges OAuth callback code for access tokens.
// After the user authorizes with the OAuth provider, the provider redirects
// back with a code. Call this method to exchange that code for JWT tokens.
func (c *AuthClient) ExchangeOAuthCallback(provider, code string, redirectURI *string) (*AuthResult, error) {
	payload := map[string]interface{}{
		"code": code,
	}
	if redirectURI != nil {
		payload["redirect_uri"] = *redirectURI
	}

	body, err := c.doRequest("POST", fmt.Sprintf("/oauth/%s/callback", provider), payload, nil)
	if err != nil {
		return nil, err
	}

	var resp authResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse oauth callback response: %w", err)
	}

	session := AuthSession{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
	}
	c.persistSession(session)

	return &AuthResult{
		User:    resp.User,
		Session: session,
	}, nil
}

// ForgotPassword requests a password reset email.
// Sends a password reset email to the user if they exist.
// Always returns success to prevent email enumeration.
func (c *AuthClient) ForgotPassword(email string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"email": email,
	}

	body, err := c.doRequest("POST", "/forgot-password", payload, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse forgot password response: %w", err)
	}

	return result, nil
}

// ResetPassword resets password with token.
// Validates the reset token and updates the user's password.
func (c *AuthClient) ResetPassword(token, newPassword string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"token":        token,
		"new_password": newPassword,
	}

	body, err := c.doRequest("POST", "/reset-password", payload, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse reset password response: %w", err)
	}

	return result, nil
}

// GetSession returns the currently stored tokens.
func (c *AuthClient) GetSession() AuthSession {
	return AuthSession{
		AccessToken:  c.accessToken,
		RefreshToken: c.refreshToken,
		TokenType:    "bearer",
	}
}

// SetSession overrides stored tokens.
func (c *AuthClient) SetSession(accessToken, refreshToken string) {
	c.accessToken = accessToken
	c.refreshToken = refreshToken
}

// ClearSession removes stored tokens.
func (c *AuthClient) ClearSession() {
	c.accessToken = ""
	c.refreshToken = ""
}

// SendOTP sends an OTP code to user's email.
// Supports login, signup, and password_reset purposes.
func (c *AuthClient) SendOTP(email, purpose string) (map[string]interface{}, error) {
	if purpose != "login" && purpose != "signup" && purpose != "password_reset" {
		return nil, fmt.Errorf("purpose must be 'login', 'signup', or 'password_reset'")
	}

	payload := map[string]interface{}{
		"email":   email,
		"purpose": purpose,
	}

	body, err := c.doRequest("POST", "/otp/send", payload, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse send OTP response: %w", err)
	}

	return result, nil
}

// VerifyOTP verifies OTP and completes authentication.
// For signup: Creates new user if doesn't exist
// For login: Authenticates existing user
// For password_reset: Updates password if newPassword provided
func (c *AuthClient) VerifyOTP(email, otp, purpose string, newPassword *string) (*AuthResult, error) {
	if purpose != "login" && purpose != "signup" && purpose != "password_reset" {
		return nil, fmt.Errorf("purpose must be 'login', 'signup', or 'password_reset'")
	}

	if purpose == "password_reset" && newPassword == nil {
		return nil, fmt.Errorf("newPassword is required for password_reset purpose")
	}

	payload := map[string]interface{}{
		"email":   email,
		"otp":     otp,
		"purpose": purpose,
	}
	if newPassword != nil {
		payload["new_password"] = *newPassword
	}

	body, err := c.doRequest("POST", "/otp/verify", payload, nil)
	if err != nil {
		return nil, err
	}

	if purpose == "password_reset" {
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse verify OTP response: %w", err)
		}
		return &AuthResult{
			User:    nil,
			Session: AuthSession{},
		}, nil
	}

	var resp authResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse verify OTP response: %w", err)
	}

	session := AuthSession{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
	}
	c.persistSession(session)

	return &AuthResult{
		User:    resp.User,
		Session: session,
	}, nil
}

// SendMagicLink sends a magic link to user's email.
// Supports login, signup, and email_verification purposes.
func (c *AuthClient) SendMagicLink(email, purpose string) (map[string]interface{}, error) {
	if purpose != "login" && purpose != "signup" && purpose != "email_verification" {
		return nil, fmt.Errorf("purpose must be 'login', 'signup', or 'email_verification'")
	}

	payload := map[string]interface{}{
		"email":   email,
		"purpose": purpose,
	}

	body, err := c.doRequest("POST", "/magic-link/send", payload, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse send magic link response: %w", err)
	}

	return result, nil
}

// VerifyEmail verifies email using token (from magic link or OTP verification).
func (c *AuthClient) VerifyEmail(token string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"token": token,
	}

	body, err := c.doRequest("POST", "/verify-email", payload, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse verify email response: %w", err)
	}

	return result, nil
}

// ResendVerification resends verification email.
// Always returns success to prevent email enumeration.
func (c *AuthClient) ResendVerification(email string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"email": email,
	}

	body, err := c.doRequest("POST", "/resend-verification", payload, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resend verification response: %w", err)
	}

	return result, nil
}

func (c *AuthClient) persistSession(session AuthSession) {
	c.accessToken = session.AccessToken
	c.refreshToken = session.RefreshToken
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

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// UNIFIED AUTHENTICATION: Use Authorization header (same as database operations)
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	} else if c.publicKey != "" {
		// Backward compatibility
		req.Header.Set("Authorization", "Bearer "+c.publicKey)
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
		baseDomain = "wowsql.com"
	}

	normalized := strings.TrimSpace(projectURL)
	
	// If it's already a full URL, use it as-is
	if strings.HasPrefix(normalized, "http://") || strings.HasPrefix(normalized, "https://") {
		normalized = strings.TrimSuffix(normalized, "/")
		if strings.HasSuffix(normalized, "/api") {
			normalized = strings.TrimSuffix(normalized, "/api")
		}
		return normalized + "/api/auth"
	}

	// If it already contains the base domain, don't append it again
	if strings.Contains(normalized, "."+baseDomain) || strings.HasSuffix(normalized, baseDomain) {
		protocol := "https"
		if !secure {
			protocol = "http"
		}
		normalized = fmt.Sprintf("%s://%s", protocol, normalized)
	} else {
		// Just a project slug, append domain
		protocol := "https"
		if !secure {
			protocol = "http"
		}
		normalized = fmt.Sprintf("%s://%s.%s", protocol, normalized, baseDomain)
	}

	normalized = strings.TrimSuffix(normalized, "/")
	if strings.HasSuffix(normalized, "/api") {
		normalized = strings.TrimSuffix(normalized, "/api")
	}

	return normalized + "/api/auth"
}
