# Research Findings: Issue #19 - OIDC/SSO Integration

## 1. Current Authentication System

### JWT-based Local Authentication
- User model with email/password credentials (bcrypt hashed)
- JWT token generation with HS256 algorithm
- Token validation middleware
- Session management via JWT claims
- Role-based access control (Admin, Operator, Viewer)
- Fine-grained permission system (sources:read, pipelines:write, etc.)

### Key Files
| File | Purpose |
|------|---------|
| `internal/api/models/auth.go` | User, JWT claims, role/permission models |
| `internal/api/services/auth.go` | Login, token generation/validation |
| `internal/api/middleware/auth.go` | JWT/API key validation middleware |
| `internal/api/repositories/user.go` | User persistence |

### Database Schema
- `philotes.users` table with email, password_hash, role, is_active, last_login_at
- `philotes.api_keys` - API key management
- `philotes.audit_logs` - Audit trail infrastructure

---

## 2. OAuth Patterns (Issue #32 Reference)

### Existing OAuth Implementation for Cloud Providers
- Full OAuth 2.0 with PKCE flow for Hetzner and OVH
- State management with 10-minute expiration
- Token encryption (AES-256-GCM) before storage
- Refresh token handling
- Provider-agnostic registry pattern

### Key Files
| File | Purpose |
|------|---------|
| `internal/api/services/oauth.go` | OAuth flow orchestration |
| `internal/api/repositories/oauth.go` | OAuth state/credential persistence |
| `internal/crypto/encryption.go` | AES-256-GCM encryption |
| `internal/installer/oauth/providers.go` | Provider abstraction |

### Database Schema
- `philotes.oauth_states` - Temporary PKCE state (expires after 10 minutes)
- `philotes.cloud_credentials` - Encrypted tokens with user/deployment associations

---

## 3. Schema Extensions Needed

```sql
-- New table for OIDC provider configurations
CREATE TABLE philotes.oidc_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    provider_type VARCHAR(50) NOT NULL,  -- google, okta, azure_ad, auth0, generic
    client_id VARCHAR(255) NOT NULL,
    client_secret_encrypted BYTEA NOT NULL,
    issuer_url VARCHAR(500) NOT NULL,
    discovery_url VARCHAR(500),
    scopes TEXT[] DEFAULT ARRAY['openid', 'profile', 'email'],
    groups_claim VARCHAR(100) DEFAULT 'groups',
    role_mapping JSONB DEFAULT '{}',
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Extend users table
ALTER TABLE philotes.users ADD COLUMN oidc_provider_id UUID REFERENCES philotes.oidc_providers(id);
ALTER TABLE philotes.users ADD COLUMN oidc_subject VARCHAR(255);
ALTER TABLE philotes.users ADD COLUMN oidc_groups TEXT[];
ALTER TABLE philotes.users ADD COLUMN oidc_attributes JSONB;
```

---

## 4. Configuration System

### Existing Pattern
- Environment variables via `internal/config/config.go`
- Vault integration for secrets
- AES-256 encryption already implemented

### Recommended OIDC Config Structure
```go
type OIDCConfig struct {
    Enabled           bool     `env:"OIDC_ENABLED" default:"false"`
    AllowLocalLogin   bool     `env:"OIDC_ALLOW_LOCAL_LOGIN" default:"true"`
    AutoCreateUsers   bool     `env:"OIDC_AUTO_CREATE_USERS" default:"true"`
    DefaultRole       string   `env:"OIDC_DEFAULT_ROLE" default:"viewer"`
}
```

---

## 5. API Endpoints Design

### Existing Auth Routes
- `POST /api/v1/auth/login` - Local login
- `POST /api/v1/auth/register` - Admin registration
- `GET /api/v1/auth/me` - Current user

