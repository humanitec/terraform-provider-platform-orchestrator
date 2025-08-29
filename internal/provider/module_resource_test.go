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

func TestAccModuleResource(t *testing.T) {
	var (
		moduleId             = fmt.Sprintf("test-module-%d", time.Now().UnixNano())
		customTypeId         = fmt.Sprintf("custom-type-%d", time.Now().UnixNano())
		metricsTypeId        = fmt.Sprintf("metrics-%d", time.Now().UnixNano())
		postgresTypeId       = fmt.Sprintf("postgres-%d", time.Now().UnixNano())
		awsProviderId        = fmt.Sprintf("aws-provider-%d", time.Now().UnixNano())
		awsUpdatedProviderId = fmt.Sprintf("aws-updated-provider-%d", time.Now().UnixNano())
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccModuleResource(moduleId, customTypeId, metricsTypeId, postgresTypeId, awsProviderId, "s3://my-bucket/module.zip", "{}"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(moduleId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact(customTypeId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("module_source"),
						knownvalue.StringExact("s3://my-bucket/module.zip"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("module_inputs"),
						knownvalue.StringExact("{}"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("dependencies"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							"database": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":   knownvalue.StringExact(postgresTypeId),
								"class":  knownvalue.StringExact("default"),
								"id":     knownvalue.Null(),
								"params": knownvalue.Null(),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test module description"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("provider_mapping"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							"aws": knownvalue.StringExact("aws." + awsProviderId),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("coprovisioned"),
						knownvalue.ListPartial(map[int]knownvalue.Check{
							0: knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":                         knownvalue.StringExact(metricsTypeId),
								"class":                        knownvalue.StringExact("default"),
								"id":                           knownvalue.Null(),
								"params":                       knownvalue.StringExact(`{"level":"info"}`),
								"copy_dependents_from_current": knownvalue.Bool(false),
								"is_dependent_on_current":      knownvalue.Bool(true),
							}),
						}),
					),
				},
			},
			// Update testing
			{
				Config: testAccModuleResourceWithUpdate(moduleId, customTypeId, metricsTypeId, postgresTypeId, awsProviderId, awsUpdatedProviderId, "s3://my-bucket/module-v2.zip", "jsonencode({ region = \"us-east-1\" })", "Updated test module description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(moduleId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact(customTypeId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("module_source"),
						knownvalue.StringExact("s3://my-bucket/module-v2.zip"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("module_inputs"),
						knownvalue.StringExact(`{"region":"us-east-1"}`),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("dependencies"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							"database": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":   knownvalue.StringExact(customTypeId),
								"class":  knownvalue.StringExact("production"),
								"id":     knownvalue.StringExact("main-db"),
								"params": knownvalue.StringExact(`{"version":"14"}`),
							}),
							"cache": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":   knownvalue.StringExact(postgresTypeId),
								"class":  knownvalue.StringExact("default"),
								"id":     knownvalue.Null(),
								"params": knownvalue.Null(),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated test module description"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("provider_mapping"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							"aws": knownvalue.StringExact("aws." + awsUpdatedProviderId),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("coprovisioned"),
						knownvalue.ListPartial(map[int]knownvalue.Check{
							0: knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":                         knownvalue.StringExact(metricsTypeId),
								"class":                        knownvalue.StringExact("advanced"),
								"id":                           knownvalue.StringExact("mon-1"),
								"params":                       knownvalue.Null(),
								"copy_dependents_from_current": knownvalue.Bool(true),
								"is_dependent_on_current":      knownvalue.Bool(false),
							}),
						}),
					),
				},
			},
			{
				ResourceName:      "platform-orchestrator_module.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccModuleResourceWithSourceCode(t *testing.T) {
	var (
		moduleId       = fmt.Sprintf("test-module-%d", time.Now().UnixNano())
		customTypeId   = fmt.Sprintf("custom-type-%d", time.Now().UnixNano())
		postgresTypeId = fmt.Sprintf("postgres-%d", time.Now().UnixNano())
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccModuleResourceWithSourceCode(moduleId, customTypeId, `resource "aws_db_instance" "default" { engine = "postgres" }`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(moduleId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact(customTypeId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("module_source_code"),
						knownvalue.StringExact(`resource "aws_db_instance" "default" { engine = "postgres" }
`),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("coprovisioned"),
						knownvalue.ListSizeExact(0),
					),
				},
			},
			// Update testing
			{
				Config: testAccModuleResourceWithSourceCodeUpdate(moduleId, customTypeId, postgresTypeId, `resource "aws_db_instance" "default" { engine = "mysql" }`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(moduleId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact(customTypeId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("module_source_code"),
						knownvalue.StringExact(`resource "aws_db_instance" "default" { engine = "mysql" }
`),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module.test",
						tfjsonpath.New("coprovisioned"),
						knownvalue.ListPartial(map[int]knownvalue.Check{
							0: knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":                         knownvalue.StringExact(postgresTypeId),
								"class":                        knownvalue.StringExact("advanced"),
								"id":                           knownvalue.StringExact("postgres-1"),
								"params":                       knownvalue.StringExact(`{"interval":"5m"}`),
								"copy_dependents_from_current": knownvalue.Bool(true),
								"is_dependent_on_current":      knownvalue.Bool(false),
							}),
						}),
					),
				},
			},
		},
	})
}

