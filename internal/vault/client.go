package vault

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
)

// Client wraps the Vault API client with additional functionality.
type Client struct {
	config    *Config
	api       *api.Client
	logger    *slog.Logger
	mu        sync.RWMutex
	token     string
	tokenExp  time.Time
	cancelCtx context.Context
	cancel    context.CancelFunc
}

// NewClient creates a new Vault client with the given configuration.
func NewClient(cfg *Config, logger *slog.Logger) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("vault config is required")
	}

	if !cfg.Enabled {
		return nil, fmt.Errorf("vault is not enabled")
	}

	if cfg.Address == "" {
		return nil, fmt.Errorf("vault address is required")
	}

	// Create Vault API config
	apiCfg := api.DefaultConfig()
	apiCfg.Address = cfg.Address

	// Configure TLS
	if cfg.TLSSkipVerify {
		tlsConfig := &tls.Config{InsecureSkipVerify: true} //nolint:gosec // User explicitly requested skip verify
		apiCfg.HttpClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	if cfg.CACert != "" {
		if err := apiCfg.ConfigureTLS(&api.TLSConfig{
			CACert: cfg.CACert,
		}); err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
	}

	// Create the Vault client
	apiClient, err := api.NewClient(apiCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// Set namespace if provided (Enterprise feature)
	if cfg.Namespace != "" {
		apiClient.SetNamespace(cfg.Namespace)
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config:    cfg,
		api:       apiClient,
		logger:    logger.With("component", "vault-client"),
		cancelCtx: ctx,
		cancel:    cancel,
	}

	return client, nil
}

// Authenticate authenticates to Vault using the configured method.
func (c *Client) Authenticate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.authenticateUnlocked(ctx)
}

// authenticateUnlocked performs authentication without acquiring the lock.
// Caller must hold the write lock.
func (c *Client) authenticateUnlocked(ctx context.Context) error {
	switch c.config.AuthMethod {
	case AuthMethodKubernetes:
		return c.authenticateKubernetes(ctx)
	case AuthMethodToken:
		return c.authenticateToken(ctx)
	default:
		return fmt.Errorf("unsupported auth method: %s", c.config.AuthMethod)
	}
}

// authenticateKubernetes authenticates using Kubernetes service account.
func (c *Client) authenticateKubernetes(ctx context.Context) error {
	// Read the service account token
	jwt, err := os.ReadFile(c.config.TokenPath)
	if err != nil {
		return fmt.Errorf("failed to read service account token: %w", err)
	}

	// Authenticate to Vault
	data := map[string]interface{}{
		"role": c.config.Role,
		"jwt":  string(jwt),
	}

	resp, err := c.api.Logical().WriteWithContext(ctx, "auth/kubernetes/login", data)
	if err != nil {
		return fmt.Errorf("failed to authenticate with kubernetes: %w", err)
	}

	if resp == nil || resp.Auth == nil {
		return fmt.Errorf("no auth response from vault")
	}

	c.token = resp.Auth.ClientToken
	c.api.SetToken(c.token)

	// Calculate token expiration
	if resp.Auth.LeaseDuration > 0 {
		c.tokenExp = time.Now().Add(time.Duration(resp.Auth.LeaseDuration) * time.Second)
	}

	c.logger.Info("authenticated to vault",
		"auth_method", AuthMethodKubernetes,
		"role", c.config.Role,
		"lease_duration", resp.Auth.LeaseDuration,
	)

	return nil
}

// authenticateToken sets a static token (for development).
func (c *Client) authenticateToken(_ context.Context) error {
	if c.config.Token == "" {
		return fmt.Errorf("vault token is required for token auth method")
	}

	c.token = c.config.Token
	c.api.SetToken(c.token)

	c.logger.Info("authenticated to vault", "auth_method", AuthMethodToken)

	return nil
}

