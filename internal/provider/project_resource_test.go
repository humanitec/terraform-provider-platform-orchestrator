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

func TestAccProjectResource(t *testing.T) {
	projectId := fmt.Sprintf("prod-%d", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfig(projectId, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(projectId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact(projectId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("created_at"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("updated_at"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("foo"),
					),
				},
			},
			// Update testing
			{
				Config: testAccProjectResourceConfig(projectId, "Example Project"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(projectId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("Example Project"),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("created_at"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("updated_at"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_project.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("active"),
					),
				},
			},
			{
				ResourceName:      "platform-orchestrator_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectResourceConfig(id, display string) string {
	if display == "" {
		return `
		resource "platform-orchestrator_project" "test" {
			id = "` + id + `"
		}
		`
	}

	return `
	resource "platform-orchestrator_project" "test" {
		id = "` + id + `"
		display_name = "` + display + `"
	}
	`
}