func testAccModuleResource(id, customTypeId, metricsTypeId, postgresTypeId, awsProviderId, moduleSource, moduleInputs string) string {
	return `
resource "platform-orchestrator_resource_type" "custom_type" {
  id           =  "` + customTypeId + `"
  output_schema = "{}"
}

resource "platform-orchestrator_resource_type" "metrics" {
  id           =  "` + metricsTypeId + `"
  output_schema = "{}"
}

resource "platform-orchestrator_resource_type" "postgres" {
  id           =  "` + postgresTypeId + `"
  output_schema = "{}"
}

resource "platform-orchestrator_provider" "test_aws" {
  id = "` + awsProviderId + `"
  provider_type = "aws"
  source = "hashicorp/aws"
  version_constraint = ">= 4.0.0"
}

resource "platform-orchestrator_module" "test" {
  id             = "` + id + `"
  description    = "Test module description"
  resource_type  = platform-orchestrator_resource_type.custom_type.id
  module_source  = "` + moduleSource + `"
  module_inputs  = "` + moduleInputs + `"
  
  provider_mapping = {
    aws = "${platform-orchestrator_provider.test_aws.provider_type}.${platform-orchestrator_provider.test_aws.id}"
  }

  dependencies = {
    database = {
      type  = platform-orchestrator_resource_type.postgres.id
      class = "default"
    }
  }
  
  coprovisioned = [
    {
      type                         = platform-orchestrator_resource_type.metrics.id
      class                        = "default"
      params                       = jsonencode({"level": "info"})
      copy_dependents_from_current = false
      is_dependent_on_current      = true
    }
  ]
}
`
}

func testAccModuleResourceWithUpdate(id, customTypeId, metricsTypeId, postgresTypeId, awsProviderId, awsUpdatedProviderId, moduleSource, moduleInputs, description string) string {
	return `
resource "platform-orchestrator_resource_type" "custom_type" {
  id           =  "` + customTypeId + `"
  output_schema = "{}"
}

resource "platform-orchestrator_resource_type" "metrics" {
  id           =  "` + metricsTypeId + `"
  output_schema = "{}"
}

resource "platform-orchestrator_resource_type" "postgres" {
  id           =  "` + postgresTypeId + `"
  output_schema = "{}"
}

resource "platform-orchestrator_provider" "test_aws" {
  id = "` + awsProviderId + `"
  provider_type = "aws"
  source = "hashicorp/aws"
  version_constraint = ">= 4.0.0"
}

resource "platform-orchestrator_provider" "test_aws_updated" {
  id = "` + awsUpdatedProviderId + `"
  provider_type = "aws"
  source = "hashicorp/aws"
  version_constraint = ">= 4.0.0"
}

resource "platform-orchestrator_module" "test" {
  id             = "` + id + `"
  description    = "` + description + `"
  resource_type  = platform-orchestrator_resource_type.custom_type.id
  module_source  = "` + moduleSource + `"
  module_inputs  = ` + moduleInputs + `
  
  provider_mapping = {
    aws = "${platform-orchestrator_provider.test_aws_updated.provider_type}.${platform-orchestrator_provider.test_aws_updated.id}"
  }

  dependencies = {
    database = {
      type   = platform-orchestrator_resource_type.custom_type.id
      class  = "production"
      id     = "main-db"
      params = jsonencode({"version": "14"})
    }
    cache = {
      type  = platform-orchestrator_resource_type.postgres.id
      class = "default"
    }
  }
  
  coprovisioned = [
    {
      type                         = platform-orchestrator_resource_type.metrics.id
      class                        = "advanced"
      id                          = "mon-1"
      params                       = null
      copy_dependents_from_current = true
      is_dependent_on_current      = false
    }
  ]
}
`
}

func testAccModuleResourceWithSourceCode(id, customTypeId, sourceCode string) string {
	rs := `
resource "platform-orchestrator_resource_type" "custom_type" {
  id           =  "` + customTypeId + `"
  output_schema = "{}"
}

resource "platform-orchestrator_module" "test" {
  id                 = "` + id + `"
  resource_type      = platform-orchestrator_resource_type.custom_type.id
  module_source      = "inline"
  module_source_code =<<EOT
` + sourceCode + `
EOT
  
  coprovisioned = []
}
`
	return rs
}

func testAccModuleResourceWithSourceCodeUpdate(id, customTypeId, postgresTypeId, sourceCode string) string {
	rs := `
resource "platform-orchestrator_resource_type" "custom_type" {
  id           =  "` + customTypeId + `"
  output_schema = "{}"
}

resource "platform-orchestrator_resource_type" "postgres" {
  id           =  "` + postgresTypeId + `"
  output_schema = "{}"
}

resource "platform-orchestrator_module" "test" {
  id                 = "` + id + `"
  resource_type      = platform-orchestrator_resource_type.custom_type.id
  module_source      = "inline"
  module_source_code =<<EOT
` + sourceCode + `
EOT
  
  coprovisioned = [
    {
      type                         = platform-orchestrator_resource_type.postgres.id
      class                        = "advanced"
      id                          = "postgres-1"
      params                       = jsonencode({"interval": "5m"})
      copy_dependents_from_current = true
      is_dependent_on_current      = false
    }
  ]
}
`
	return rs
}
