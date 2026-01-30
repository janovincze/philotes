# Phase 4 Implementation: Onboarding Integration

## Summary

Phase 4 updates the onboarding SSO configuration step with full OIDC provider setup capabilities, allowing users to configure SSO during initial Philotes setup.

## Files Modified

### web/src/components/onboarding/step-sso-config.tsx (~350 LOC)

Complete rewrite of the SSO configuration step with:

**Features:**
- Provider template selection (Google, Okta, Azure AD, Auth0, Generic)
- Quick setup form for adding providers
- View/manage existing providers
- Enable/disable and delete providers
- Skip option for later configuration

**Provider Templates:**
```typescript
const providerTemplates = [
  { type: "google", label: "Google Workspace", defaultIssuer: "https://accounts.google.com" },
  { type: "okta", label: "Okta", issuerPlaceholder: "https://your-org.okta.com" },
  { type: "azure_ad", label: "Microsoft Entra ID", issuerPlaceholder: "https://login.microsoftonline.com/{tenant-id}/v2.0" },
  { type: "auth0", label: "Auth0", issuerPlaceholder: "https://your-tenant.auth0.com" },
  { type: "generic", label: "Other OIDC Provider", issuerPlaceholder: "https://your-idp.example.com" },
]
```

**Quick Setup Form Fields:**
- Provider type (select)
- Display name
- Issuer URL
- Client ID
- Client secret
- Advanced settings (collapsible, deferred to Settings page)

**UI States:**
1. **No providers configured** - Shows provider template grid
2. **Providers exist** - Shows configured providers list with actions
3. **Adding provider** - Shows quick setup form

**Hooks Used:**
- `useOIDCProviders()` - Fetch existing providers
- `useCreateOIDCProvider()` - Create new provider
- `useDeleteOIDCProvider()` - Remove provider

## User Flow

```
1. User reaches SSO step in onboarding
2. If no providers exist:
   - Choose provider template (Google, Okta, etc.)
   - Fill in quick setup form
   - Provider created with sensible defaults
   - Return to SSO step showing configured provider
3. If providers exist:
   - View list of configured providers
   - Option to add more or continue
4. Skip available at any time
```

## Default Values Applied

When creating via quick setup:
- `scopes`: ["openid", "profile", "email"]
- `enabled`: true
- `auto_create_users`: true
- `name`: generated from display_name (lowercase, hyphenated)

## Verification

```bash
cd web
pnpm exec tsc --noEmit  # ✓ Passes
pnpm lint               # ✓ Passes (1 warning about React Hook Form)
pnpm build              # ✓ Passes
```

## Next Steps (Phase 5: Testing)

1. Unit tests for OIDC hooks
2. Component tests for forms
3. Manual testing with real providers:
   - Google Cloud Console OAuth setup
   - Okta developer account
   - Azure AD app registration
   - Auth0 tenant configuration
4. E2E test for complete OIDC flow
