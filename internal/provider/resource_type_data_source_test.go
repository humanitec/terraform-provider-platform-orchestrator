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

func TestAccResourceTypeDataSource(t *testing.T) {
	var resourceTypeId = fmt.Sprintf("aws-rds-%d", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccResourceTypeDataSourceConfig(resourceTypeId),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_resource_type.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(resourceTypeId),
					),
				},
			},
		},
	})
}

func testAccResourceTypeDataSourceConfig(resourceTypeId string) string {
	return `
resource "platform-orchestrator_resource_type" "test" {
	id = "` + resourceTypeId + `"
	output_schema = "{}"
}
	
data "platform-orchestrator_resource_type" "test" {
  id = platform-orchestrator_resource_type.test.id
}
`
}
