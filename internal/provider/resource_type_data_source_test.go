package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccResourceTypeDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccResourceTypeDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.humanitec_resource_type.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("example"),
					),
				},
			},
		},
	})
}

const testAccResourceTypeDataSourceConfig = `
resource "humanitec_resource_type" "test" {
	id = "example"
	output_schema = "{}"
}
	
data "humanitec_resource_type" "test" {
  id = humanitec_resource_type.test.id
}
`
