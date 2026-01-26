package vault

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// SecretProvider defines the interface for retrieving secrets.
type SecretProvider interface {
	// GetDatabasePassword returns the buffer database password
	GetDatabasePassword(ctx context.Context) (string, error)

	// GetSourcePassword returns the source database password
	GetSourcePassword(ctx context.Context) (string, error)

	// GetStorageCredentials returns MinIO/S3 access and secret keys
	GetStorageCredentials(ctx context.Context) (accessKey, secretKey string, err error)

	// Refresh refreshes all cached secrets
	Refresh(ctx context.Context) error

	// Close cleans up resources
	Close() error
}

// VaultSecretProvider retrieves secrets from HashiCorp Vault.
type VaultSecretProvider struct {
	client      *Client
	paths       SecretPaths
	logger      *slog.Logger
	mu          sync.RWMutex
	cache       secretCache
	refreshInterval time.Duration
	cancelCtx   context.Context
	cancel      context.CancelFunc
}

type secretCache struct {
	databasePassword string
	sourcePassword   string
	storageAccessKey string
	storageSecretKey string
	lastRefresh      time.Time
}

// NewVaultSecretProvider creates a new VaultSecretProvider.
func NewVaultSecretProvider(client *Client, paths SecretPaths, refreshInterval time.Duration, logger *slog.Logger) *VaultSecretProvider {
	ctx, cancel := context.WithCancel(context.Background())
	return &VaultSecretProvider{
		client:          client,
		paths:           paths,
		logger:          logger.With("component", "vault-secrets"),
		refreshInterval: refreshInterval,
		cancelCtx:       ctx,
		cancel:          cancel,
	}
}

// GetDatabasePassword returns the buffer database password.
func (p *VaultSecretProvider) GetDatabasePassword(ctx context.Context) (string, error) {
	// Read cache under read lock
	p.mu.RLock()
	cachedPassword := p.cache.databasePassword
	needsRefresh := p.needsRefresh()
	p.mu.RUnlock()

	// Return cached value if valid
	if cachedPassword != "" && !needsRefresh {
		return cachedPassword, nil
	}

	// Fetch from Vault
	password, err := p.client.GetSecretString(ctx, p.paths.DatabaseBuffer, SecretKeyPassword)
	if err != nil {
		return "", fmt.Errorf("failed to get database password: %w", err)
	}

	// Update cache under write lock
	p.mu.Lock()
	p.cache.databasePassword = password
	p.cache.lastRefresh = time.Now()
	p.mu.Unlock()

	return password, nil
}

// GetSourcePassword returns the source database password.
func (p *VaultSecretProvider) GetSourcePassword(ctx context.Context) (string, error) {
	// Read cache under read lock
	p.mu.RLock()
	cachedPassword := p.cache.sourcePassword
	needsRefresh := p.needsRefresh()
	p.mu.RUnlock()

	// Return cached value if valid
	if cachedPassword != "" && !needsRefresh {
		return cachedPassword, nil
	}

	// Fetch from Vault
	password, err := p.client.GetSecretString(ctx, p.paths.DatabaseSource, SecretKeyPassword)
	if err != nil {
		return "", fmt.Errorf("failed to get source password: %w", err)
	}

	// Update cache under write lock
	p.mu.Lock()
	p.cache.sourcePassword = password
	p.cache.lastRefresh = time.Now()
	p.mu.Unlock()

	return password, nil
}

// GetStorageCredentials returns MinIO/S3 credentials.
func (p *VaultSecretProvider) GetStorageCredentials(ctx context.Context) (accessKey, secretKey string, err error) {
	// Read cache under read lock
	p.mu.RLock()
	cachedAccessKey := p.cache.storageAccessKey
	cachedSecretKey := p.cache.storageSecretKey
	needsRefresh := p.needsRefresh()
	p.mu.RUnlock()

	// Return cached values if valid
	if cachedAccessKey != "" && cachedSecretKey != "" && !needsRefresh {
		return cachedAccessKey, cachedSecretKey, nil
	}

	// Fetch from Vault
	data, err := p.client.GetSecret(ctx, p.paths.StorageMinio)
	if err != nil {
		return "", "", fmt.Errorf("failed to get storage credentials: %w", err)
	}

	ak, ok := data[SecretKeyAccessKey].(string)
	if !ok {
		return "", "", fmt.Errorf("access_key not found in storage secret")
	}

	sk, ok := data[SecretKeySecretKey].(string)
	if !ok {
		return "", "", fmt.Errorf("secret_key not found in storage secret")
	}

	// Update cache under write lock
	p.mu.Lock()
	p.cache.storageAccessKey = ak
	p.cache.storageSecretKey = sk
	p.cache.lastRefresh = time.Now()
	p.mu.Unlock()

	return ak, sk, nil
}

