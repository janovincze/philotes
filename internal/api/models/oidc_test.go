package models

import (
	"testing"
)

func TestCreateOIDCProviderRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       *CreateOIDCProviderRequest
		wantErr   bool
		errFields []string
	}{
		{
			name: "valid request",
			req: &CreateOIDCProviderRequest{
				Name:         "google-workspace",
				DisplayName:  "Google Workspace",
				ProviderType: OIDCProviderTypeGoogle,
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "client-id-123",
				ClientSecret: "client-secret-456",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: &CreateOIDCProviderRequest{
				DisplayName:  "Google Workspace",
				ProviderType: OIDCProviderTypeGoogle,
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "client-id-123",
				ClientSecret: "client-secret-456",
			},
			wantErr:   true,
			errFields: []string{"name"},
		},
		{
			name: "invalid name - uppercase",
			req: &CreateOIDCProviderRequest{
				Name:         "Google-Workspace",
				DisplayName:  "Google Workspace",
				ProviderType: OIDCProviderTypeGoogle,
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "client-id-123",
				ClientSecret: "client-secret-456",
			},
			wantErr:   true,
			errFields: []string{"name"},
		},
		{
			name: "invalid name - starts with hyphen",
			req: &CreateOIDCProviderRequest{
				Name:         "-google",
				DisplayName:  "Google Workspace",
				ProviderType: OIDCProviderTypeGoogle,
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "client-id-123",
				ClientSecret: "client-secret-456",
			},
			wantErr:   true,
			errFields: []string{"name"},
		},
		{
			name: "invalid name - ends with hyphen",
			req: &CreateOIDCProviderRequest{
				Name:         "google-",
				DisplayName:  "Google Workspace",
				ProviderType: OIDCProviderTypeGoogle,
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "client-id-123",
				ClientSecret: "client-secret-456",
			},
			wantErr:   true,
			errFields: []string{"name"},
		},
		{
			name: "missing display name",
			req: &CreateOIDCProviderRequest{
				Name:         "google",
				ProviderType: OIDCProviderTypeGoogle,
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "client-id-123",
				ClientSecret: "client-secret-456",
			},
			wantErr:   true,
			errFields: []string{"display_name"},
		},
		{
			name: "invalid provider type",
			req: &CreateOIDCProviderRequest{
				Name:         "custom",
				DisplayName:  "Custom Provider",
				ProviderType: "invalid-type",
				IssuerURL:    "https://example.com",
				ClientID:     "client-id-123",
				ClientSecret: "client-secret-456",
			},
			wantErr:   true,
			errFields: []string{"provider_type"},
		},
		{
			name: "missing issuer URL",
			req: &CreateOIDCProviderRequest{
				Name:         "google",
				DisplayName:  "Google Workspace",
				ProviderType: OIDCProviderTypeGoogle,
				ClientID:     "client-id-123",
				ClientSecret: "client-secret-456",
			},
			wantErr:   true,
			errFields: []string{"issuer_url"},
		},
		{
			name: "invalid issuer URL",
			req: &CreateOIDCProviderRequest{
				Name:         "google",
				DisplayName:  "Google Workspace",
				ProviderType: OIDCProviderTypeGoogle,
				IssuerURL:    "not-a-url",
				ClientID:     "client-id-123",
				ClientSecret: "client-secret-456",
			},
			wantErr:   true,
			errFields: []string{"issuer_url"},
		},
		{
			name: "missing client ID",
			req: &CreateOIDCProviderRequest{
				Name:         "google",
				DisplayName:  "Google Workspace",
				ProviderType: OIDCProviderTypeGoogle,
				IssuerURL:    "https://accounts.google.com",
				ClientSecret: "client-secret-456",
			},
			wantErr:   true,
			errFields: []string{"client_id"},
		},
		{
			name: "missing client secret",
			req: &CreateOIDCProviderRequest{
				Name:         "google",
				DisplayName:  "Google Workspace",
				ProviderType: OIDCProviderTypeGoogle,
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "client-id-123",
			},
			wantErr:   true,
			errFields: []string{"client_secret"},
		},
		{
			name: "invalid default role",
			req: &CreateOIDCProviderRequest{
				Name:         "google",
				DisplayName:  "Google Workspace",
				ProviderType: OIDCProviderTypeGoogle,
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "client-id-123",
				ClientSecret: "client-secret-456",
				DefaultRole:  "superuser",
			},
			wantErr:   true,
			errFields: []string{"default_role"},
		},
		{
			name: "invalid role mapping value",
			req: &CreateOIDCProviderRequest{
				Name:         "google",
				DisplayName:  "Google Workspace",
				ProviderType: OIDCProviderTypeGoogle,
				IssuerURL:    "https://accounts.google.com",
				ClientID:     "client-id-123",
				ClientSecret: "client-secret-456",
				RoleMapping:  map[string]UserRole{"admins": "superuser"},
			},
			wantErr:   true,
			errFields: []string{"role_mapping"},
		},
		{
			name: "all provider types valid - okta",
			req: &CreateOIDCProviderRequest{
				Name:         "okta",
				DisplayName:  "Okta",
				ProviderType: OIDCProviderTypeOkta,
				IssuerURL:    "https://dev-123.okta.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
			wantErr: false,
		},
		{
			name: "all provider types valid - azure_ad",
			req: &CreateOIDCProviderRequest{
				Name:         "azure",
				DisplayName:  "Azure AD",
				ProviderType: OIDCProviderTypeAzureAD,
				IssuerURL:    "https://login.microsoftonline.com/tenant/v2.0",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
			wantErr: false,
		},
		{
			name: "all provider types valid - auth0",
			req: &CreateOIDCProviderRequest{
				Name:         "auth0",
				DisplayName:  "Auth0",
				ProviderType: OIDCProviderTypeAuth0,
				IssuerURL:    "https://dev-123.auth0.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
			wantErr: false,
		},
		{
			name: "all provider types valid - generic",
			req: &CreateOIDCProviderRequest{
				Name:         "custom-idp",
				DisplayName:  "Custom IdP",
				ProviderType: OIDCProviderTypeGeneric,
				IssuerURL:    "https://idp.example.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.req.Validate()
			if tt.wantErr && len(errors) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("unexpected validation error: %v", errors)
			}
			if tt.wantErr && len(tt.errFields) > 0 {
				for _, field := range tt.errFields {
					found := false
					for _, err := range errors {
						if err.Field == field {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error for field %s, but not found in errors: %v", field, errors)
					}
				}
			}
		})
	}
}

func TestCreateOIDCProviderRequest_ApplyDefaults(t *testing.T) {
	req := &CreateOIDCProviderRequest{
		Name:         "google",
		DisplayName:  "Google",
		ProviderType: OIDCProviderTypeGoogle,
		IssuerURL:    "https://accounts.google.com",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}

	req.ApplyDefaults()

	// Check scopes
	if len(req.Scopes) != 3 {
		t.Errorf("expected 3 default scopes, got %d", len(req.Scopes))
	}
	expectedScopes := []string{"openid", "profile", "email"}
	for i, scope := range expectedScopes {
		if req.Scopes[i] != scope {
			t.Errorf("expected scope %s, got %s", scope, req.Scopes[i])
		}
	}

	// Check groups claim
	if req.GroupsClaim != "groups" {
		t.Errorf("expected groups_claim 'groups', got '%s'", req.GroupsClaim)
	}

	// Check default role
	if req.DefaultRole != RoleViewer {
		t.Errorf("expected default_role 'viewer', got '%s'", req.DefaultRole)
	}

	// Check enabled
	if req.Enabled == nil || !*req.Enabled {
		t.Error("expected enabled to be true by default")
	}

	// Check auto_create_users
	if req.AutoCreateUsers == nil || !*req.AutoCreateUsers {
		t.Error("expected auto_create_users to be true by default")
	}

	// Check role mapping is initialized
	if req.RoleMapping == nil {
		t.Error("expected role_mapping to be initialized")
	}
}

func TestCreateOIDCProviderRequest_ApplyDefaults_PreservesValues(t *testing.T) {
	enabled := false
	autoCreate := false
	req := &CreateOIDCProviderRequest{
		Name:            "google",
		DisplayName:     "Google",
		ProviderType:    OIDCProviderTypeGoogle,
		IssuerURL:       "https://accounts.google.com",
		ClientID:        "client-id",
		ClientSecret:    "client-secret",
		Scopes:          []string{"openid", "email"},
		GroupsClaim:     "custom_groups",
		DefaultRole:     RoleAdmin,
		Enabled:         &enabled,
		AutoCreateUsers: &autoCreate,
		RoleMapping:     map[string]UserRole{"admins": RoleAdmin},
	}

	req.ApplyDefaults()

	// Should preserve custom scopes
	if len(req.Scopes) != 2 {
		t.Errorf("expected custom scopes to be preserved, got %d scopes", len(req.Scopes))
	}

	// Should preserve custom groups claim
	if req.GroupsClaim != "custom_groups" {
		t.Errorf("expected custom groups_claim to be preserved, got '%s'", req.GroupsClaim)
	}

	// Should preserve custom default role
	if req.DefaultRole != RoleAdmin {
		t.Errorf("expected custom default_role to be preserved, got '%s'", req.DefaultRole)
	}

	// Should preserve enabled=false
	if *req.Enabled != false {
		t.Error("expected enabled=false to be preserved")
	}

	// Should preserve auto_create_users=false
	if *req.AutoCreateUsers != false {
		t.Error("expected auto_create_users=false to be preserved")
	}

	// Should preserve role mapping
	if len(req.RoleMapping) != 1 {
		t.Errorf("expected role_mapping to be preserved, got %d entries", len(req.RoleMapping))
	}
}

func TestUpdateOIDCProviderRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       *UpdateOIDCProviderRequest
		wantErr   bool
		errFields []string
	}{
		{
			name:    "empty request is valid",
			req:     &UpdateOIDCProviderRequest{},
			wantErr: false,
		},
		{
			name: "valid display name update",
			req: &UpdateOIDCProviderRequest{
				DisplayName: stringPtr("New Display Name"),
			},
			wantErr: false,
		},
		{
			name: "display name too long",
			req: &UpdateOIDCProviderRequest{
				DisplayName: stringPtr(string(make([]byte, 300))),
			},
			wantErr:   true,
			errFields: []string{"display_name"},
		},
		{
			name: "valid issuer URL update",
			req: &UpdateOIDCProviderRequest{
				IssuerURL: stringPtr("https://new-issuer.example.com"),
			},
			wantErr: false,
		},
		{
			name: "invalid issuer URL",
			req: &UpdateOIDCProviderRequest{
				IssuerURL: stringPtr("not-a-url"),
			},
			wantErr:   true,
			errFields: []string{"issuer_url"},
		},
		{
			name: "invalid default role",
			req: &UpdateOIDCProviderRequest{
				DefaultRole: userRolePtr("superuser"),
			},
			wantErr:   true,
			errFields: []string{"default_role"},
		},
		{
			name: "valid default role - admin",
			req: &UpdateOIDCProviderRequest{
				DefaultRole: userRolePtr(string(RoleAdmin)),
			},
			wantErr: false,
		},
		{
			name: "valid default role - operator",
			req: &UpdateOIDCProviderRequest{
				DefaultRole: userRolePtr(string(RoleOperator)),
			},
			wantErr: false,
		},
		{
			name: "valid default role - viewer",
			req: &UpdateOIDCProviderRequest{
				DefaultRole: userRolePtr(string(RoleViewer)),
			},
			wantErr: false,
		},
		{
			name: "invalid role mapping value",
			req: &UpdateOIDCProviderRequest{
				RoleMapping: map[string]UserRole{"group": "invalid"},
			},
			wantErr:   true,
			errFields: []string{"role_mapping"},
		},
		{
			name: "valid role mapping",
			req: &UpdateOIDCProviderRequest{
				RoleMapping: map[string]UserRole{
					"admins":    RoleAdmin,
					"operators": RoleOperator,
					"users":     RoleViewer,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.req.Validate()
			if tt.wantErr && len(errors) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("unexpected validation error: %v", errors)
			}
			if tt.wantErr && len(tt.errFields) > 0 {
				for _, field := range tt.errFields {
					found := false
					for _, err := range errors {
						if err.Field == field {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error for field %s, but not found in errors: %v", field, errors)
					}
				}
			}
		})
	}
}

func TestOIDCAuthorizeRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       *OIDCAuthorizeRequest
		wantErr   bool
		errFields []string
	}{
		{
			name: "valid request",
			req: &OIDCAuthorizeRequest{
				RedirectURI: "https://app.example.com/auth/callback",
			},
			wantErr: false,
		},
		{
			name:      "missing redirect URI",
			req:       &OIDCAuthorizeRequest{},
			wantErr:   true,
			errFields: []string{"redirect_uri"},
		},
		{
			name: "invalid redirect URI",
			req: &OIDCAuthorizeRequest{
				RedirectURI: "not-a-url",
			},
			wantErr:   true,
			errFields: []string{"redirect_uri"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.req.Validate()
			if tt.wantErr && len(errors) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("unexpected validation error: %v", errors)
			}
		})
	}
}

func TestOIDCCallbackRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       *OIDCCallbackRequest
		wantErr   bool
		errFields []string
	}{
		{
			name: "valid request",
			req: &OIDCCallbackRequest{
				Code:  "authorization-code",
				State: "state-value",
			},
			wantErr: false,
		},
		{
			name: "missing code",
			req: &OIDCCallbackRequest{
				State: "state-value",
			},
			wantErr:   true,
			errFields: []string{"code"},
		},
		{
			name: "missing state",
			req: &OIDCCallbackRequest{
				Code: "authorization-code",
			},
			wantErr:   true,
			errFields: []string{"state"},
		},
		{
			name: "error from IdP",
			req: &OIDCCallbackRequest{
				Error: "access_denied",
			},
			wantErr:   true,
			errFields: []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.req.Validate()
			if tt.wantErr && len(errors) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("unexpected validation error: %v", errors)
			}
		})
	}
}

func TestOIDCProvider_ToSummary(t *testing.T) {
	provider := &OIDCProvider{
		Name:                  "google",
		DisplayName:           "Google Workspace",
		ProviderType:          OIDCProviderTypeGoogle,
		IssuerURL:             "https://accounts.google.com",
		ClientID:              "client-id",
		ClientSecretEncrypted: []byte("encrypted-secret"),
		Scopes:                []string{"openid", "profile", "email"},
		GroupsClaim:           "groups",
		RoleMapping:           map[string]UserRole{"admins": RoleAdmin},
		DefaultRole:           RoleViewer,
		Enabled:               true,
		AutoCreateUsers:       true,
	}

	summary := provider.ToSummary()

	// Verify fields are copied
	if summary.Name != provider.Name {
		t.Errorf("expected name %s, got %s", provider.Name, summary.Name)
	}
	if summary.DisplayName != provider.DisplayName {
		t.Errorf("expected display_name %s, got %s", provider.DisplayName, summary.DisplayName)
	}
	if summary.ProviderType != provider.ProviderType {
		t.Errorf("expected provider_type %s, got %s", provider.ProviderType, summary.ProviderType)
	}
	if summary.IssuerURL != provider.IssuerURL {
		t.Errorf("expected issuer_url %s, got %s", provider.IssuerURL, summary.IssuerURL)
	}
	if summary.ClientID != provider.ClientID {
		t.Errorf("expected client_id %s, got %s", provider.ClientID, summary.ClientID)
	}
	if len(summary.Scopes) != len(provider.Scopes) {
		t.Errorf("expected %d scopes, got %d", len(provider.Scopes), len(summary.Scopes))
	}
	if summary.GroupsClaim != provider.GroupsClaim {
		t.Errorf("expected groups_claim %s, got %s", provider.GroupsClaim, summary.GroupsClaim)
	}
	if len(summary.RoleMapping) != len(provider.RoleMapping) {
		t.Errorf("expected %d role mappings, got %d", len(provider.RoleMapping), len(summary.RoleMapping))
	}
	if summary.DefaultRole != provider.DefaultRole {
		t.Errorf("expected default_role %s, got %s", provider.DefaultRole, summary.DefaultRole)
	}
	if summary.Enabled != provider.Enabled {
		t.Errorf("expected enabled %v, got %v", provider.Enabled, summary.Enabled)
	}
	if summary.AutoCreateUsers != provider.AutoCreateUsers {
		t.Errorf("expected auto_create_users %v, got %v", provider.AutoCreateUsers, summary.AutoCreateUsers)
	}
}

