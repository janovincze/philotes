package vault

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "vault disabled",
			config: &Config{
				Enabled: false,
			},
			wantErr: true,
		},
		{
			name: "missing address",
			config: &Config{
				Enabled: true,
				Address: "",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &Config{
				Enabled:    true,
				Address:    "http://localhost:8200",
				AuthMethod: AuthMethodToken,
				Token:      "test-token",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestClient_AuthenticateToken(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "valid token",
			token:   "test-token",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Enabled:    true,
				Address:    "http://localhost:8200",
				AuthMethod: AuthMethodToken,
				Token:      tt.token,
			}

			client, err := NewClient(cfg, logger)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			defer client.Close()

			err = client.Authenticate(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_GetSecret(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create a mock Vault server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/secret/data/test/secret":
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"data": map[string]interface{}{
						"password": "test-password",
						"username": "test-user",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/v1/secret/data/missing":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &Config{
		Enabled:         true,
		Address:         server.URL,
		AuthMethod:      AuthMethodToken,
		Token:           "test-token",
		SecretMountPath: "secret",
	}

	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	if err := client.Authenticate(context.Background()); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	tests := []struct {
		name     string
		path     string
		wantKeys []string
		wantErr  bool
	}{
		{
			name:     "existing secret",
			path:     "test/secret",
			wantKeys: []string{"password", "username"},
			wantErr:  false,
		},
		{
			name:    "missing secret",
			path:    "missing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := client.GetSecret(context.Background(), tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for _, key := range tt.wantKeys {
					if _, ok := data[key]; !ok {
						t.Errorf("GetSecret() missing key %s", key)
					}
				}
			}
		})
	}
}

func TestClient_GetSecretString(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"password": "secret-password",
					"count":    123, // non-string value
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &Config{
		Enabled:         true,
		Address:         server.URL,
		AuthMethod:      AuthMethodToken,
		Token:           "test-token",
		SecretMountPath: "secret",
	}

	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	if err := client.Authenticate(context.Background()); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr bool
	}{
		{
			name:    "existing string key",
			key:     "password",
			want:    "secret-password",
			wantErr: false,
		},
		{
			name:    "missing key",
			key:     "missing",
			wantErr: true,
		},
		{
			name:    "non-string key",
			key:     "count",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.GetSecretString(context.Background(), "test", tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecretString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("GetSecretString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvSecretProvider(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	provider := NewEnvSecretProvider(logger)
	ctx := context.Background()

	t.Run("GetDatabasePassword", func(t *testing.T) {
		// Test missing env var
		os.Unsetenv("PHILOTES_DB_PASSWORD")
		_, err := provider.GetDatabasePassword(ctx)
		if err == nil {
			t.Error("expected error for missing env var")
		}

		// Test with env var set
		os.Setenv("PHILOTES_DB_PASSWORD", "test-password")
		defer os.Unsetenv("PHILOTES_DB_PASSWORD")

		password, err := provider.GetDatabasePassword(ctx)
		if err != nil {
			t.Errorf("GetDatabasePassword() error = %v", err)
		}
		if password != "test-password" {
			t.Errorf("GetDatabasePassword() = %v, want test-password", password)
		}
	})

	t.Run("GetSourcePassword", func(t *testing.T) {
		os.Unsetenv("PHILOTES_CDC_SOURCE_PASSWORD")
		_, err := provider.GetSourcePassword(ctx)
		if err == nil {
			t.Error("expected error for missing env var")
		}

		os.Setenv("PHILOTES_CDC_SOURCE_PASSWORD", "source-password")
		defer os.Unsetenv("PHILOTES_CDC_SOURCE_PASSWORD")

		password, err := provider.GetSourcePassword(ctx)
		if err != nil {
			t.Errorf("GetSourcePassword() error = %v", err)
		}
		if password != "source-password" {
			t.Errorf("GetSourcePassword() = %v, want source-password", password)
		}
	})

	t.Run("GetStorageCredentials", func(t *testing.T) {
		os.Unsetenv("PHILOTES_STORAGE_ACCESS_KEY")
		os.Unsetenv("PHILOTES_STORAGE_SECRET_KEY")

		_, _, err := provider.GetStorageCredentials(ctx)
		if err == nil {
			t.Error("expected error for missing env vars")
		}

		os.Setenv("PHILOTES_STORAGE_ACCESS_KEY", "access-key")
		os.Setenv("PHILOTES_STORAGE_SECRET_KEY", "secret-key")
		defer os.Unsetenv("PHILOTES_STORAGE_ACCESS_KEY")
		defer os.Unsetenv("PHILOTES_STORAGE_SECRET_KEY")

		ak, sk, err := provider.GetStorageCredentials(ctx)
		if err != nil {
			t.Errorf("GetStorageCredentials() error = %v", err)
		}
		if ak != "access-key" {
			t.Errorf("GetStorageCredentials() access_key = %v, want access-key", ak)
		}
		if sk != "secret-key" {
			t.Errorf("GetStorageCredentials() secret_key = %v, want secret-key", sk)
		}
	})
}

func TestVaultSecretProvider_Caching(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"password": "cached-password",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &Config{
		Enabled:               true,
		Address:               server.URL,
		AuthMethod:            AuthMethodToken,
		Token:                 "test-token",
		SecretMountPath:       "secret",
		SecretRefreshInterval: time.Hour, // Long interval to test caching
		SecretPaths: SecretPaths{
			DatabaseBuffer: "db/buffer",
		},
	}

	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	if err := client.Authenticate(context.Background()); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	provider := NewVaultSecretProvider(client, cfg.SecretPaths, cfg.SecretRefreshInterval, logger)
	defer provider.Close()

	ctx := context.Background()

	// First call should fetch from Vault
	_, err = provider.GetDatabasePassword(ctx)
	if err != nil {
		t.Fatalf("GetDatabasePassword() error = %v", err)
	}

	initialCallCount := callCount

	// Second call should use cache
	_, err = provider.GetDatabasePassword(ctx)
	if err != nil {
		t.Fatalf("GetDatabasePassword() error = %v", err)
	}

	if callCount != initialCallCount {
		t.Errorf("Expected cached result, but made %d additional calls", callCount-initialCallCount)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Enabled {
		t.Error("default config should have Enabled=false")
	}
	if cfg.AuthMethod != AuthMethodKubernetes {
		t.Errorf("default AuthMethod = %v, want %v", cfg.AuthMethod, AuthMethodKubernetes)
	}
	if cfg.Role != "philotes" {
		t.Errorf("default Role = %v, want philotes", cfg.Role)
	}
	if cfg.TokenPath != DefaultTokenPath {
		t.Errorf("default TokenPath = %v, want %v", cfg.TokenPath, DefaultTokenPath)
	}
	if !cfg.FallbackToEnv {
		t.Error("default config should have FallbackToEnv=true")
	}
	if cfg.SecretMountPath != "secret" {
		t.Errorf("default SecretMountPath = %v, want secret", cfg.SecretMountPath)
	}
}

func TestNewSecretProvider_NilConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	_, err := NewSecretProvider(context.Background(), nil, logger)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestNewSecretProvider_Disabled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := &Config{
		Enabled: false,
	}

	provider, err := NewSecretProvider(context.Background(), cfg, logger)
	if err != nil {
		t.Errorf("NewSecretProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("expected non-nil provider")
	}

	// Should return EnvSecretProvider
	_, ok := provider.(*EnvSecretProvider)
	if !ok {
		t.Error("expected EnvSecretProvider when Vault is disabled")
	}
}
