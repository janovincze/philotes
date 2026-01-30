# Implementation Plan: Issue #19 - OIDC/SSO Integration

## Overview

Implement Single Sign-On (SSO) via OIDC providers (Google, Okta, Azure AD, Auth0, Generic OIDC) to allow organizations to use their existing identity systems for Philotes authentication.

**Estimate:** ~6,000 LOC

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Frontend (Next.js)                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │
│  │  Login Page  │  │  OIDC Config │  │   Callback   │                   │
│  │  + Providers │  │  (Settings)  │  │   Handler    │                   │
│  └──────────────┘  └──────────────┘  └──────────────┘                   │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Backend (Go/Gin)                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │
│  │ OIDC Handler │  │ OIDC Service │  │  OIDC Repo   │                   │
│  │  /auth/oidc  │  │  + go-oidc   │  │  + crypto    │                   │
│  └──────────────┘  └──────────────┘  └──────────────┘                   │
│                           │                                              │
│                           ▼                                              │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                    Provider Registry                              │   │
│  │  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────────┐  │   │
│  │  │ Google │  │  Okta  │  │ Azure  │  │ Auth0  │  │  Generic   │  │   │
│  │  └────────┘  └────────┘  └────────┘  └────────┘  └────────────┘  │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Identity Providers                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │
│  │    Google    │  │     Okta     │  │   Azure AD   │                   │
│  │  Workspace   │  │              │  │  / Entra ID  │                   │
│  └──────────────┘  └──────────────┘  └──────────────┘                   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## OIDC Flow

```
1. User clicks "Sign in with Google" on login page
2. Frontend calls POST /api/v1/auth/oidc/google/authorize
3. Backend:
   - Generates state + nonce (PKCE)
   - Stores in philotes.oidc_states
   - Returns redirect URL to IdP
4. Frontend redirects to IdP
5. User authenticates at IdP
6. IdP redirects to /auth/oidc/callback?code=...&state=...
7. Frontend calls backend with code + state
8. Backend:
   - Validates state (CSRF protection)
   - Exchanges code for tokens
   - Validates ID token signature
   - Extracts user info (email, groups, attributes)
   - Creates/updates user (JIT provisioning)
   - Maps groups to roles
   - Generates Philotes JWT
   - Returns JWT to frontend
9. Frontend stores JWT, redirects to dashboard
```

---

## Database Schema

### New Table: `philotes.oidc_providers`

```sql
CREATE TABLE philotes.oidc_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(200) NOT NULL,
    provider_type VARCHAR(50) NOT NULL,  -- google, okta, azure_ad, auth0, generic

    -- OIDC Configuration
    issuer_url VARCHAR(500) NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    client_secret_encrypted BYTEA NOT NULL,
    scopes TEXT[] DEFAULT ARRAY['openid', 'profile', 'email'],

    -- Optional Discovery
    authorization_endpoint VARCHAR(500),
    token_endpoint VARCHAR(500),
    userinfo_endpoint VARCHAR(500),
    jwks_uri VARCHAR(500),

    -- Role Mapping
    groups_claim VARCHAR(100) DEFAULT 'groups',
    email_claim VARCHAR(100) DEFAULT 'email',
    name_claim VARCHAR(100) DEFAULT 'name',
    role_mapping JSONB DEFAULT '{}',  -- {"admin-group": "admin", "users": "viewer"}
    default_role VARCHAR(50) DEFAULT 'viewer',

    -- Settings
    enabled BOOLEAN DEFAULT true,
    auto_create_users BOOLEAN DEFAULT true,
    update_user_on_login BOOLEAN DEFAULT true,

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES philotes.users(id)
);

CREATE INDEX idx_oidc_providers_name ON philotes.oidc_providers(name);
CREATE INDEX idx_oidc_providers_enabled ON philotes.oidc_providers(enabled) WHERE enabled = true;
```

### New Table: `philotes.oidc_states`

```sql
CREATE TABLE philotes.oidc_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state VARCHAR(255) NOT NULL UNIQUE,
    nonce VARCHAR(255) NOT NULL,
    code_verifier VARCHAR(255) NOT NULL,  -- PKCE
    provider_id UUID NOT NULL REFERENCES philotes.oidc_providers(id),
    redirect_uri VARCHAR(500) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oidc_states_state ON philotes.oidc_states(state);
CREATE INDEX idx_oidc_states_expires ON philotes.oidc_states(expires_at);
```

### Extend: `philotes.users`

```sql
ALTER TABLE philotes.users ADD COLUMN oidc_provider_id UUID REFERENCES philotes.oidc_providers(id);
ALTER TABLE philotes.users ADD COLUMN oidc_subject VARCHAR(255);
ALTER TABLE philotes.users ADD COLUMN oidc_groups TEXT[];
ALTER TABLE philotes.users ADD COLUMN oidc_attributes JSONB;
ALTER TABLE philotes.users ADD COLUMN oidc_last_login TIMESTAMPTZ;

CREATE INDEX idx_users_oidc ON philotes.users(oidc_provider_id, oidc_subject) WHERE oidc_provider_id IS NOT NULL;
```

---

