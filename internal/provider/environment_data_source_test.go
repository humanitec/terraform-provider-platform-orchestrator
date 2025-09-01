package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccEnvironmentDataSourceConfig("test-env-data", "test-project-data", "development", "Test Environment Data"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.platform-orchestrator_environment.test", "id", "test-env-data"),
					resource.TestCheckResourceAttr("data.platform-orchestrator_environment.test", "project_id", "test-project-data"),
					resource.TestCheckResourceAttr("data.platform-orchestrator_environment.test", "env_type_id", "development"),
					resource.TestCheckResourceAttr("data.platform-orchestrator_environment.test", "display_name", "Test Environment Data"),
					resource.TestCheckResourceAttrSet("data.platform-orchestrator_environment.test", "uuid"),
					resource.TestCheckResourceAttrSet("data.platform-orchestrator_environment.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.platform-orchestrator_environment.test", "updated_at"),
					resource.TestCheckResourceAttr("data.platform-orchestrator_environment.test", "status", "active"),
					resource.TestCheckResourceAttrSet("data.platform-orchestrator_environment.test", "runner_id"),
				),
			},
		},
	})
}

func testAccEnvironmentDataSourceConfig(id, projectId, envTypeId, displayName string) string {
	return fmt.Sprintf(`
resource "platform-orchestrator_project" "test_project" {
  id           = %[2]q
  display_name = "Test Project for Environment Data"
}

resource "platform-orchestrator_environment_type" "test_env_type" {
  id           = %[3]q
  display_name = "Test Environment Type"
}

resource "platform-orchestrator_kubernetes_agent_runner" "test_runner" {
  id = "test-runner-env-data"
  runner_configuration = {
    key = <<EOT
-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAc5dgCx4ano39JT0XgTsHnts3jej+5xl7ZAwSIrKpef0=
-----END PUBLIC KEY-----
EOT
    job = {
      namespace = "default"
      service_account = "humanitec-runner"
    }
  }
  state_storage_configuration = {
    type = "kubernetes"
    kubernetes_configuration = {
      namespace = "humanitec-runner"
    }
  }
}

resource "platform-orchestrator_runner_rule" "test_runner_rule" {
  runner_id   = platform-orchestrator_kubernetes_agent_runner.test_runner.id
  env_type_id = platform-orchestrator_environment_type.test_env_type.id
}

resource "platform-orchestrator_environment" "test" {
  id           = %[1]q
  project_id   = platform-orchestrator_project.test_project.id
  env_type_id  = platform-orchestrator_environment_type.test_env_type.id
  display_name = %[4]q
  depends_on   = [platform-orchestrator_runner_rule.test_runner_rule]
}

data "platform-orchestrator_environment" "test" {
  id         = platform-orchestrator_environment.test.id
  project_id = platform-orchestrator_environment.test.project_id
}
`, id, projectId, envTypeId, displayName)
}
