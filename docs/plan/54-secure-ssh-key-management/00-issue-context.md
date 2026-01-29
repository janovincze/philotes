# Issue #54: Secure SSH Key Management using Pulumi secrets or Vault

## Summary

Implement secure SSH key management in Pulumi that avoids storing private key content in Pulumi state and supports both local development and CI/CD environments.

## Current Problem

In `pkg/cluster/kubeconfig.go`, SSH private keys are read from local files and passed directly to remote commands. This has issues:
- Private key content is stored in Pulumi state (even if encrypted)
- Requires the private key file to exist on the machine running Pulumi
- No integration with secrets managers
- No audit trail for key access

## Proposed Solution

1. **Option A: Pulumi ESC** - Use Pulumi secrets for simple key management
2. **Option B: HashiCorp Vault** - Leverage Vault for production (already exists)
3. **Option C: Cloud Provider Secrets** - Provider-native secrets (limited support)

**Recommendation:** Start with Pulumi secrets, add Vault support for production.

## Acceptance Criteria

- [ ] SSH private key not stored in plain Pulumi state
- [ ] Support for Pulumi secrets as key source
- [ ] Support for Vault as key source (production)
- [ ] Local file fallback for development
- [ ] Documentation for each approach

## Dependencies

- INFRA-017 (Vault Integration) - if exists
- FOUND-003 (Infrastructure as Code) - completed in Issue #3

## Labels

- `epic:infrastructure`
- `phase:v1`
- `priority:high`
- `type:infra`
