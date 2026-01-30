# Phase 3 Implementation: Frontend

## Summary

Phase 3 implements the frontend components for OIDC/SSO authentication including:
- OIDC API client and TypeScript types
- React Query hooks for OIDC operations
- OIDC callback page for handling IdP redirects
- Login button and section components
- Admin provider configuration form and list

## Files Created

### web/src/lib/api/types.ts (Extended)
Added OIDC types:
- `OIDCProviderType` - Provider type enum
- `OIDCProviderSummary` - Provider info without secrets
- `OIDCProvidersResponse` - List response
- `CreateOIDCProviderRequest` - Create request
- `UpdateOIDCProviderRequest` - Update request
- `OIDCAuthorizeRequest/Response` - Authorization flow
- `OIDCCallbackRequest/Response` - Callback handling
- `OIDCTestResponse` - Test connection response

### web/src/lib/api/oidc.ts (~140 LOC)
OIDC API client with methods:
- `listEnabledProviders()` - Get enabled providers for login
- `authorize()` - Start OIDC flow
- `callback()` - Handle IdP callback
- `listProviders()` - Admin: list all providers
- `createProvider()` - Admin: create provider
- `getProvider()` - Admin: get by ID
- `updateProvider()` - Admin: update provider
- `deleteProvider()` - Admin: delete provider
- `testProvider()` - Admin: test connection
- `getCallbackUrl()` - Get frontend callback URL
- `getProviderIcon()` - Get icon name for provider type
- `getProviderTypeLabel()` - Get display label for type

### web/src/lib/hooks/use-oidc.ts (~130 LOC)
React Query hooks:
- `useEnabledOIDCProviders()` - Fetch enabled providers
- `useOIDCProviders()` - Fetch all providers (admin)
- `useOIDCProvider(id)` - Fetch single provider
- `useCreateOIDCProvider()` - Create mutation
- `useUpdateOIDCProvider()` - Update mutation
- `useDeleteOIDCProvider()` - Delete mutation
- `useTestOIDCProvider()` - Test mutation
- `useOIDCAuthorize()` - Start auth mutation
- `useOIDCCallback()` - Handle callback mutation
- `useStartOIDCFlow()` - Combined flow starter

### web/src/app/auth/oidc/callback/page.tsx (~130 LOC)
OIDC callback page that:
- Parses URL params (code, state, error)
- Shows processing spinner
- Exchanges code for tokens via API
- Stores JWT in localStorage
- Redirects to dashboard on success
- Shows error with retry button on failure

### web/src/components/auth/oidc-login-button.tsx (~160 LOC)
Provider-branded login button:
- Provider-specific colors (Google, Okta, Azure AD, Auth0)
- SVG icons for each provider
- Loading state during auth flow
- Supports `default` and `outline` variants

### web/src/components/auth/oidc-login-section.tsx (~80 LOC)
Login section component:
- Fetches enabled providers
- Renders provider buttons
- Shows loading skeleton
- Shows error alert on failure
- Optional divider ("Or continue with")

### web/src/components/settings/oidc-provider-form.tsx (~420 LOC)
Admin form for creating/editing providers:
- Basic settings: name, display name, type, issuer URL
- Client credentials: client ID, client secret
- Scopes configuration
- User provisioning settings
- Role mapping (group → role)
- Enable/disable toggle
- Test connection button (edit mode)
- Zod validation

### web/src/components/settings/oidc-providers-list.tsx (~340 LOC)
Admin provider list:
- Table with name, type, issuer, status
- Dropdown menu: edit, test, enable/disable, delete
- Create provider dialog
- Edit provider dialog
- Delete confirmation dialog
- Test result indicators

### web/src/components/ui/dialog.tsx (~120 LOC)
New shadcn/ui Dialog component for modals.

## Files Modified

### web/src/lib/api/index.ts
Added export for `oidcApi`.

## Integration Points

### Login Page
```tsx
import { OIDCLoginSection } from "@/components/auth/oidc-login-section"

function LoginPage() {
  return (
    <div>
      {/* Email/password form */}
      <OIDCLoginSection />
    </div>
  )
}
```

### Settings Page
```tsx
import { OIDCProvidersList } from "@/components/settings/oidc-providers-list"

function SettingsPage() {
  return (
    <div>
      <OIDCProvidersList />
    </div>
  )
}
```

## OIDC Flow (Frontend)

```
1. User clicks OIDC provider button
2. OIDCLoginButton calls useStartOIDCFlow().startFlow(providerName)
3. Frontend calls POST /api/v1/auth/oidc/:provider/authorize
4. Backend returns authorization_url
5. Frontend redirects to IdP
6. User authenticates at IdP
7. IdP redirects to /auth/oidc/callback?code=...&state=...
8. OIDCCallbackPage parses params
9. Frontend calls POST /api/v1/auth/oidc/callback
10. Backend returns JWT + user info
11. Frontend stores JWT in localStorage
12. Frontend redirects to dashboard
```

## Verification

```bash
cd web
pnpm exec tsc --noEmit  # ✓ Passes
pnpm lint               # ✓ Passes
pnpm build              # ✓ Passes
```

## Next Steps (Phase 4: Onboarding Integration)

1. Update `step-sso-config.tsx` with full implementation
2. Add provider quick-setup templates
3. Test onboarding flow with SSO step

## Next Steps (Phase 5: Testing)

1. Unit tests for hooks
2. Component tests for forms
3. Manual testing with real OIDC providers