// Refresh refreshes all cached secrets.
func (p *VaultSecretProvider) Refresh(ctx context.Context) error {
	p.logger.Debug("refreshing secrets from vault")

	var errs []error

	// Refresh database password
	if dbPassword, err := p.client.GetSecretString(ctx, p.paths.DatabaseBuffer, SecretKeyPassword); err != nil {
		errs = append(errs, fmt.Errorf("database password: %w", err))
	} else {
		p.mu.Lock()
		p.cache.databasePassword = dbPassword
		p.mu.Unlock()
	}

	// Refresh source password
	if sourcePassword, err := p.client.GetSecretString(ctx, p.paths.DatabaseSource, SecretKeyPassword); err != nil {
		errs = append(errs, fmt.Errorf("source password: %w", err))
	} else {
		p.mu.Lock()
		p.cache.sourcePassword = sourcePassword
		p.mu.Unlock()
	}

	// Refresh storage credentials
	if data, err := p.client.GetSecret(ctx, p.paths.StorageMinio); err != nil {
		errs = append(errs, fmt.Errorf("storage credentials: %w", err))
	} else {
		p.mu.Lock()
		if ak, ok := data[SecretKeyAccessKey].(string); ok {
			p.cache.storageAccessKey = ak
		}
		if sk, ok := data[SecretKeySecretKey].(string); ok {
			p.cache.storageSecretKey = sk
		}
		p.mu.Unlock()
	}

	p.mu.Lock()
	p.cache.lastRefresh = time.Now()
	p.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("failed to refresh some secrets: %v", errs)
	}

	p.logger.Debug("secrets refreshed successfully")
	return nil
}

// StartRefreshLoop starts a background goroutine to periodically refresh secrets.
func (p *VaultSecretProvider) StartRefreshLoop() {
	go func() {
		ticker := time.NewTicker(p.refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-p.cancelCtx.Done():
				p.logger.Info("stopping secret refresh loop")
				return
			case <-ticker.C:
				if err := p.Refresh(p.cancelCtx); err != nil {
					p.logger.Warn("failed to refresh secrets", "error", err)
				}
			}
		}
	}()

	p.logger.Info("started secret refresh loop", "interval", p.refreshInterval)
}

// needsRefresh checks if the cache needs to be refreshed.
func (p *VaultSecretProvider) needsRefresh() bool {
	return time.Since(p.cache.lastRefresh) > p.refreshInterval
}

// Close stops the refresh loop.
func (p *VaultSecretProvider) Close() error {
	p.cancel()
	return nil
}

// EnvSecretProvider retrieves secrets from environment variables.
// This is used as a fallback when Vault is unavailable.
type EnvSecretProvider struct {
	logger *slog.Logger
}

// NewEnvSecretProvider creates a new EnvSecretProvider.
func NewEnvSecretProvider(logger *slog.Logger) *EnvSecretProvider {
	return &EnvSecretProvider{
		logger: logger.With("component", "env-secrets"),
	}
}

// GetDatabasePassword returns the database password from environment.
func (p *EnvSecretProvider) GetDatabasePassword(_ context.Context) (string, error) {
	password := os.Getenv("PHILOTES_DB_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("PHILOTES_DB_PASSWORD not set")
	}
	return password, nil
}

// GetSourcePassword returns the source database password from environment.
func (p *EnvSecretProvider) GetSourcePassword(_ context.Context) (string, error) {
	password := os.Getenv("PHILOTES_CDC_SOURCE_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("PHILOTES_CDC_SOURCE_PASSWORD not set")
	}
	return password, nil
}

// GetStorageCredentials returns storage credentials from environment.
func (p *EnvSecretProvider) GetStorageCredentials(_ context.Context) (accessKey, secretKey string, err error) {
	accessKey = os.Getenv("PHILOTES_STORAGE_ACCESS_KEY")
	if accessKey == "" {
		return "", "", fmt.Errorf("PHILOTES_STORAGE_ACCESS_KEY not set")
	}

	secretKey = os.Getenv("PHILOTES_STORAGE_SECRET_KEY")
	if secretKey == "" {
		return "", "", fmt.Errorf("PHILOTES_STORAGE_SECRET_KEY not set")
	}

	return accessKey, secretKey, nil
}

// Refresh is a no-op for environment variables.
func (p *EnvSecretProvider) Refresh(_ context.Context) error {
	p.logger.Debug("refresh called on env provider (no-op)")
	return nil
}

// Close is a no-op for environment variables.
func (p *EnvSecretProvider) Close() error {
	return nil
}

// NewSecretProvider creates an appropriate SecretProvider based on configuration.
// If Vault is enabled and accessible, it returns a VaultSecretProvider.
// Otherwise, it falls back to EnvSecretProvider if FallbackToEnv is enabled.
func NewSecretProvider(ctx context.Context, cfg *Config, logger *slog.Logger) (SecretProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("vault config is required")
	}

	if !cfg.Enabled {
		logger.Info("vault disabled, using environment secrets")
		return NewEnvSecretProvider(logger), nil
	}

	// Try to create Vault client
	client, err := NewClient(cfg, logger)
	if err != nil {
		if cfg.FallbackToEnv {
			logger.Warn("failed to create vault client, falling back to environment", "error", err)
			return NewEnvSecretProvider(logger), nil
		}
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// Authenticate to Vault
	if err := client.Authenticate(ctx); err != nil {
		if cfg.FallbackToEnv {
			logger.Warn("failed to authenticate to vault, falling back to environment", "error", err)
			return NewEnvSecretProvider(logger), nil
		}
		return nil, fmt.Errorf("failed to authenticate to vault: %w", err)
	}

	// Start token renewal
	client.StartTokenRenewal()

	// Create the provider
	provider := NewVaultSecretProvider(client, cfg.SecretPaths, cfg.SecretRefreshInterval, logger)

	// Initial secret fetch
	if err := provider.Refresh(ctx); err != nil {
		if cfg.FallbackToEnv {
			logger.Warn("failed to fetch initial secrets, falling back to environment", "error", err)
			client.Close()
			return NewEnvSecretProvider(logger), nil
		}
		return nil, fmt.Errorf("failed to fetch initial secrets: %w", err)
	}

	// Start refresh loop
	provider.StartRefreshLoop()

	logger.Info("using vault secret provider")
	return provider, nil
}
