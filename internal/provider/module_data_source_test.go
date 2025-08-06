package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccModuleDataSourceWithSourceCode(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create module resource with source code and read via data source
			{
				Config: testAccModuleDataSourceConfigWithSourceCode,
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct module
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_source_code",
						tfjsonpath.New("id"),
						knownvalue.StringExact("tf-module-source-code-data-test"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_source_code",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test Module with source code for data source"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_source_code",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact("aws-rds"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_source_code",
						tfjsonpath.New("module_source_code"),
						knownvalue.StringExact(`resource "aws_db_instance" "example" {
  identifier = var.identifier
  engine     = "postgres"
}
`),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_source_code",
						tfjsonpath.New("module_source"),
						knownvalue.Null(),
					),
				},
			},
		},
	})
}

func TestAccModuleDataSourceWithComplexStructure(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create module resource with complex structure and read via data source
			{
				Config: testAccModuleDataSourceConfigWithComplexStructure,
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct module
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_complex",
						tfjsonpath.New("id"),
						knownvalue.StringExact("tf-module-complex-data-test"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_complex",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact("postgres"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_complex",
						tfjsonpath.New("module_source"),
						knownvalue.StringExact("git::https://github.com/test/postgres-module"),
					),
					// Verify coprovisioned structure
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_complex",
						tfjsonpath.New("coprovisioned").AtSliceIndex(0).AtMapKey("type"),
						knownvalue.StringExact("logging"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_complex",
						tfjsonpath.New("coprovisioned").AtSliceIndex(0).AtMapKey("is_dependent_on_current"),
						knownvalue.Bool(true),
					),
					// Verify dependencies structure
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_complex",
						tfjsonpath.New("dependencies").AtMapKey("vpc").AtMapKey("type"),
						knownvalue.StringExact("aws-vpc"),
					),
					// Verify provider mapping
					statecheck.ExpectKnownValue(
						"data.humanitec_module.test_complex",
						tfjsonpath.New("provider_mapping").AtMapKey("aws"),
						knownvalue.StringExact("aws.tf-provider-aws-test"),
					),
				},
			},
		},
	})
}

const testAccModuleDataSourceConfigWithSourceCode = `
resource "humanitec_resource_type" "aws_rds" {
  id = "aws-rds"
  description = "Postgres Database"
  output_schema = jsonencode({})
}

resource "humanitec_module" "test_source_code" {
  id = "tf-module-source-code-data-test"
  description = "Test Module with source code for data source"
  resource_type = humanitec_resource_type.aws_rds.id
  
  module_source_code = <<-EOT
resource "aws_db_instance" "example" {
  identifier = var.identifier
  engine     = "postgres"
}
EOT
}

data "humanitec_module" "test_source_code" {
  id = humanitec_module.test_source_code.id
}
`

const testAccModuleDataSourceConfigWithComplexStructure = `
resource "humanitec_provider" "test_aws" {
  id = "tf-provider-aws-test"
  description = "Test AWS Provider"
  provider_type = "aws"
  source = "hashicorp/aws"
  version_constraint = ">= 4.0.0"
}

resource "humanitec_resource_type" "postgres" {
  id = "postgres"
  description = "Postgres Database"
  output_schema = jsonencode({})
}

resource "humanitec_resource_type" "logging" {
  id = "logging"
  description = "Logging Resource"
  output_schema = jsonencode({})
}

resource "humanitec_resource_type" "aws_vpc" {
  id = "aws-vpc"
  description = "AWS VPC"
  output_schema = jsonencode({})
}

resource "humanitec_module" "test_complex" {
  id = "tf-module-complex-data-test"
  description = "Test Module with complex structure"
  resource_type = humanitec_resource_type.postgres.id
  module_source = "git::https://github.com/test/postgres-module"
  
  module_inputs = jsonencode({
    instance_class = "db.t3.micro"
    allocated_storage = 20
  })

  provider_mapping = {
    aws = "${humanitec_provider.test_aws.provider_type}.${humanitec_provider.test_aws.id}"
  }

  coprovisioned = [{
    type = humanitec_resource_type.logging.id
    is_dependent_on_current = true
    params = jsonencode({
      log_group = "/aws/rds/postgres"
    })
  }]

  dependencies = {
    vpc = {
      type = humanitec_resource_type.aws_vpc.id
      class = "default"
    }
  }
}

data "humanitec_module" "test_complex" {
  id = humanitec_module.test_complex.id
}
`
