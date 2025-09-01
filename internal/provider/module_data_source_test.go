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

func TestAccModuleDataSourceWithSourceCode(t *testing.T) {
	var (
		moduleId     = fmt.Sprintf("test-module-%d", time.Now().UnixNano())
		awsRdsTypeId = fmt.Sprintf("custom-type-%d", time.Now().UnixNano())
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create module resource with source code and read via data source
			{
				Config: testAccModuleDataSourceConfigWithSourceCode(moduleId, awsRdsTypeId),
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct module
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_source_code",
						tfjsonpath.New("id"),
						knownvalue.StringExact(moduleId),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_source_code",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test Module with source code for data source"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_source_code",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact(awsRdsTypeId),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_source_code",
						tfjsonpath.New("module_source_code"),
						knownvalue.StringExact(`resource "aws_db_instance" "example" {
  identifier = var.identifier
  engine     = "postgres"
}
`),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_source_code",
						tfjsonpath.New("module_source"),
						knownvalue.Null(),
					),
				},
			},
		},
	})
}

func TestAccModuleDataSourceWithComplexStructure(t *testing.T) {
	var (
		moduleId       = fmt.Sprintf("test-module-%d", time.Now().UnixNano())
		postgresTypeId = fmt.Sprintf("postgres-%d", time.Now().UnixNano())
		awsVpcTypeId   = fmt.Sprintf("aws-vpc-%d", time.Now().UnixNano())
		loggingTypeId  = fmt.Sprintf("logging-%d", time.Now().UnixNano())
		providerId     = fmt.Sprintf("aws-provider-%d", time.Now().UnixNano())
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create module resource with complex structure and read via data source
			{
				Config: testAccModuleDataSourceConfigWithComplexStructure(moduleId, postgresTypeId, awsVpcTypeId, loggingTypeId, providerId),
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct module
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_complex",
						tfjsonpath.New("id"),
						knownvalue.StringExact(moduleId),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_complex",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact(postgresTypeId),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_complex",
						tfjsonpath.New("module_source"),
						knownvalue.StringExact("git::https://github.com/test/postgres-module"),
					),
					// Verify coprovisioned structure
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_complex",
						tfjsonpath.New("coprovisioned").AtSliceIndex(0).AtMapKey("type"),
						knownvalue.StringExact(loggingTypeId),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_complex",
						tfjsonpath.New("coprovisioned").AtSliceIndex(0).AtMapKey("is_dependent_on_current"),
						knownvalue.Bool(true),
					),
					// Verify dependencies structure
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_complex",
						tfjsonpath.New("dependencies").AtMapKey("vpc").AtMapKey("type"),
						knownvalue.StringExact(awsVpcTypeId),
					),
					// Verify provider mapping
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_module.test_complex",
						tfjsonpath.New("provider_mapping").AtMapKey("aws"),
						knownvalue.StringExact("aws."+providerId),
					),
				},
			},
		},
	})
}

func testAccModuleDataSourceConfigWithSourceCode(id, awsRdsTypeId string) string {
	return `
resource "platform-orchestrator_resource_type" "aws_rds" {
  id = "` + awsRdsTypeId + `"
  description = "Postgres Database"
  output_schema = jsonencode({})
}

resource "platform-orchestrator_module" "test_source_code" {
  id = "` + id + `"
  description = "Test Module with source code for data source"
  resource_type = platform-orchestrator_resource_type.aws_rds.id

  module_source = "inline"
  module_source_code = <<-EOT
resource "aws_db_instance" "example" {
  identifier = var.identifier
  engine     = "postgres"
}
EOT
}

data "platform-orchestrator_module" "test_source_code" {
  id = platform-orchestrator_module.test_source_code.id
}
`
}

func testAccModuleDataSourceConfigWithComplexStructure(moduleId, postgresTypeId, awsVpcTypeId, loggingTypeId, awsProviderId string) string {
	return `
resource "platform-orchestrator_provider" "test_aws" {
  id = "` + awsProviderId + `"
  description = "Test AWS Provider"
  provider_type = "aws"
  source = "hashicorp/aws"
  version_constraint = ">= 4.0.0"
}

resource "platform-orchestrator_resource_type" "postgres" {
  id = "` + postgresTypeId + `"
  description = "Postgres Database"
  output_schema = jsonencode({})
}

resource "platform-orchestrator_resource_type" "logging" {
  id = "` + loggingTypeId + `"
  description = "Logging Resource"
  output_schema = jsonencode({})
}

resource "platform-orchestrator_resource_type" "aws_vpc" {
  id = "` + awsVpcTypeId + `"
  description = "AWS VPC"
  output_schema = jsonencode({})
}

resource "platform-orchestrator_module" "test_complex" {
  id = "` + moduleId + `"
  description = "Test Module with complex structure"
  resource_type = platform-orchestrator_resource_type.postgres.id
  module_source = "git::https://github.com/test/postgres-module"
  
  module_inputs = jsonencode({
    instance_class = "db.t3.micro"
    allocated_storage = 20
  })

  provider_mapping = {
    aws = "${platform-orchestrator_provider.test_aws.provider_type}.${platform-orchestrator_provider.test_aws.id}"
  }

  coprovisioned = [{
    type = platform-orchestrator_resource_type.logging.id
    is_dependent_on_current = true
    params = jsonencode({
      log_group = "/aws/rds/postgres"
    })
  }]

  dependencies = {
    vpc = {
      type = platform-orchestrator_resource_type.aws_vpc.id
      class = "default"
    }
  }
}

data "platform-orchestrator_module" "test_complex" {
  id = platform-orchestrator_module.test_complex.id
}
`
}
