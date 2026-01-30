# Issue #19: AUTH-001 - OIDC/SSO Integration

## Overview

**Goal:** Enable Single Sign-On (SSO) via OIDC providers (Google, Okta, Azure AD) so organizations can use their existing identity systems instead of managing separate Philotes credentials.

**Problem it solves:** Managing separate usernames/passwords for every tool is a security risk and user annoyance. SSO centralizes authentication, enables MFA enforcement, and simplifies offboarding.

**Estimate:** ~6,000 LOC

## Supported Providers

- Google Workspace
- Okta
- Azure AD / Entra ID
- Auth0
- Generic OIDC

## Acceptance Criteria

- [ ] OIDC client implementation
- [ ] Provider configuration UI
- [ ] JWT token validation
- [ ] User provisioning on first login (JIT provisioning)
- [ ] Group/role mapping from IdP claims
- [ ] Session management
- [ ] Logout with IdP redirect

## Dependencies

- API-003 (depends on)

## Blocks

- AUTH-002 (RBAC and Multi-Tenancy Foundation)

## Who Benefits

- Enterprise security teams with SSO mandates
- Users who want one-click login via their org identity
- IT admins managing access across tools

## Value for Philotes

SSO is often a procurement requirement for enterprise customers. This feature unlocks larger organization adoption.

## Related Work

- Issue #34 (Onboarding Wizard) - Step 3 is SSO configuration placeholder
- OAuth integration for cloud providers (Issue #32) - Similar OAuth flow patterns
