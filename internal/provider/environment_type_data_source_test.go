package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccEnvironmentTypeDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccExampleDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_environment_type.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("example"),
					),
				},
			},
		},
	})
}

const testAccExampleDataSourceConfig = `
resource "platform-orchestrator_environment_type" "test" {
	id = "example"
	display_name = "Example Environment Type"
}

data "platform-orchestrator_environment_type" "test" {
  id = platform-orchestrator_environment_type.test.id
}
`
