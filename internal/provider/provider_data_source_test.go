package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProviderDataSource(t *testing.T) {
	var providerId = fmt.Sprintf("aws-provider-%d", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create provider resource and read via data source
			{
				Config: testAccProviderDataSourceConfig(providerId),
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct provider
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(providerId),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test",
						tfjsonpath.New("provider_type"),
						knownvalue.StringExact("aws"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test AWS Provider for data source"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test",
						tfjsonpath.New("source"),
						knownvalue.StringExact("hashicorp/aws"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test",
						tfjsonpath.New("version_constraint"),
						knownvalue.StringExact(">= 4.0.0"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test",
						tfjsonpath.New("configuration"),
						knownvalue.StringExact(`{"assume_role":{"role_arn":"arn:aws:iam::123456789012:role/HumanitecRole"},"region":"us-east-1"}`),
					),
				},
			},
		},
	})
}

func TestAccProviderDataSourceWithoutConfiguration(t *testing.T) {
	var providerId = fmt.Sprintf("aws-provider-%d", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create provider resource without configuration and read via data source
			{
				Config: testAccProviderDataSourceConfigWithoutConfiguration(providerId),
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct provider
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test_gcp",
						tfjsonpath.New("id"),
						knownvalue.StringExact(providerId),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test_gcp",
						tfjsonpath.New("provider_type"),
						knownvalue.StringExact("gcp"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test_gcp",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test GCP Provider for data source"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test_gcp",
						tfjsonpath.New("source"),
						knownvalue.StringExact("hashicorp/google"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test_gcp",
						tfjsonpath.New("version_constraint"),
						knownvalue.StringExact("~> 5.0"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_provider.test_gcp",
						tfjsonpath.New("configuration"),
						knownvalue.StringExact(`{}`),
					),
				},
			},
		},
	})
}

func testAccProviderDataSourceConfig(providerId string) string {
	return `
resource "platform-orchestrator_provider" "test" {
  id = "` + providerId + `"
  description = "Test AWS Provider for data source"
  provider_type = "aws"
  source = "hashicorp/aws"
  version_constraint = ">= 4.0.0"
  
  configuration = jsonencode({
    region = "us-east-1"
    assume_role = {
      role_arn = "arn:aws:iam::123456789012:role/HumanitecRole"
    }
  })
}

data "platform-orchestrator_provider" "test" {
  id = platform-orchestrator_provider.test.id
  provider_type = platform-orchestrator_provider.test.provider_type
}
`
}

func testAccProviderDataSourceConfigWithoutConfiguration(providerId string) string {
	return `
  resource "platform-orchestrator_provider" "test_gcp" {
  id = "` + providerId + `"
  description = "Test GCP Provider for data source"
  provider_type = "gcp"
  source = "hashicorp/google"
  version_constraint = "~> 5.0"
  configuration = "{}"
}

data "platform-orchestrator_provider" "test_gcp" {
  id = platform-orchestrator_provider.test_gcp.id
  provider_type = platform-orchestrator_provider.test_gcp.provider_type
}
`
}
