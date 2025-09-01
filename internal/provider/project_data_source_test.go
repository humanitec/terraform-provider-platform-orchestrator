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

func TestAccProjectDataSource(t *testing.T) {
	projectId := fmt.Sprintf("prod-%d", time.Now().UnixNano())

	cfg1 := fmt.Sprintf(`
resource "platform-orchestrator_project" "test" {
	id = "%[1]s"
}
`, projectId)
	cfg2 := cfg1 + `
data "platform-orchestrator_project" "test" {
  id = platform-orchestrator_project.test.id
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// First create the project
			{
				Config: cfg1,
			},
			// Then try to read it with the data source
			{
				Config: cfg2,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_project.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(projectId),
					),
				},
			},
		},
	})
}
