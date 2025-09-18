package provider

import (
	"cmp"
	"context"
	"net/http"
	"os"
	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
	"testing"
	"time"

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
	checkEnvVar(t, HUM_ORG_ID_ENV_VAR)
	checkEnvVar(t, HUM_AUTH_TOKEN_ENV_VAR)
}

func checkEnvVar(t *testing.T, name string) {
	if v := os.Getenv(name); v == "" {
		t.Fatalf("Missing environment variable %s", name)
	}
}

func NewPlatformOrchestratorControlPlaneClient(t *testing.T) *canyoncp.ClientWithResponses {
	cpc, err := canyoncp.NewClientWithResponses(cmp.Or(os.Getenv(HUM_API_URL_ENV_VAR), HUM_DEFAULT_API_URL), canyoncp.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+os.Getenv(HUM_AUTH_TOKEN_ENV_VAR))
		return nil
	}), canyoncp.WithHTTPClient(&http.Client{
		Transport: retryhttp.New(),
		Timeout:   30 * time.Second,
	}))
	if err != nil {
		t.Fatalf("Error creating Platform Orchestrator Controlplane client: %s", err)
	}
	return cpc
}
