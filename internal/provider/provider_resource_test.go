package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProviderResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProviderResourceConfig("test", "aws", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("test"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test Provider"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("configuration"),
						knownvalue.StringExact(`{}`),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("source"),
						knownvalue.StringExact("hashicorp/aws"),
					),
				},
			},
			// Update testing empty configuration
			{
				Config: testAccProviderResourceConfig("test", "aws", "{}"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("test"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test Provider"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("configuration"),
						knownvalue.StringExact(`{}`),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("source"),
						knownvalue.StringExact("hashicorp/aws"),
					),
				},
			},
			// Update testing
			{
				Config: testAccProviderResourceConfig("test", "aws", "{ region = \"us-west-2\" }"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("test"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test Provider"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("configuration"),
						knownvalue.StringExact(`{"region":"us-west-2"}`),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_provider.test",
						tfjsonpath.New("source"),
						knownvalue.StringExact("hashicorp/aws"),
					),
				},
			},
			{
				ResourceName: "platform-orchestrator_provider.test",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return "aws.test", nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProviderResourceConfig(id, providerType, configuration string) string {
	var configurationBlock string
	if configuration != "" {
		configurationBlock = "\n  configuration = jsonencode(" + configuration + ")"
	}

	return `
resource "platform-orchestrator_provider" "test" {
  id = "` + id + `"
  description = "Test Provider"
  provider_type = "` + providerType + `"
  source = "hashicorp/aws"
  version_constraint = ">= 1.0.0"
  ` + configurationBlock + `
}
`
}
