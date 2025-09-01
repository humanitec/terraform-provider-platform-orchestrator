package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccExampleProjectsDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_projects.test",
						tfjsonpath.New("projects"),
						knownvalue.ListPartial(map[int]knownvalue.Check{
							0: knownvalue.NotNull(),
						}),
					),
				},
			},
		},
	})
}

const testAccExampleProjectsDataSourceConfig = `
resource "platform-orchestrator_project" "test" {
	id = "example"
	display_name = "Example Environment Type"
}

data "platform-orchestrator_projects" "test" {
}
`
