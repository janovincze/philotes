package sshkeys

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// VaultProvider loads SSH keys from HashiCorp Vault.
// This is the recommended approach for production deployments.
// Supports token and Kubernetes authentication methods.
type VaultProvider struct {
	address    string
	secretPath string
	authMethod string
	token      string
	role       string
}

// NewVaultProvider creates a new Vault-based SSH key provider.
func NewVaultProvider(address, secretPath, authMethod, token, role string) *VaultProvider {
	return &VaultProvider{
		address:    address,
		secretPath: secretPath,
		authMethod: authMethod,
		token:      token,
		role:       role,
	}
}

// GetPrivateKey retrieves the SSH private key from Vault.
func (p *VaultProvider) GetPrivateKey(ctx *pulumi.Context) (pulumi.StringOutput, error) {
	// Get Vault token (authenticate if needed)
	token, err := p.getToken()
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("failed to authenticate with Vault: %w", err)
	}

	// Read secret from Vault
	keyContent, err := p.readSecret(token)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("failed to read SSH key from Vault: %w", err)
	}

	ctx.Log.Info("SSH key loaded from HashiCorp Vault", nil)

	// Return as secret to protect in state and logs
	return pulumi.ToSecret(pulumi.String(keyContent)).(pulumi.StringOutput), nil
}

// Source returns the source type.
func (p *VaultProvider) Source() SSHKeySource {
	return SourceVault
}

// getToken returns a Vault token, authenticating if necessary.
func (p *VaultProvider) getToken() (string, error) {
	// If token is provided directly, use it
	if p.token != "" {
		return p.token, nil
	}

	// Check environment variable
	if envToken := os.Getenv("VAULT_TOKEN"); envToken != "" {
		return envToken, nil
	}

	// Authenticate based on method
	switch p.authMethod {
	case "kubernetes":
		return p.authenticateKubernetes()
	case "token", "":
		return "", fmt.Errorf("Vault token required. Set VAULT_TOKEN env var or provide vaultToken config")
	default:
		return "", fmt.Errorf("unsupported Vault auth method: %s (supported: token, kubernetes)", p.authMethod)
	}
}

// authenticateKubernetes authenticates using Kubernetes service account JWT.
func (p *VaultProvider) authenticateKubernetes() (string, error) {
	// Read Kubernetes service account token
	jwt, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", fmt.Errorf("failed to read Kubernetes service account token: %w", err)
	}

	role := p.role
	if role == "" {
		role = "philotes"
	}

	// Authenticate with Vault
	authURL := fmt.Sprintf("%s/v1/auth/kubernetes/login", strings.TrimSuffix(p.address, "/"))
	payload := fmt.Sprintf(`{"jwt": "%s", "role": "%s"}`, string(jwt), role)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(authURL, "application/json", strings.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("Vault authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Vault authentication failed (status %d): %s", resp.StatusCode, string(body))
	}

	var authResp struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to parse Vault auth response: %w", err)
	}

	return authResp.Auth.ClientToken, nil
}

// readSecret reads the SSH key from Vault.
func (p *VaultProvider) readSecret(token string) (string, error) {
	// Build URL for KV v2 secret
	secretURL := fmt.Sprintf("%s/v1/%s", strings.TrimSuffix(p.address, "/"), p.secretPath)

	req, err := http.NewRequest("GET", secretURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Vault-Token", token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Vault request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to read secret (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse KV v2 response
	var secretResp struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&secretResp); err != nil {
		return "", fmt.Errorf("failed to parse secret response: %w", err)
	}

	// Look for private_key field
	privateKey, ok := secretResp.Data.Data["private_key"].(string)
	if !ok {
		return "", fmt.Errorf("secret does not contain 'private_key' field")
	}

	return privateKey, nil
}
