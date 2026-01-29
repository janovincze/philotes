# Implementation Plan: Issue #32 - Cloud Provider OAuth Integration

## Summary

Add OAuth 2.0 authentication for Hetzner and OVHcloud, with manual API key fallback for all providers. This enables a seamless "Connect with..." flow while maintaining flexibility for providers without OAuth support.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Frontend (Next.js)                            │
│  Provider Page → [Connect with OAuth] or [Enter API Key]        │
│                 ↓                                                │
│  Redirect to Provider → Callback → Show "Connected" Status      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Backend API (Go/Gin)                          │
│  /oauth/:provider/authorize  → Generate auth URL + state        │
│  /oauth/:provider/callback   → Exchange code for tokens         │
│  /credentials/:provider      → Store/retrieve credentials       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Database (PostgreSQL)                         │
│  oauth_states    → PKCE state, code_verifier (temp)             │
│  cloud_credentials → Encrypted tokens (existing table)          │
└─────────────────────────────────────────────────────────────────┘
```

## OAuth Flow

1. User clicks "Connect with Hetzner"
2. Frontend calls `POST /api/v1/installer/oauth/hetzner/authorize`
3. Backend generates:
   - `state` parameter (CSRF protection)
   - `code_verifier` + `code_challenge` (PKCE)
   - Stores state in database with short TTL (10 min)
4. Backend returns authorization URL
5. Frontend redirects user to provider
6. User approves access on provider site
7. Provider redirects to `/api/v1/installer/oauth/hetzner/callback?code=...&state=...`
8. Backend:
   - Validates state matches stored state
   - Exchanges code for access token (+ refresh token)
   - Encrypts tokens using AES-GCM
   - Stores in `cloud_credentials` table
   - Redirects to frontend with success
9. Frontend shows "Connected" status

## Files to Create

### Backend

| File | Description |
|------|-------------|
| `internal/api/handlers/oauth.go` | OAuth HTTP handlers |
| `internal/api/services/oauth.go` | OAuth business logic |
| `internal/api/models/oauth.go` | OAuth request/response types |
| `internal/api/repositories/oauth.go` | OAuth state storage |
| `internal/installer/oauth/hetzner.go` | Hetzner OAuth config |
| `internal/installer/oauth/ovh.go` | OVH OAuth config |
| `internal/crypto/encryption.go` | AES-GCM encryption utilities |

### Frontend

| File | Description |
|------|-------------|
| `web/src/lib/api/oauth.ts` | OAuth API client |
| `web/src/lib/hooks/use-oauth.ts` | OAuth React hooks |
| `web/src/components/installer/oauth-connect.tsx` | OAuth connect button |
| `web/src/components/installer/manual-credentials.tsx` | Manual API key form |
| `web/src/app/install/oauth/callback/page.tsx` | OAuth callback page |

### Database

| File | Description |
|------|-------------|
| `deployments/docker/init-scripts/11-oauth-schema.sql` | OAuth state table |

## Files to Modify

| File | Changes |
|------|---------|
| `internal/api/server.go` | Register OAuth routes |
| `internal/config/config.go` | Add OAuth config (client IDs) |
| `internal/installer/providers.go` | Add OAuth metadata |
| `web/src/app/install/[provider]/page.tsx` | Add OAuth/manual toggle |
| `web/src/lib/api/types.ts` | Add OAuth types |

## Database Schema

```sql
-- OAuth state for PKCE flow (temporary, auto-expires)
CREATE TABLE IF NOT EXISTS philotes.oauth_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    state VARCHAR(255) NOT NULL UNIQUE,
    code_verifier VARCHAR(255) NOT NULL,
    redirect_uri TEXT NOT NULL,
    user_id UUID REFERENCES philotes.users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth_states_state ON philotes.oauth_states(state);
CREATE INDEX idx_oauth_states_expires_at ON philotes.oauth_states(expires_at);

-- Extend cloud_credentials for OAuth tokens
ALTER TABLE philotes.cloud_credentials
ADD COLUMN IF NOT EXISTS credential_type VARCHAR(20) DEFAULT 'manual',
ADD COLUMN IF NOT EXISTS refresh_token_encrypted BYTEA,
ADD COLUMN IF NOT EXISTS token_expires_at TIMESTAMPTZ;
```

## API Endpoints

```
# OAuth Flow
POST   /api/v1/installer/oauth/:provider/authorize
       Request: { redirect_uri: string }
       Response: { authorization_url: string, state: string }

