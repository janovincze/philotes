# Issue Context - INFRA-002: HashiCorp Vault Integration

## Issue Details

- **Issue Number:** #17
- **Title:** INFRA-002: HashiCorp Vault Integration
- **Type:** Infrastructure
- **Priority:** Medium
- **Epic:** Infrastructure
- **Phase:** v1
- **Milestone:** M3: Production Ready
- **Estimate:** ~4,000 LOC

## Goal

Securely manage sensitive configuration (database passwords, API keys, cloud credentials) using HashiCorp Vault instead of Kubernetes Secrets.

## Problem Statement

Kubernetes Secrets are base64-encoded, not encrypted. Sensitive data like database passwords needs proper secrets management with audit trails, rotation, and access control.

## Who Benefits

- Security teams requiring compliance (SOC2, GDPR)
- Organizations with existing Vault deployments
- Enterprises with strict secrets management policies

## How It's Used

Philotes components authenticate to Vault and retrieve secrets at runtime. Secrets can be rotated without redeploying. Audit logs track all secret access.

## Value for Philotes

Enterprise customers often require Vault integration. This enables adoption in security-conscious organizations and demonstrates Philotes takes security seriously.

## Acceptance Criteria

- [ ] Vault client integration in Go services
- [ ] Kubernetes auth method support
- [ ] Dynamic database credentials
- [ ] Secret rotation without restart
- [ ] Fallback to K8s Secrets if Vault unavailable
- [ ] Vault Agent sidecar option
- [ ] Audit logging for secret access

## Secrets to Manage

- Database passwords (source, buffer)
- MinIO credentials
- API keys
- Cloud provider tokens
- TLS certificates

## Dependencies

- INFRA-001 (Helm Charts) - Completed (#16)

## Blocks

- None
