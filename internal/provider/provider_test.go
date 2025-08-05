package provider

import (
	"context"
	"encoding/base64"
	"maps"
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
	"humanitec": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	checkEnvVar(t, "HUMANITEC_ORG_ID")
	checkEnvVar(t, "HUMANITEC_AUTH_TOKEN")
}

func checkEnvVar(t *testing.T, name string) {
	if v := os.Getenv(name); v == "" {
		t.Fatalf("Missing environment variable %s", name)
	}
}

func testAccGetCanyonCPClient(t *testing.T) (canyoncp.ClientWithResponsesInterface, string) {
	t.Helper()

	orgId := os.Getenv(HUM_ORG_ID_ENV_VAR)
	authToken := os.Getenv(HUM_AUTH_TOKEN_ENV_VAR)
	var apiUrl = HUM_DEFAULT_API_URL
	if os.Getenv(HUM_API_URL_ENV_VAR) != "" {
		apiUrl = os.Getenv(HUM_API_URL_ENV_VAR)
	}

	if orgId == "" || authToken == "" {
		t.Fatal("Missing required environment variables for client creation")
	}

	extraHeaders := make(http.Header)
	extraHeaders.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(orgId+":"+authToken)))

	client := &http.Client{
		Transport: retryhttp.New(),
		Timeout:   30 * time.Second,
	}

	cpc, err := canyoncp.NewClientWithResponses(apiUrl, canyoncp.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		maps.Copy(req.Header, extraHeaders)
		return nil
	}), canyoncp.WithHTTPClient(client))
	if err != nil {
		t.Fatalf("Failed to create canyon CP client: %v", err)
	}

	return cpc, orgId
}
