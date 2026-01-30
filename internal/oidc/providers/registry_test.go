package providers

import (
	"testing"

	"github.com/janovincze/philotes/internal/api/models"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	// Should have all 5 provider types registered
	providerTypes := []models.OIDCProviderType{
		models.OIDCProviderTypeGoogle,
		models.OIDCProviderTypeOkta,
		models.OIDCProviderTypeAzureAD,
		models.OIDCProviderTypeAuth0,
		models.OIDCProviderTypeGeneric,
	}

	for _, pt := range providerTypes {
		config, ok := registry.Get(pt)
		if !ok {
			t.Errorf("expected provider config for %s, got none", pt)
		}
		if config == nil {
			t.Errorf("expected non-nil config for %s", pt)
		}
	}
}

func TestRegistry_Get_Unknown(t *testing.T) {
	registry := NewRegistry()

	config, ok := registry.Get("unknown-provider")
	if ok {
		t.Error("expected ok=false for unknown provider type")
	}
	if config != nil {
		t.Error("expected nil for unknown provider type")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	configs := registry.List()
	if len(configs) != 5 {
		t.Errorf("expected 5 provider configs, got %d", len(configs))
	}
}

func TestRegistry_ApplyDefaults(t *testing.T) {
	registry := NewRegistry()

	// Test with Google provider
	req := &models.CreateOIDCProviderRequest{
		Name:         "google",
		DisplayName:  "Google",
		ProviderType: models.OIDCProviderTypeGoogle,
		IssuerURL:    "https://accounts.google.com",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}

	registry.ApplyDefaults(req)

	// Should have default scopes applied
	if len(req.Scopes) == 0 {
		t.Error("expected scopes to be applied")
	}

	// Should have groups claim applied
	if req.GroupsClaim == "" {
		t.Error("expected groups_claim to be applied")
	}
}

func TestRegistry_ApplyDefaults_PreservesExisting(t *testing.T) {
	registry := NewRegistry()

	req := &models.CreateOIDCProviderRequest{
		Name:         "google",
		DisplayName:  "Google",
		ProviderType: models.OIDCProviderTypeGoogle,
		IssuerURL:    "https://accounts.google.com",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		Scopes:       []string{"openid", "custom-scope"},
		GroupsClaim:  "custom_groups",
	}

	registry.ApplyDefaults(req)

	// Should preserve custom scopes
	if len(req.Scopes) != 2 {
		t.Errorf("expected custom scopes to be preserved, got %d", len(req.Scopes))
	}

	// Should preserve custom groups claim
	if req.GroupsClaim != "custom_groups" {
		t.Errorf("expected custom groups_claim to be preserved, got %s", req.GroupsClaim)
	}
}

func TestGoogleConfig(t *testing.T) {
	config := &GoogleConfig{}

	// Check type
	if config.Type() != models.OIDCProviderTypeGoogle {
		t.Errorf("expected type 'google', got '%s'", config.Type())
	}

	// Check default issuer URL
	if config.DefaultIssuerURL() != "https://accounts.google.com" {
		t.Errorf("expected Google issuer URL, got '%s'", config.DefaultIssuerURL())
	}

	// Check default scopes
	scopes := config.DefaultScopes()
	if len(scopes) < 3 {
		t.Error("expected at least 3 default scopes")
	}
	hasOpenID := false
	for _, s := range scopes {
		if s == "openid" {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		t.Error("expected 'openid' in default scopes")
	}

	// Check groups claim
	if config.DefaultGroupsClaim() == "" {
		t.Error("expected non-empty groups claim")
	}

	// Should support discovery
	if !config.SupportsDiscovery() {
		t.Error("expected Google to support discovery")
	}
}

func TestOktaConfig(t *testing.T) {
	config := &OktaConfig{}

	// Check type
	if config.Type() != models.OIDCProviderTypeOkta {
		t.Errorf("expected type 'okta', got '%s'", config.Type())
	}

	// Okta doesn't have a default issuer (it's tenant-specific)
	if config.DefaultIssuerURL() != "" {
		t.Errorf("expected empty default issuer for Okta, got '%s'", config.DefaultIssuerURL())
	}

	// Check default scopes
	scopes := config.DefaultScopes()
	hasGroups := false
	for _, s := range scopes {
		if s == "groups" {
			hasGroups = true
			break
		}
	}
	if !hasGroups {
		t.Error("expected 'groups' in Okta default scopes")
	}

	// Should support discovery
	if !config.SupportsDiscovery() {
		t.Error("expected Okta to support discovery")
	}
}

func TestAzureADConfig(t *testing.T) {
	config := &AzureADConfig{}

	// Check type
	if config.Type() != models.OIDCProviderTypeAzureAD {
		t.Errorf("expected type 'azure_ad', got '%s'", config.Type())
	}

	// Azure AD doesn't have a default issuer (it's tenant-specific)
	if config.DefaultIssuerURL() != "" {
		t.Errorf("expected empty default issuer for Azure AD, got '%s'", config.DefaultIssuerURL())
	}

	// Check default scopes
	scopes := config.DefaultScopes()
	if len(scopes) < 3 {
		t.Error("expected at least 3 default scopes for Azure AD")
	}

	// Should support discovery
	if !config.SupportsDiscovery() {
		t.Error("expected Azure AD to support discovery")
	}
}

func TestAuth0Config(t *testing.T) {
	config := &Auth0Config{}

	// Check type
	if config.Type() != models.OIDCProviderTypeAuth0 {
		t.Errorf("expected type 'auth0', got '%s'", config.Type())
	}

	// Auth0 doesn't have a default issuer (it's tenant-specific)
	if config.DefaultIssuerURL() != "" {
		t.Errorf("expected empty default issuer for Auth0, got '%s'", config.DefaultIssuerURL())
	}

	// Check default scopes
	scopes := config.DefaultScopes()
	if len(scopes) < 3 {
		t.Error("expected at least 3 default scopes for Auth0")
	}

	// Should support discovery
	if !config.SupportsDiscovery() {
		t.Error("expected Auth0 to support discovery")
	}
}

func TestGenericConfig(t *testing.T) {
	config := &GenericConfig{}

	// Check type
	if config.Type() != models.OIDCProviderTypeGeneric {
		t.Errorf("expected type 'generic', got '%s'", config.Type())
	}

	// Generic doesn't have a default issuer
	if config.DefaultIssuerURL() != "" {
		t.Errorf("expected empty default issuer for generic, got '%s'", config.DefaultIssuerURL())
	}

	// Check default scopes
	scopes := config.DefaultScopes()
	expectedScopes := []string{"openid", "profile", "email"}
	if len(scopes) != len(expectedScopes) {
		t.Errorf("expected %d default scopes, got %d", len(expectedScopes), len(scopes))
	}
	for i, s := range expectedScopes {
		if scopes[i] != s {
			t.Errorf("expected scope %s at position %d, got %s", s, i, scopes[i])
		}
	}

	// Check groups claim
	if config.DefaultGroupsClaim() != "groups" {
		t.Errorf("expected groups claim 'groups', got '%s'", config.DefaultGroupsClaim())
	}

	// Should support discovery
	if !config.SupportsDiscovery() {
		t.Error("expected generic to support discovery")
	}
}

func TestProviderConfig_Interface(t *testing.T) {
	// Verify all configs implement the ProviderConfig interface
	var _ ProviderConfig = &GoogleConfig{}
	var _ ProviderConfig = &OktaConfig{}
	var _ ProviderConfig = &AzureADConfig{}
	var _ ProviderConfig = &Auth0Config{}
	var _ ProviderConfig = &GenericConfig{}
}
