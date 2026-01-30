package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGenerateState(t *testing.T) {
	state1, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState failed: %v", err)
	}
	state2, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState failed: %v", err)
	}

	// Should be non-empty
	if state1 == "" {
		t.Error("expected non-empty state")
	}

	// Should be unique
	if state1 == state2 {
		t.Error("expected unique states")
	}

	// Should be reasonable length
	if len(state1) < 20 {
		t.Errorf("state too short: %d chars", len(state1))
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce1, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce failed: %v", err)
	}
	nonce2, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce failed: %v", err)
	}

	// Should be non-empty
	if nonce1 == "" {
		t.Error("expected non-empty nonce")
	}

	// Should be unique
	if nonce1 == nonce2 {
		t.Error("expected unique nonces")
	}
}

func TestGenerateCodeVerifier(t *testing.T) {
	verifier, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("GenerateCodeVerifier failed: %v", err)
	}

	// Should be non-empty
	if verifier == "" {
		t.Error("expected non-empty code verifier")
	}

	// PKCE spec: verifier should be 43-128 characters
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("code verifier length %d outside valid range [43-128]", len(verifier))
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := GenerateCodeChallenge(verifier)

	// Should be non-empty
	if challenge == "" {
		t.Error("expected non-empty code challenge")
	}

	// Should be URL-safe base64
	if strings.ContainsAny(challenge, "+/=") {
		t.Error("code challenge should be URL-safe base64")
	}

	// Same verifier should produce same challenge
	challenge2 := GenerateCodeChallenge(verifier)
	if challenge != challenge2 {
		t.Error("same verifier should produce same challenge")
	}

	// Different verifier should produce different challenge
	verifier2, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("GenerateCodeVerifier failed: %v", err)
	}
	challenge3 := GenerateCodeChallenge(verifier2)
	if challenge == challenge3 {
		t.Error("different verifiers should produce different challenges")
	}
}

func TestClient_Discover(t *testing.T) {
	// Create a mock OIDC discovery server
	discoveryDoc := map[string]interface{}{
		"issuer":                 "https://example.com",
		"authorization_endpoint": "https://example.com/authorize",
		"token_endpoint":         "https://example.com/token",
		"userinfo_endpoint":      "https://example.com/userinfo",
		"jwks_uri":               "https://example.com/.well-known/jwks.json",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(discoveryDoc) //nolint:errcheck
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "client-id", []string{"openid", "profile", "email"})
	ctx := context.Background()

	config, err := client.Discover(ctx)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if config.AuthorizationEndpoint != "https://example.com/authorize" {
		t.Errorf("expected authorization endpoint, got %s", config.AuthorizationEndpoint)
	}
	if config.TokenEndpoint != "https://example.com/token" {
		t.Errorf("expected token endpoint, got %s", config.TokenEndpoint)
	}
	if config.UserInfoEndpoint != "https://example.com/userinfo" {
		t.Errorf("expected userinfo endpoint, got %s", config.UserInfoEndpoint)
	}
}

func TestClient_Discover_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json")) //nolint:errcheck
	}))
	defer server.Close()

	client := NewClient(server.URL, "client-id", []string{"openid"})
	ctx := context.Background()

	_, err := client.Discover(ctx)
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestClient_Discover_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "client-id", []string{"openid"})
	ctx := context.Background()

	_, err := client.Discover(ctx)
	if err == nil {
		t.Error("expected error for server error response")
	}
}

func TestClient_AuthorizationURL(t *testing.T) {
	// Create a mock discovery server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			discoveryDoc := map[string]interface{}{
				"issuer":                 r.Host,
				"authorization_endpoint": "https://example.com/authorize",
				"token_endpoint":         "https://example.com/token",
				"userinfo_endpoint":      "https://example.com/userinfo",
				"jwks_uri":               "https://example.com/.well-known/jwks.json",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(discoveryDoc) //nolint:errcheck
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "client-id", []string{"openid", "profile", "email"})
	ctx := context.Background()

	redirectURI := "https://app.example.com/callback"
	state := "test-state"
	nonce := "test-nonce"
	codeChallenge := "test-challenge"

	authURL, err := client.AuthorizationURL(ctx, state, nonce, codeChallenge, redirectURI)
	if err != nil {
		t.Fatalf("AuthorizationURL failed: %v", err)
	}

	// Check required parameters
	if !strings.Contains(authURL, "response_type=code") {
		t.Error("expected response_type=code")
	}
	if !strings.Contains(authURL, "client_id=client-id") {
		t.Error("expected client_id parameter")
	}
	if !strings.Contains(authURL, "redirect_uri=") {
		t.Error("expected redirect_uri parameter")
	}
	if !strings.Contains(authURL, "state=test-state") {
		t.Error("expected state parameter")
	}
	if !strings.Contains(authURL, "nonce=test-nonce") {
		t.Error("expected nonce parameter")
	}
	if !strings.Contains(authURL, "code_challenge=test-challenge") {
		t.Error("expected code_challenge parameter")
	}
	if !strings.Contains(authURL, "code_challenge_method=S256") {
		t.Error("expected code_challenge_method=S256")
	}
	if !strings.Contains(authURL, "scope=openid") {
		t.Error("expected scope parameter with openid")
	}
}

