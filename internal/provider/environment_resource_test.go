package provider

import (
	"crypto/rand"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentResource(t *testing.T) {
	envTypeId := "development-" + strings.ToLower(rand.Text())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentResourceConfig("test-env", "test-project", envTypeId, "Test Environment"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "id", "test-env"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "project_id", "test-project"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "env_type_id", envTypeId),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "display_name", "Test Environment"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "uuid"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "created_at"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "updated_at"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "status", "active"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "platform-orchestrator_environment.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "test-project/test-env",
			},
			// Update and Read testing
			{
				Config: testAccEnvironmentResourceConfig("test-env", "test-project", envTypeId, "Updated Test Environment"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "id", "test-env"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "project_id", "test-project"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "env_type_id", envTypeId),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "display_name", "Updated Test Environment"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "uuid"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "created_at"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "updated_at"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "status", "active"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccEnvironmentResourceMinimalConfig(t *testing.T) {
	envTypeId := "development-" + strings.ToLower(rand.Text())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with minimal configuration
			{
				Config: testAccEnvironmentResourceMinimalConfig("minimal-env", "test-project", envTypeId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "id", "minimal-env"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "project_id", "test-project"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "env_type_id", envTypeId),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "display_name", "minimal-env"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "uuid"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "created_at"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "updated_at"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "status", "active"),
				),
			},
		},
	})
}

func testAccEnvironmentResourceConfig(id, projectId, envTypeId, displayName string) string {
	return fmt.Sprintf(`
resource "platform-orchestrator_project" "test_project" {
  id           = %[2]q
  display_name = "Test Project for Environment"
}

resource "platform-orchestrator_environment_type" "test_env_type" {
  id           = %[3]q
  display_name = "Test Environment Type"
}


resource "platform-orchestrator_environment" "test" {
  id           = %[1]q
  project_id   = platform-orchestrator_project.test_project.id
  env_type_id  = platform-orchestrator_environment_type.test_env_type.id
  display_name = %[4]q

  timeouts {
    delete = "1m"
  }
}
`, id, projectId, envTypeId, displayName)
}

func testAccEnvironmentResourceMinimalConfig(id, projectId, envTypeId string) string {
	return fmt.Sprintf(`
resource "platform-orchestrator_project" "test_project" {
  id           = %[2]q
  display_name = "Test Project for Environment"
}

resource "platform-orchestrator_environment_type" "test_env_type" {
  id           = %[3]q
  display_name = "Test Environment Type"
}

resource "platform-orchestrator_environment" "test" {
  id          = %[1]q
  project_id  = platform-orchestrator_project.test_project.id
  env_type_id = platform-orchestrator_environment_type.test_env_type.id
}
`, id, projectId, envTypeId)
}
