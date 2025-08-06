package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccEnvironmentTypeResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentTypeResourceConfig("example", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_environment_type.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("example"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_environment_type.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("example"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_environment_type.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
				},
			},
			// Update testing
			{
				Config: testAccEnvironmentTypeResourceConfig("example", "Example Environment Type"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_environment_type.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("example"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_environment_type.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("Example Environment Type"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_environment_type.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				ResourceName:      "humanitec_environment_type.test",
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
		resource "humanitec_environment_type" "test" {
			id = "` + id + `"
		}
		`
	}

	return `
	resource "humanitec_environment_type" "test" {
		id = "` + id + `"
		display_name = "` + display + `"
	}
	`
}
