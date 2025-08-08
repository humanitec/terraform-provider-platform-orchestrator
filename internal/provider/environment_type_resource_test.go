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

func TestAccEnvironmentTypeResource(t *testing.T) {
	envTypeId := fmt.Sprintf("example-%d", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentTypeResourceConfig(envTypeId, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_environment_type.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(envTypeId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_environment_type.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact(envTypeId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_environment_type.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
				},
			},
			// Update testing
			{
				Config: testAccEnvironmentTypeResourceConfig(envTypeId, "Example Environment Type"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_environment_type.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(envTypeId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_environment_type.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("Example Environment Type"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_environment_type.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				ResourceName:      "platform-orchestrator_environment_type.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccEnvironmentTypeResourceConfig(id, display string) string {
	if display == "" {
		return `
		resource "platform-orchestrator_environment_type" "test" {
			id = "` + id + `"
		}
		`
	}

	return `
	resource "platform-orchestrator_environment_type" "test" {
		id = "` + id + `"
		display_name = "` + display + `"
	}
	`
}