func TestValidOIDCProviderTypes(t *testing.T) {
	validTypes := []OIDCProviderType{
		OIDCProviderTypeGoogle,
		OIDCProviderTypeOkta,
		OIDCProviderTypeAzureAD,
		OIDCProviderTypeAuth0,
		OIDCProviderTypeGeneric,
	}

	for _, pt := range validTypes {
		if !ValidOIDCProviderTypes[pt] {
			t.Errorf("expected %s to be a valid provider type", pt)
		}
	}

	// Check invalid type
	if ValidOIDCProviderTypes["invalid"] {
		t.Error("expected 'invalid' to not be a valid provider type")
	}
}

func TestIsValidProviderName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"lowercase letters", "google", true},
		{"with numbers", "google2", true},
		{"with hyphens", "google-workspace", true},
		{"uppercase letters", "Google", false},
		{"spaces", "google workspace", false},
		{"underscores", "google_workspace", false},
		{"starts with hyphen", "-google", false},
		{"ends with hyphen", "google-", false},
		{"special chars", "google@workspace", false},
		{"empty string", "", true}, // empty handled by required validation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidProviderName(tt.input)
			if result != tt.valid {
				t.Errorf("isValidProviderName(%q) = %v, want %v", tt.input, result, tt.valid)
			}
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func userRolePtr(s string) *UserRole {
	r := UserRole(s)
	return &r
}
