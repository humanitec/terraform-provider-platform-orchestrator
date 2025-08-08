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

func TestAccModuleRuleDataSource(t *testing.T) {
	var (
		moduleId       = fmt.Sprintf("test-module-%d", time.Now().UnixNano())
		envTypeId      = fmt.Sprintf("test-env-type-%d", time.Now().UnixNano())
		resourceTypeId = fmt.Sprintf("custom-type-%d", time.Now().UnixNano())
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create module rule resource and read via data source
			{
				Config: testAccModuleRuleDataSourceConfig(moduleId, resourceTypeId, envTypeId),
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct module rule
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test",
						tfjsonpath.New("module_id"),
						knownvalue.StringExact(moduleId),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact(resourceTypeId),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test",
						tfjsonpath.New("resource_class"),
						knownvalue.StringExact("custom-class"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test",
						tfjsonpath.New("env_type_id"),
						knownvalue.StringExact(envTypeId),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test",
						tfjsonpath.New("project_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test",
						tfjsonpath.New("env_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test",
						tfjsonpath.New("resource_id"),
						knownvalue.Null(),
					),
				},
			},
		},
	})
}

func TestAccModuleRuleDataSourceDefaultFields(t *testing.T) {
	var (
		moduleId       = fmt.Sprintf("test-module-%d", time.Now().UnixNano())
		envTypeId      = fmt.Sprintf("test-env-type-%d", time.Now().UnixNano())
		resourceTypeId = fmt.Sprintf("custom-type-%d", time.Now().UnixNano())
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create module rule resource with default fields and read via data source
			{
				Config: testAccModuleRuleDataSourceConfigDefault(moduleId, resourceTypeId, envTypeId),
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct module rule
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test_default",
						tfjsonpath.New("module_id"),
						knownvalue.StringExact(moduleId),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test_default",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact(resourceTypeId),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test_default",
						tfjsonpath.New("resource_class"),
						knownvalue.StringExact("default"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test_default",
						tfjsonpath.New("env_type_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test_default",
						tfjsonpath.New("project_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test_default",
						tfjsonpath.New("env_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_module_rule.test_default",
						tfjsonpath.New("resource_id"),
						knownvalue.Null(),
					),
				},
			},
		},
	})
}

func testAccModuleRuleDataSourceConfig(moduleId, resourceTypeId, envTypeId string) string {
	return `
resource "humanitec_resource_type" "custom_type" {
  id           =  "` + resourceTypeId + `"
  output_schema = "{}"
}

resource "humanitec_environment_type" "test" {
  id             = "` + envTypeId + `"
}
 
resource "humanitec_module" "test" {
  id             = "` + moduleId + `"
  description    = "Test module description"
  resource_type  = humanitec_resource_type.custom_type.id
  module_source  = "s3://my-bucket/module.zip"
}

resource "humanitec_module_rule" "test" {
  module_id       = humanitec_module.test.id
  resource_class  = "custom-class"
  env_type_id     = humanitec_environment_type.test.id
}

data "humanitec_module_rule" "test" {
  id = humanitec_module_rule.test.id
}
`
}

func testAccModuleRuleDataSourceConfigDefault(moduleId, resourceTypeId, envTypeId string) string {
	return `
resource "humanitec_resource_type" "custom_type" {
  id           =  "` + resourceTypeId + `"
  output_schema = "{}"
}

resource "humanitec_environment_type" "test" {
  id             = "` + envTypeId + `"
}
 
resource "humanitec_module" "test" {
  id             = "` + moduleId + `"
  description    = "Test module description"
  resource_type  = humanitec_resource_type.custom_type.id
  module_source  = "s3://my-bucket/module.zip"
}

resource "humanitec_module_rule" "test_default" {
  module_id       = humanitec_module.test.id
}

data "humanitec_module_rule" "test_default" {
  id = humanitec_module_rule.test_default.id
}
`
}
