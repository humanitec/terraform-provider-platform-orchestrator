package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
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
