# Phase 2 Implementation: OIDC Flow

## Summary

Phase 2 implements the OIDC authentication flow including:
- OIDC client wrapper for token exchange and validation
- Provider registry with provider-specific defaults
- OIDC service with authorization and callback handling
- HTTP handlers for public and admin endpoints
- Server.go updates for OIDC integration

## Files Created

### internal/oidc/client.go (~280 LOC)
- `Client` struct wrapping OIDC operations
- `Discover()` - Fetches OIDC discovery document
- `AuthorizationURL()` - Builds authorization URL with PKCE
- `Exchange()` - Exchanges authorization code for tokens
- `ParseIDToken()` - Parses and validates ID token (issuer, audience, expiration, nonce)
- `GetUserInfo()` - Fetches user info from userinfo endpoint
- PKCE helpers: `GenerateState`, `GenerateNonce`, `GenerateCodeVerifier`, `GenerateCodeChallenge`

### internal/oidc/providers/registry.go (~80 LOC)
- `ProviderConfig` interface for provider-specific configuration
- `Registry` struct for managing provider configs
- `ApplyDefaults()` - Applies provider-specific defaults to requests

### internal/oidc/providers/*.go (~150 LOC each)
- `google.go` - Google Workspace defaults
- `okta.go` - Okta defaults
- `azure.go` - Azure AD / Entra ID defaults
- `auth0.go` - Auth0 defaults
- `generic.go` - Generic OIDC defaults

### internal/api/services/oidc.go (~500 LOC)
**Public methods:**
- `ListEnabledProviders()` - List enabled providers for login page
- `StartAuthorization()` - Generate state/nonce/PKCE and return auth URL
- `HandleCallback()` - Validate state, exchange code, provision user, generate JWT

**Admin methods:**
- `CreateProvider()` - Create OIDC provider
- `GetProvider()` - Get provider by ID
- `ListProviders()` - List all providers
- `UpdateProvider()` - Update provider
- `DeleteProvider()` - Delete provider
- `TestProvider()` - Test provider discovery
- `CleanupExpiredStates()` - Remove expired states

**Internal methods:**
- `provisionUser()` - JIT user provisioning with group/role mapping
- `mapGroupsToRole()` - Map IdP groups to Philotes roles
- `generateJWT()` - Generate JWT for OIDC users

### internal/api/handlers/oidc.go (~300 LOC)
**Public endpoints:**
- `GET /api/v1/auth/oidc/providers` - List enabled providers
- `POST /api/v1/auth/oidc/:provider/authorize` - Start OIDC flow
- `POST /api/v1/auth/oidc/callback` - Handle callback
- `GET /api/v1/auth/oidc/callback` - Handle callback (GET for IdP redirects)

**Admin endpoints (protected):**
- `GET /api/v1/settings/oidc/providers` - List all providers
- `POST /api/v1/settings/oidc/providers` - Create provider
- `GET /api/v1/settings/oidc/providers/:id` - Get provider
- `PUT /api/v1/settings/oidc/providers/:id` - Update provider
- `DELETE /api/v1/settings/oidc/providers/:id` - Delete provider
- `POST /api/v1/settings/oidc/providers/:id/test` - Test provider

## Files Modified

### internal/api/server.go
- Added `oidcService` field to Server struct
- Added `OIDCService` field to ServerConfig struct
- Added OIDC handler creation and route registration

## Security Features

- PKCE (Proof Key for Code Exchange) with S256 challenge
- State parameter with configurable expiration (default 10 minutes)
- Nonce validation in ID token
- ID token validation: issuer, audience, expiration
- Redirect URI validation
- One-time use state (deleted after callback)
- No sensitive data logged (client secrets, code verifier, nonce)

## OIDC Flow

```
1. User clicks "Sign in with Provider"
2. Frontend calls POST /api/v1/auth/oidc/:provider/authorize
3. Backend generates state, nonce, code_verifier
4. Backend stores state in DB with 10-minute expiration
5. Backend returns authorization_url with PKCE code_challenge
6. Frontend redirects to IdP
7. User authenticates at IdP
8. IdP redirects to /api/v1/auth/oidc/callback
9. Backend validates state, exchanges code for tokens
10. Backend validates ID token (issuer, audience, nonce, expiration)
11. Backend provisions/updates user (JIT provisioning)
12. Backend maps IdP groups to Philotes roles
13. Backend generates Philotes JWT
14. Backend returns JWT to frontend
```

## JIT User Provisioning

1. Look up user by OIDC subject (provider_id + subject)
2. If found: update groups, map roles, update last_login
3. If not found by subject: look up by email
4. If found by email: link OIDC provider to existing user
5. If not found and auto_create_users enabled: create new user
6. Role assigned based on role_mapping or default_role

## Verification

```bash
go build ./...     # ✓ Passes
go vet ./...       # ✓ Passes
go test ./internal/api/... ./internal/oidc/...  # ✓ Passes
```

## Next Steps (Phase 3: Frontend)

1. Create OIDC API client (`web/src/lib/api/oidc.ts`)
2. Create React Query hooks (`web/src/lib/hooks/use-oidc.ts`)
3. Create callback page (`web/src/app/auth/oidc/callback/page.tsx`)
4. Create provider login buttons
5. Update login page with OIDC options
6. Create provider configuration form for admin settings