### New OIDC Routes
```
GET    /api/v1/auth/oidc/providers              - List enabled providers (public)
POST   /api/v1/auth/oidc/:provider/authorize    - Start OIDC flow
GET    /api/v1/auth/oidc/callback               - Handle IdP callback
POST   /api/v1/auth/oidc/logout                 - Logout with IdP redirect

# Admin routes (protected)
GET    /api/v1/settings/oidc/providers          - List all providers
POST   /api/v1/settings/oidc/providers          - Create provider
PUT    /api/v1/settings/oidc/providers/:id      - Update provider
DELETE /api/v1/settings/oidc/providers/:id      - Delete provider
POST   /api/v1/settings/oidc/providers/:id/test - Test provider connection
```

---

## 6. Frontend Integration Points

### Existing OAuth Frontend
- `web/src/lib/api/oauth.ts` - OAuth API client
- `web/src/components/installer/oauth-connect.tsx` - OAuth UI
- `web/src/app/install/oauth/callback/page.tsx` - Callback handler

### Onboarding Integration
- `web/src/components/onboarding/step-sso-config.tsx` - SSO configuration step (placeholder)

---

## 7. Library Recommendations

### Go Libraries
- `github.com/coreos/go-oidc/v3` - Standard OIDC client library
- `github.com/golang-jwt/jwt/v5` - Already in use for JWT

### TypeScript Libraries
- Reuse existing patterns from OAuth implementation
- Consider `oidc-client-ts` for advanced features

---

## 8. Security Considerations

1. **PKCE Flow** - Already established pattern, reuse for OIDC
2. **Token Encryption** - Use existing `crypto.Encryptor`
3. **ID Token Validation** - Verify signature, audience, issuer, expiration
4. **CSRF Protection** - State parameter with 10-minute expiration
5. **Nonce Validation** - Include nonce in ID token requests

---

## 9. Files to Create

### Backend (New)
| File | Purpose |
|------|---------|
| `internal/api/models/oidc.go` | OIDC models and types |
| `internal/api/services/oidc.go` | OIDC business logic |
| `internal/api/repositories/oidc.go` | OIDC data persistence |
| `internal/api/handlers/oidc.go` | HTTP handlers |
| `internal/oidc/client.go` | OIDC client wrapper |
| `internal/oidc/providers/google.go` | Google-specific config |
| `internal/oidc/providers/okta.go` | Okta-specific config |
| `internal/oidc/providers/azure.go` | Azure AD-specific config |
| `internal/oidc/providers/auth0.go` | Auth0-specific config |
| `deployments/docker/init-scripts/13-oidc-schema.sql` | Database migration |

### Backend (Modify)
| File | Change |
|------|--------|
| `internal/config/config.go` | Add OIDCConfig |
| `internal/api/server.go` | Register OIDC routes |
| `internal/api/models/auth.go` | Add OIDC fields to User |
| `internal/api/repositories/user.go` | OIDC lookup methods |

### Frontend (New)
| File | Purpose |
|------|---------|
| `web/src/lib/api/oidc.ts` | OIDC API client |
| `web/src/lib/hooks/use-oidc.ts` | React Query hooks |
| `web/src/app/auth/oidc/callback/page.tsx` | Callback handler |
| `web/src/components/auth/oidc-login-button.tsx` | Provider login buttons |
| `web/src/components/settings/oidc-provider-form.tsx` | Provider config form |

### Frontend (Modify)
| File | Change |
|------|--------|
| `web/src/components/onboarding/step-sso-config.tsx` | Full implementation |
| `web/src/app/settings/page.tsx` | Add OIDC settings section |

---

## 10. Implementation Phases

### Phase 1: Core OIDC Infrastructure (~2,500 LOC)
- Database schema
- OIDC client wrapper
- Provider model and repository
- Basic authorize/callback flow
- JIT user provisioning

### Phase 2: Provider Implementations (~1,500 LOC)
- Google Workspace
- Okta
- Azure AD / Entra ID
- Auth0
- Generic OIDC

### Phase 3: Admin UI (~1,500 LOC)
- Provider configuration form
- Provider management UI
- Onboarding step implementation
- Login page with provider buttons

### Phase 4: Advanced Features (~500 LOC)
- Group/role mapping
- RP-Initiated Logout
- Token refresh
- Session management enhancements