// GetSecret retrieves a secret from Vault.
func (c *Client) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	// Check if we need to re-authenticate using double-checked locking
	// to prevent multiple concurrent re-authentication attempts
	if c.needsReauthSafe() {
		c.mu.Lock()
		// Double-check after acquiring write lock
		if c.needsReauth() {
			// Perform authentication while holding the lock
			if err := c.authenticateUnlocked(ctx); err != nil {
				c.mu.Unlock()
				return nil, fmt.Errorf("failed to re-authenticate: %w", err)
			}
		}
		c.mu.Unlock()
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Construct the full path for KV v2
	fullPath := fmt.Sprintf("%s/data/%s", c.config.SecretMountPath, path)

	c.logger.Debug("fetching secret", "path", fullPath)

	secret, err := c.api.Logical().ReadWithContext(ctx, fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret at %s: %w", fullPath, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no secret found at %s", fullPath)
	}

	// For KV v2, the actual data is nested under "data"
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected secret format at %s", fullPath)
	}

	c.logger.Debug("fetched secret", "path", fullPath, "keys", len(data))

	return data, nil
}

// GetSecretString retrieves a specific string value from a secret.
func (c *Client) GetSecretString(ctx context.Context, path, key string) (string, error) {
	data, err := c.GetSecret(ctx, path)
	if err != nil {
		return "", err
	}

	value, ok := data[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret at %s", key, path)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value for key %s is not a string", key)
	}

	return strValue, nil
}

// needsReauthSafe checks if the token needs to be renewed (thread-safe).
func (c *Client) needsReauthSafe() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.needsReauth()
}

// needsReauth checks if the token needs to be renewed.
// Note: caller must hold at least a read lock.
func (c *Client) needsReauth() bool {
	if c.token == "" {
		return true
	}

	if c.config.AuthMethod == AuthMethodToken {
		return false // Static tokens don't need renewal
	}

	// Renew if within 10% of expiration
	if !c.tokenExp.IsZero() {
		buffer := c.config.TokenRenewalInterval / 10
		return time.Now().Add(buffer).After(c.tokenExp)
	}

	return false
}

// StartTokenRenewal starts a background goroutine to renew the token.
func (c *Client) StartTokenRenewal() {
	if c.config.AuthMethod == AuthMethodToken {
		return // Static tokens don't need renewal
	}

	go func() {
		ticker := time.NewTicker(c.config.TokenRenewalInterval)
		defer ticker.Stop()

		for {
			select {
			case <-c.cancelCtx.Done():
				c.logger.Info("stopping token renewal")
				return
			case <-ticker.C:
				if err := c.renewToken(); err != nil {
					c.logger.Error("failed to renew token, will re-authenticate", "error", err)
					if err := c.Authenticate(context.Background()); err != nil {
						c.logger.Error("failed to re-authenticate", "error", err)
					}
				}
			}
		}
	}()

	c.logger.Info("started token renewal", "interval", c.config.TokenRenewalInterval)
}

// renewToken renews the current Vault token.
func (c *Client) renewToken() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	resp, err := c.api.Auth().Token().RenewSelfWithContext(c.cancelCtx, 0)
	if err != nil {
		return fmt.Errorf("failed to renew token: %w", err)
	}

	if resp.Auth != nil && resp.Auth.LeaseDuration > 0 {
		c.tokenExp = time.Now().Add(time.Duration(resp.Auth.LeaseDuration) * time.Second)
	}

	c.logger.Debug("renewed vault token", "new_expiration", c.tokenExp)

	return nil
}

// HealthCheck checks if Vault is accessible and authenticated.
func (c *Client) HealthCheck(ctx context.Context) error {
	health, err := c.api.Sys().HealthWithContext(ctx)
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}

	if !health.Initialized {
		return fmt.Errorf("vault is not initialized")
	}

	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	return nil
}

// Close stops background goroutines and cleans up resources.
func (c *Client) Close() error {
	c.cancel()
	c.logger.Info("vault client closed")
	return nil
}
