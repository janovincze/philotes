# Research - Issue #32 OAuth Integration

## Existing Infrastructure

### Backend Code Structure
- **Handler**: `internal/api/handlers/installer.go` - endpoints for providers, deployments
- **Service**: `internal/api/services/installer.go` - business logic
- **Repository**: `internal/api/repositories/deployment.go` - data access
- **Models**: `internal/api/models/installer.go` - `ProviderCredentials` struct

### Current Credential Handling
```go
type ProviderCredentials struct {
    HetznerToken           string
    ScalewayAccessKey      string
    ScalewaySecretKey      string
    ScalewayProjectID      string
    OVHEndpoint            string
    OVHApplicationKey      string
    OVHApplicationSecret   string
    OVHConsumerKey         string
    OVHServiceName         string
    ExoscaleAPIKey         string
    ExoscaleAPISecret      string
    ContaboClientID        string
    ContaboClientSecret    string
    ContaboAPIUser         string
    ContaboAPIPassword     string
}
```

### Database Schema
`cloud_credentials` table already exists with:
- `id`, `user_id`, `provider`, `credential_type`
- `encrypted_data` (BYTEA for encrypted storage)
- `expires_at`, timestamps

## Provider OAuth Support

| Provider | OAuth Support | Notes |
|----------|---------------|-------|
| Hetzner | ✅ OAuth 2.0 | Simple token-based, supports OAuth apps |
| OVHcloud | ✅ OAuth 2.0 | Complex with 4 credential fields |
| Scaleway | ❌ | API keys only |
| Exoscale | ❌ | API keys only |
| Contabo | ❌ | API keys only |

## Security Patterns Found

- JWT authentication via `golang-jwt/jwt/v5`
- API key hashing with SHA256 in `internal/api/services/api_key.go`
- Vault integration exists at `internal/vault/config.go`
- `golang.org/x/crypto` available for encryption

## Recommended Approach

### OAuth Library
Use `golang.org/x/oauth2` - Standard Go OAuth2 client with PKCE support

### API Endpoints
```
POST   /api/v1/installer/oauth/:provider/authorize  → Start OAuth flow
GET    /api/v1/installer/oauth/:provider/callback   → Exchange code for token
POST   /api/v1/installer/credentials/:provider      → Manual credential entry
DELETE /api/v1/installer/credentials/:provider      → Revoke credentials
GET    /api/v1/installer/credentials                → List stored credentials
```

### Frontend Flow
1. User clicks "Connect with Hetzner"
2. Frontend calls `/oauth/:provider/authorize`
3. Backend returns authorization URL with state + PKCE
4. User redirected to provider
5. Provider redirects back to `/oauth/:provider/callback`
6. Backend exchanges code for tokens
7. Tokens encrypted and stored in database
8. Frontend shows "Connected" status

## Files to Modify

### Backend
- `internal/api/handlers/installer.go` - Add OAuth handlers
- `internal/api/services/installer.go` - OAuth business logic
- `internal/api/models/installer.go` - OAuth token models
- `internal/installer/providers.go` - OAuth config (client IDs, scopes)
- `internal/config/config.go` - OAuth environment config
- `go.mod` - Add `golang.org/x/oauth2`

### Frontend
- `web/src/app/install/[provider]/page.tsx` - OAuth connect buttons
- `web/src/lib/api/installer.ts` - OAuth API calls
- `web/src/lib/api/types.ts` - OAuth types

### Database
- New migration for OAuth-specific columns
