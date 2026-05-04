package provider

import (
	"cmp"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cp "terraform-provider-platform-orchestrator/internal/clients/platform-orchestrator-cp"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/justinrixx/retryhttp"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"platform-orchestrator": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	checkEnvVar(t, PO_ORG_ID_ENV_VAR)
	checkEnvVar(t, PO_AUTH_TOKEN_ENV_VAR)
}

func checkEnvVar(t *testing.T, name string) {
	if v := os.Getenv(name); v == "" {
		t.Fatalf("Missing environment variable %s", name)
	}
}

func NewPlatformOrchestratorControlPlaneClient(t *testing.T) *cp.ClientWithResponses {
	cpc, err := cp.NewClientWithResponses(cmp.Or(os.Getenv(PO_API_URL_ENV_VAR), PO_DEFAULT_API_URL), cp.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+os.Getenv(PO_AUTH_TOKEN_ENV_VAR))
		return nil
	}), cp.WithHTTPClient(&http.Client{
		Transport: retryhttp.New(),
		Timeout:   30 * time.Second,
	}))
	if err != nil {
		t.Fatalf("Error creating Platform Orchestrator Controlplane client: %s", err)
	}
	return cpc
}

func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv(PO_API_URL_ENV_VAR, "")
	t.Setenv(PO_ORG_ID_ENV_VAR, "")
	t.Setenv(PO_AUTH_TOKEN_ENV_VAR, "")
}

func TestLoadClientConfig_basic(t *testing.T) {
	clearEnv(t)
	d := new(diag.Diagnostics)
	u, o, a := loadClientConfig(t.Context(), PlatformOrchestratorProviderModel{
		ApiUrl:    types.StringValue("https://some-api.com"),
		OrgId:     types.StringValue("some-org"),
		AuthToken: types.StringValue("some-token"),
	}, d)
	assert.Equal(t, "https://some-api.com", u)
	assert.Equal(t, "some-org", o)
	assert.Equal(t, "some-token", a)
	assert.Empty(t, d.Errors())
	assert.Empty(t, d.Warnings())
}

func TestLoadClientConfig_with_file(t *testing.T) {
	clearEnv(t)
	tf := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(tf, []byte(`{"default_org_id": "some-org", "token": "some-token"}`), 0600))

	d := new(diag.Diagnostics)
	u, o, a := loadClientConfig(t.Context(), PlatformOrchestratorProviderModel{
		ConfigFilePath: types.StringValue(tf),
		ApiUrl:         types.StringValue("https://some-api.com"),
	}, d)
	assert.Equal(t, "https://some-api.com", u)
	assert.Equal(t, "some-org", o)
	assert.Equal(t, "some-token", a)
	assert.Empty(t, d.Errors())
	assert.Empty(t, d.Warnings())
}

func TestLoadClientConfig_with_env(t *testing.T) {
	clearEnv(t)
	t.Setenv(PO_ORG_ID_ENV_VAR, "another-org")
	t.Setenv(PO_AUTH_TOKEN_ENV_VAR, "a-token")
	d := new(diag.Diagnostics)
	u, o, a := loadClientConfig(t.Context(), PlatformOrchestratorProviderModel{}, d)
	// FIXME: Update the default API URL to match the new domain.
	assert.Equal(t, "[Default api url]", u)
	assert.Equal(t, "another-org", o)
	assert.Equal(t, "a-token", a)
	assert.Empty(t, d.Errors())
	assert.Empty(t, d.Warnings())
}

func TestLoadClientConfig_with_fallback_file(t *testing.T) {
	clearEnv(t)
	td := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", td)
	cd, _ := os.UserConfigDir()
	require.Contains(t, cd, td)
	require.NoError(t, os.MkdirAll(filepath.Join(cd, "hctl"), 0700))
	tf := filepath.Join(cd, "hctl", "config.yaml")
	require.NoError(t, os.WriteFile(tf, []byte(`{"default_org_id": "some-org", "token": "some-token"}`), 0600))
	d := new(diag.Diagnostics)

	ctx := tflogtest.RootLogger(t.Context(), os.Stdout)
	u, o, a := loadClientConfig(ctx, PlatformOrchestratorProviderModel{}, d)
	// FIXME: Update the default API URL to match the new domain.
	assert.Equal(t, "[Default api url]", u)
	assert.Equal(t, "some-org", o)
	assert.Equal(t, "some-token", a)
	assert.Empty(t, d.Errors())
	assert.Empty(t, d.Warnings())
}
