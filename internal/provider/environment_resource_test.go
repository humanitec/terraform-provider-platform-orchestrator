package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentResourceConfig("test-env", "test-project", "development", "Test Environment"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "id", "test-env"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "project_id", "test-project"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "env_type_id", "development"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "display_name", "Test Environment"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "uuid"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "created_at"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "updated_at"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "status", "active"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "runner_id"),
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
				Config: testAccEnvironmentResourceConfig("test-env", "test-project", "development", "Updated Test Environment"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "id", "test-env"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "project_id", "test-project"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "env_type_id", "development"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "display_name", "Updated Test Environment"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "uuid"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "created_at"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "updated_at"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "status", "active"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "runner_id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccEnvironmentResourceMinimalConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with minimal configuration
			{
				Config: testAccEnvironmentResourceMinimalConfig("minimal-env", "test-project", "development"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "id", "minimal-env"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "project_id", "test-project"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "env_type_id", "development"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "display_name", "minimal-env"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "uuid"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "created_at"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "updated_at"),
					resource.TestCheckResourceAttr("platform-orchestrator_environment.test", "status", "active"),
					resource.TestCheckResourceAttrSet("platform-orchestrator_environment.test", "runner_id"),
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

resource "platform-orchestrator_kubernetes_agent_runner" "test_runner" {
  id = "test-runner-env-resource"
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

resource "platform-orchestrator_kubernetes_agent_runner" "test_runner_minimal" {
  id = "test-runner-env-minimal"
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

resource "platform-orchestrator_runner_rule" "test_runner_rule_minimal" {
  runner_id   = platform-orchestrator_kubernetes_agent_runner.test_runner_minimal.id
  env_type_id = platform-orchestrator_environment_type.test_env_type.id
}

resource "platform-orchestrator_environment" "test" {
  id          = %[1]q
  project_id  = platform-orchestrator_project.test_project.id
  env_type_id = platform-orchestrator_environment_type.test_env_type.id
  depends_on  = [platform-orchestrator_runner_rule.test_runner_rule_minimal]
}
`, id, projectId, envTypeId)
}
