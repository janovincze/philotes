package sshkeys

import (
	"fmt"
	"os"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// FileProvider loads SSH keys from local files.
// WARNING: This provider is less secure than Pulumi secrets or Vault.
// The private key content is read from disk and passed to Pulumi,
// which may store it in the state file (even if encrypted).
// Use only for development or when other options are not available.
type FileProvider struct {
	filePath string
}

// NewFileProvider creates a new file-based SSH key provider.
func NewFileProvider(filePath string) *FileProvider {
	return &FileProvider{filePath: filePath}
}

// GetPrivateKey reads the SSH private key from a local file.
// It logs a security warning since file-based keys are less secure.
func (p *FileProvider) GetPrivateKey(ctx *pulumi.Context) (pulumi.StringOutput, error) {
	// Log security warning
	ctx.Log.Warn("SSH key loaded from local file. For production, use 'pulumi' or 'vault' source.", nil)

	// Read the private key from file
	content, err := os.ReadFile(p.filePath)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("failed to read SSH private key from %s: %w", p.filePath, err)
	}

	// Return as a secret to avoid plain text in state
	// Note: The key content is still processed at runtime, but marking it as secret
	// ensures it's encrypted in state and not displayed in logs
	return pulumi.ToSecret(pulumi.String(string(content))).(pulumi.StringOutput), nil
}

// Source returns the source type.
func (p *FileProvider) Source() SSHKeySource {
	return SourceFile
}