func TestClient_Exchange(t *testing.T) {
	// Create mock token endpoint
	tokenResponse := map[string]interface{}{
		"access_token":  "test-access-token",
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": "test-refresh-token",
		"id_token":      "header.payload.signature",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			discoveryDoc := map[string]interface{}{
				"issuer":                 "http://" + r.Host,
				"authorization_endpoint": "http://" + r.Host + "/authorize",
				"token_endpoint":         "http://" + r.Host + "/token",
				"userinfo_endpoint":      "http://" + r.Host + "/userinfo",
				"jwks_uri":               "http://" + r.Host + "/.well-known/jwks.json",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(discoveryDoc) //nolint:errcheck
			return
		}
		if r.URL.Path == "/token" {
			if r.Method != "POST" {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			// Verify content type
			if !strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
				http.Error(w, "invalid content type", http.StatusBadRequest)
				return
			}

			// Verify required parameters
			_ = r.ParseForm() //nolint:errcheck
			if r.Form.Get("grant_type") != "authorization_code" {
				http.Error(w, "invalid grant_type", http.StatusBadRequest)
				return
			}
			if r.Form.Get("code") == "" {
				http.Error(w, "missing code", http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tokenResponse) //nolint:errcheck
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient(server.URL, "client-id", []string{"openid"})
	ctx := context.Background()

	tokens, err := client.Exchange(ctx, "auth-code", "verifier", "https://app.example.com/callback", "client-secret")
	if err != nil {
		t.Fatalf("Exchange failed: %v", err)
	}

	if tokens.AccessToken != "test-access-token" {
		t.Errorf("expected access token, got %s", tokens.AccessToken)
	}
	if tokens.TokenType != "Bearer" {
		t.Errorf("expected token type Bearer, got %s", tokens.TokenType)
	}
	if tokens.RefreshToken != "test-refresh-token" {
		t.Errorf("expected refresh token, got %s", tokens.RefreshToken)
	}
}

func TestClient_Exchange_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			discoveryDoc := map[string]interface{}{
				"issuer":                 "http://" + r.Host,
				"authorization_endpoint": "http://" + r.Host + "/authorize",
				"token_endpoint":         "http://" + r.Host + "/token",
				"userinfo_endpoint":      "http://" + r.Host + "/userinfo",
				"jwks_uri":               "http://" + r.Host + "/.well-known/jwks.json",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(discoveryDoc) //nolint:errcheck
			return
		}
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
				"error":             "invalid_grant",
				"error_description": "The authorization code has expired",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient(server.URL, "client-id", []string{"openid"})
	ctx := context.Background()

	_, err := client.Exchange(ctx, "expired-code", "verifier", "https://app.example.com/callback", "client-secret")
	if err == nil {
		t.Error("expected error for invalid grant")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to contain status code '400', got: %v", err)
	}
}

func TestClient_ParseIDToken(t *testing.T) {
	issuer := "https://example.com"
	clientID := "client-id"
	client := NewClient(issuer, clientID, []string{"openid"})

	// Create a test ID token (JWT)
	idToken := createTestIDToken(issuer, clientID)
	nonce := "test-nonce"

	// Parse and validate
	claims, err := client.ParseIDToken(idToken, nonce)
	if err != nil {
		t.Fatalf("ParseIDToken failed: %v", err)
	}

	if claims.Subject != "user-123" {
		t.Errorf("expected subject 'user-123', got '%s'", claims.Subject)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got '%s'", claims.Email)
	}
	if claims.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", claims.Name)
	}
}

func TestClient_ParseIDToken_InvalidFormat(t *testing.T) {
	client := NewClient("https://example.com", "client-id", []string{"openid"})

	// Invalid JWT format
	_, err := client.ParseIDToken("not-a-jwt", "nonce")
	if err == nil {
		t.Error("expected error for invalid JWT format")
	}

	// Invalid base64 in payload
	_, err = client.ParseIDToken("header.!!!invalid!!!.signature", "nonce")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestClient_ParseIDToken_WrongIssuer(t *testing.T) {
	client := NewClient("https://example.com", "client-id", []string{"openid"})

	// Token with wrong issuer
	idToken := createTestIDToken("https://wrong-issuer.com", "client-id")
	_, err := client.ParseIDToken(idToken, "test-nonce")
	if err == nil {
		t.Error("expected error for wrong issuer")
	}
	if !strings.Contains(err.Error(), "invalid issuer") {
		t.Errorf("expected issuer error, got: %v", err)
	}
}

func TestClient_ParseIDToken_WrongAudience(t *testing.T) {
	client := NewClient("https://example.com", "client-id", []string{"openid"})

	// Token with wrong audience
	idToken := createTestIDToken("https://example.com", "wrong-client")
	_, err := client.ParseIDToken(idToken, "test-nonce")
	if err == nil {
		t.Error("expected error for wrong audience")
	}
	if !strings.Contains(err.Error(), "invalid audience") {
		t.Errorf("expected audience error, got: %v", err)
	}
}

func TestClient_ParseIDToken_WrongNonce(t *testing.T) {
	issuer := "https://example.com"
	clientID := "client-id"
	client := NewClient(issuer, clientID, []string{"openid"})

	idToken := createTestIDToken(issuer, clientID)
	_, err := client.ParseIDToken(idToken, "wrong-nonce")
	if err == nil {
		t.Error("expected error for wrong nonce")
	}
	if !strings.Contains(err.Error(), "invalid nonce") {
		t.Errorf("expected nonce error, got: %v", err)
	}
}

func TestClient_GetUserInfo(t *testing.T) {
	userInfo := map[string]interface{}{
		"sub":            "user-123",
		"email":          "user@example.com",
		"email_verified": true,
		"name":           "Test User",
		"given_name":     "Test",
		"family_name":    "User",
		"picture":        "https://example.com/photo.jpg",
		"groups":         []string{"admins", "users"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			// Update userinfo endpoint to match server
			discoveryDoc := map[string]interface{}{
				"issuer":                 "http://" + r.Host,
				"authorization_endpoint": "http://" + r.Host + "/authorize",
				"token_endpoint":         "http://" + r.Host + "/token",
				"userinfo_endpoint":      "http://" + r.Host + "/userinfo",
				"jwks_uri":               "http://" + r.Host + "/.well-known/jwks.json",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(discoveryDoc) //nolint:errcheck
			return
		}
		if r.URL.Path == "/userinfo" {
			// Verify authorization header
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(userInfo) //nolint:errcheck
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient(server.URL, "client-id", []string{"openid"})
	ctx := context.Background()

	info, err := client.GetUserInfo(ctx, "test-access-token")
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}

	if info.Subject != "user-123" {
		t.Errorf("expected subject 'user-123', got '%s'", info.Subject)
	}
	if info.Email != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got '%s'", info.Email)
	}
	if !info.EmailVerified {
		t.Error("expected email_verified to be true")
	}
	if info.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", info.Name)
	}
	if len(info.Groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(info.Groups))
	}
}

func TestClaimsToUserInfo(t *testing.T) {
	claims := &IDTokenClaims{
		Subject:       "user-123",
		Email:         "user@example.com",
		EmailVerified: true,
		Name:          "Test User",
		GivenName:     "Test",
		FamilyName:    "User",
		Picture:       "https://example.com/photo.jpg",
		Groups:        []string{"admins", "users"},
	}

	userInfo := ClaimsToUserInfo(claims, "groups")

	if userInfo.Subject != claims.Subject {
		t.Errorf("expected subject %s, got %s", claims.Subject, userInfo.Subject)
	}
	if userInfo.Email != claims.Email {
		t.Errorf("expected email %s, got %s", claims.Email, userInfo.Email)
	}
	if userInfo.Name != claims.Name {
		t.Errorf("expected name %s, got %s", claims.Name, userInfo.Name)
	}
	if len(userInfo.Groups) != len(claims.Groups) {
		t.Errorf("expected %d groups, got %d", len(claims.Groups), len(userInfo.Groups))
	}
}

func TestAudience_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single string",
			input:    `"client-id"`,
			expected: []string{"client-id"},
		},
		{
			name:     "array of strings",
			input:    `["client-1", "client-2"]`,
			expected: []string{"client-1", "client-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var aud Audience
			if err := json.Unmarshal([]byte(tt.input), &aud); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if len(aud) != len(tt.expected) {
				t.Errorf("expected %d audiences, got %d", len(tt.expected), len(aud))
			}
			for i, a := range tt.expected {
				if aud[i] != a {
					t.Errorf("expected audience %s at position %d, got %s", a, i, aud[i])
				}
			}
		})
	}
}

// Helper to create a test ID token (unsigned JWT for testing)
func createTestIDToken(issuer, audience string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))

	claims := map[string]interface{}{
		"iss":            issuer,
		"sub":            "user-123",
		"aud":            audience,
		"exp":            time.Now().Add(time.Hour).Unix(),
		"iat":            time.Now().Unix(),
		"nonce":          "test-nonce",
		"email":          "user@example.com",
		"email_verified": true,
		"name":           "Test User",
		"given_name":     "Test",
		"family_name":    "User",
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		// This should never happen with a static map, but handle it anyway
		return ""
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Use a fake signature for testing (real validation would need JWKS)
	signature := base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))

	return header + "." + payload + "." + signature
}
