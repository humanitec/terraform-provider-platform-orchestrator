package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccExampleProjectDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_project.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("example"),
					),
				},
			},
		},
	})
}

const testAccExampleProjectDataSourceConfig = `
resource "platform-orchestrator_project" "test" {
	id = "example"
	display_name = "Example Environment Type"
}

data "platform-orchestrator_project" "test" {
  id = platform-orchestrator_project.test.id
}
`