GET    /api/v1/installer/oauth/:provider/callback
       Query: code, state
       Response: Redirect to frontend with success/error

# Credential Management
POST   /api/v1/installer/credentials/:provider
       Request: { credentials: ProviderCredentials, deployment_id?: UUID }
       Response: { credential_id: UUID }

GET    /api/v1/installer/credentials
       Response: { credentials: [{ provider, type, connected_at, expires_at }] }

DELETE /api/v1/installer/credentials/:provider
       Response: 204 No Content
```

## Provider OAuth Configuration

### Hetzner Cloud
```go
type HetznerOAuthConfig struct {
    ClientID     string  // From env: HETZNER_OAUTH_CLIENT_ID
    ClientSecret string  // From env: HETZNER_OAUTH_CLIENT_SECRET
    AuthURL      string  // https://console.hetzner.cloud/oauth/authorize
    TokenURL     string  // https://console.hetzner.cloud/oauth/token
    Scopes       []string // ["read", "write"]
    RedirectURL  string  // {BASE_URL}/api/v1/installer/oauth/hetzner/callback
}
```

### OVHcloud
```go
type OVHOAuthConfig struct {
    ClientID     string  // From env: OVH_OAUTH_CLIENT_ID
    ClientSecret string  // From env: OVH_OAUTH_CLIENT_SECRET
    AuthURL      string  // https://www.ovh.com/auth/oauth2/authorize
    TokenURL     string  // https://www.ovh.com/auth/oauth2/token
    Scopes       []string // ["all"] (OVH uses broad scopes)
    RedirectURL  string  // {BASE_URL}/api/v1/installer/oauth/ovh/callback
}
```

## Security

### Token Encryption
- Use AES-256-GCM for encryption at rest
- Encryption key from Vault or environment variable
- Random nonce per encryption operation

### PKCE Flow
- Generate 32-byte random `code_verifier`
- Create `code_challenge` = base64url(sha256(code_verifier))
- Store `code_verifier` in database with `state`
- Include `code_challenge` in authorization request
- Include `code_verifier` in token exchange

### State Management
- 32-byte random state parameter
- 10-minute expiration
- One-time use (deleted after callback)
- Linked to user session if authenticated

## Implementation Order

### Part 1: Database & Models (~200 LOC)
1. Create OAuth schema migration
2. Add OAuth models and types

### Part 2: Encryption Utilities (~200 LOC)
1. Create AES-GCM encryption/decryption
2. Add config for encryption key

### Part 3: OAuth Service & Repository (~600 LOC)
1. Implement OAuth state repository
2. Implement OAuth service with PKCE

### Part 4: OAuth Handlers (~400 LOC)
1. Implement authorize endpoint
2. Implement callback endpoint
3. Implement credential management endpoints

### Part 5: Provider OAuth Configs (~300 LOC)
1. Implement Hetzner OAuth config
2. Implement OVH OAuth config
3. Update providers.go with OAuth metadata

### Part 6: Route Registration (~100 LOC)
1. Register OAuth routes in server.go
2. Add OAuth config to main config

### Part 7: Frontend OAuth Components (~800 LOC)
1. Create OAuth connect button component
2. Create manual credentials form
3. Create callback page
4. Update provider page with OAuth/manual toggle

### Part 8: Frontend Hooks & API (~400 LOC)
1. Create OAuth API client
2. Create OAuth hooks
3. Add OAuth types

## Verification

```bash
# Backend
go build ./...
go vet ./...

# Frontend
cd web && npm run build
cd web && npm run lint
```

## Test Plan

1. **Unit Tests**
   - Encryption/decryption utilities
   - PKCE code challenge generation
   - OAuth state management

2. **Integration Tests**
   - Full OAuth flow (mock provider)
   - Token storage and retrieval
   - Credential expiration

3. **Manual Testing**
   - Hetzner OAuth flow
   - OVH OAuth flow
   - Manual API key entry
   - Error handling (cancelled, expired)