## API Endpoints

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/auth/oidc/providers` | List enabled OIDC providers |
| POST | `/api/v1/auth/oidc/:provider/authorize` | Start OIDC authorization flow |
| POST | `/api/v1/auth/oidc/callback` | Handle IdP callback |
| POST | `/api/v1/auth/oidc/logout` | Logout with IdP redirect |

### Admin Endpoints (Protected)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/settings/oidc/providers` | List all providers |
| POST | `/api/v1/settings/oidc/providers` | Create provider |
| GET | `/api/v1/settings/oidc/providers/:id` | Get provider details |
| PUT | `/api/v1/settings/oidc/providers/:id` | Update provider |
| DELETE | `/api/v1/settings/oidc/providers/:id` | Delete provider |
| POST | `/api/v1/settings/oidc/providers/:id/test` | Test provider connection |

---

## Files to Create

### Backend (~3,500 LOC)

| File | LOC | Description |
|------|-----|-------------|
| `deployments/docker/init-scripts/13-oidc-schema.sql` | 80 | Database migration |
| `internal/api/models/oidc.go` | 200 | OIDC models and request/response types |
| `internal/api/repositories/oidc.go` | 350 | Provider and state persistence |
| `internal/api/services/oidc.go` | 500 | OIDC business logic |
| `internal/api/handlers/oidc.go` | 300 | HTTP handlers |
| `internal/oidc/client.go` | 250 | go-oidc wrapper |
| `internal/oidc/providers/registry.go` | 100 | Provider registry |
| `internal/oidc/providers/google.go` | 150 | Google Workspace config |
| `internal/oidc/providers/okta.go` | 150 | Okta config |
| `internal/oidc/providers/azure.go` | 150 | Azure AD config |
| `internal/oidc/providers/auth0.go` | 150 | Auth0 config |
| `internal/oidc/providers/generic.go` | 100 | Generic OIDC config |

### Backend Modifications (~500 LOC)

| File | Changes |
|------|---------|
| `internal/config/config.go` | Add OIDCConfig struct |
| `internal/api/server.go` | Register OIDC routes, create OIDCService |
| `internal/api/models/auth.go` | Add OIDC fields to User model |
| `internal/api/repositories/user.go` | Add OIDC lookup methods |
| `internal/api/services/auth.go` | Support OIDC users in session |

### Frontend (~2,000 LOC)

| File | LOC | Description |
|------|-----|-------------|
| `web/src/lib/api/oidc.ts` | 150 | OIDC API client |
| `web/src/lib/hooks/use-oidc.ts` | 100 | React Query hooks |
| `web/src/app/auth/oidc/callback/page.tsx` | 150 | Callback handler |
| `web/src/components/auth/oidc-login-button.tsx` | 100 | Provider login button |
| `web/src/components/auth/oidc-login-section.tsx` | 150 | Login section with providers |
| `web/src/components/settings/oidc-provider-form.tsx` | 400 | Provider config form |
| `web/src/components/settings/oidc-providers-list.tsx` | 200 | Provider list component |
| `web/src/components/onboarding/step-sso-config.tsx` | 350 | Full SSO step implementation |
| `web/src/app/settings/page.tsx` (modify) | 100 | Add OIDC settings section |

---

## Implementation Phases

### Phase 1: Backend Foundation (Day 1-2)

1. Create database migration
2. Implement OIDC models
3. Implement OIDC repository
4. Create go-oidc client wrapper
5. Implement provider registry
6. Add config support

### Phase 2: OIDC Flow (Day 2-3)

1. Implement OIDC service
2. Implement OIDC handlers
3. Authorize endpoint (generate state, redirect)
4. Callback endpoint (exchange, validate, provision)
5. Logout endpoint

### Phase 3: Provider Implementations (Day 3-4)

1. Google Workspace
2. Okta
3. Azure AD / Entra ID
4. Auth0
5. Generic OIDC

### Phase 4: Frontend (Day 4-5)

1. OIDC API client and hooks
2. Callback page
3. Login buttons
4. Provider configuration form
5. Settings page integration

### Phase 5: Onboarding Integration (Day 5)

1. Update step-sso-config.tsx
2. Test full onboarding flow
3. Documentation

### Phase 6: Testing & Polish (Day 6)

1. Unit tests
2. Integration tests
3. Manual testing with real providers
4. Security review

---

## Security Checklist

- [ ] PKCE for all flows
- [ ] State parameter with 10-minute expiration
- [ ] Nonce validation in ID token
- [ ] ID token signature verification
- [ ] Audience validation
- [ ] Issuer validation
- [ ] Token expiration check
- [ ] Client secret encryption at rest
- [ ] No sensitive data in logs
- [ ] Rate limiting on auth endpoints

---

## Verification Plan

### Backend
```bash
# Build
go build ./...

# Lint
make lint

# Tests
go test ./internal/api/... ./internal/oidc/...
```

### Frontend
```bash
cd web
npm run build
npm run lint
npm test
```

### Manual Testing
1. Configure Google OIDC provider
2. Test login flow
3. Verify JIT user creation
4. Test group/role mapping
5. Test logout
6. Test onboarding SSO step

---

## LOC Summary

| Category | LOC |
|----------|-----|
| Backend Go (new) | 2,480 |
| Backend Go (modified) | 500 |
| Database SQL | 80 |
| Frontend TypeScript | 1,700 |
| Tests | 750 |
| **Total** | **~5,510** |
