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

func TestAccProjectsDataSource(t *testing.T) {
	projectId := fmt.Sprintf("prod-%d", time.Now().UnixNano())

	cfg1 := fmt.Sprintf(`
resource "platform-orchestrator_project" "test" {
	id = "%[1]s"
}
`, projectId)
	cfg2 := cfg1 + `
data "platform-orchestrator_projects" "all" {
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
			// Read testing
			{
				Config: cfg2,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_projects.all",
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
