package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccModuleRuleResourceDefaultFields(t *testing.T) {
	var moduleId = fmt.Sprintf("test-module-%d", time.Now().UnixNano())
	var envTypeId = fmt.Sprintf("test-env-type-%d", time.Now().UnixNano())
	var resourceTypeId = fmt.Sprintf("custom-type-%d", time.Now().UnixNano())
	var ruleId uuid.UUID
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccModuleRuleResource(moduleId, resourceTypeId, envTypeId, "", ""),
				Check: func(s *terraform.State) error {
					ruleId = uuid.Must(uuid.Parse(s.RootModule().Resources["platform-orchestrator_module_rule.test"].Primary.ID))
					return nil
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact(resourceTypeId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("module_id"),
						knownvalue.StringExact(moduleId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("project_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("env_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("env_type_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("resource_class"),
						knownvalue.StringExact("default"),
					),
				},
			},
			// Update any field means re-creating the resource
			{
				Config: testAccModuleRuleResource(moduleId, resourceTypeId, envTypeId, "resource_class = \"custom-class\"", ""),
				Check: func(s *terraform.State) error {
					newRuleId := s.RootModule().Resources["platform-orchestrator_module_rule.test"].Primary.ID
					if newRuleId == ruleId.String() {
						return fmt.Errorf("expected new rule ID after update, got same ID: %s", newRuleId)
					}
					return nil
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact(resourceTypeId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("module_id"),
						knownvalue.StringExact(moduleId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("project_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("env_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("env_type_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("resource_class"),
						knownvalue.StringExact("custom-class"),
					),
				},
			},
			{
				ResourceName:      "platform-orchestrator_module_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccModuleRuleResource(t *testing.T) {
	var moduleId = fmt.Sprintf("test-module-%d", time.Now().UnixNano())
	var envTypeId = fmt.Sprintf("test-env-type-%d", time.Now().UnixNano())
	var resourceTypeId = fmt.Sprintf("custom-type-%d", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccModuleRuleResource(moduleId, resourceTypeId, envTypeId, "resource_class = \"custom-class\"", "env_type_id = platform-orchestrator_environment_type.test.id"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact(resourceTypeId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("module_id"),
						knownvalue.StringExact(moduleId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("project_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("env_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("env_type_id"),
						knownvalue.StringExact(envTypeId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_module_rule.test",
						tfjsonpath.New("resource_class"),
						knownvalue.StringExact("custom-class"),
					),
				},
			},
			{
				ResourceName:      "platform-orchestrator_module_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccModuleRuleResource(moduleId, resourceTypeId, envTypeId, classBlock, envTypeBlock string) string {
	return `
resource "platform-orchestrator_resource_type" "custom_type" {
  id           =  "` + resourceTypeId + `"
  output_schema = "{}"
}

resource "platform-orchestrator_environment_type" "test" {
  id             = "` + envTypeId + `"
}
 
resource "platform-orchestrator_module" "test" {
  id             = "` + moduleId + `"
  description    = "Test module description"
  resource_type  = platform-orchestrator_resource_type.custom_type.id
  module_source  = "s3://my-bucket/module.zip"
}

resource "platform-orchestrator_module_rule" "test" {
  module_id       = platform-orchestrator_module.test.id
  ` + classBlock + `
  ` + envTypeBlock + `
}
`
}
