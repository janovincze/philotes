# Issue #32 - Cloud Provider OAuth Integration

## Summary

Enable secure, user-friendly authentication to cloud providers via OAuth, eliminating manual API token handling.

## Context

- **Goal:** Provide "Login with..." flow for cloud providers instead of manual token copying
- **Problem:** Manual API token handling is error-prone and intimidating for non-technical users
- **Value:** Reduced setup friction, professional UX, better security

## Supported Providers

| Provider | Auth Type | Notes |
|----------|-----------|-------|
| Hetzner | OAuth 2.0 | Read/write servers, networks, volumes |
| OVHcloud | OAuth 2.0 | Manage cloud projects |
| Scaleway | API Key only | No OAuth available - manual entry |
| Exoscale | API Key only | No OAuth available - manual entry |
| Contabo | API Key only | No OAuth available - manual entry |

## Acceptance Criteria

- [ ] OAuth client registration flow for Hetzner and OVH
- [ ] Secure redirect flow with PKCE
- [ ] Token exchange and encrypted storage
- [ ] Token refresh handling
- [ ] Fallback to manual API key entry for all providers
- [ ] Clear permission explanations
- [ ] Revocation instructions in dashboard

## Security Requirements

- Use PKCE for public clients
- Store tokens encrypted at rest
- Request minimum required scopes
- Auto-expire tokens after deployment
- Clear audit trail

## Dependencies

- Issue #31 (INSTALL-001) - One-Click Installer Foundation ✅ COMPLETED

## Estimate

~5,000 LOC
