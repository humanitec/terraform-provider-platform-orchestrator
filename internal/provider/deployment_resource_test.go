package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const deploymentScenario = `
resource "random_id" "r" {
  byte_length = 4
}

resource "platform-orchestrator_project" "project" {
  id           = "project-${random_id.r.hex}"
}

resource "platform-orchestrator_kubernetes_agent_runner" "runner" {
  id = "runner-${random_id.r.hex}"
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

resource "platform-orchestrator_runner_rule" "rrule" {
  runner_id   = platform-orchestrator_kubernetes_agent_runner.runner.id
  project_id = platform-orchestrator_project.project.id
}

resource "platform-orchestrator_environment_type" "env_type" {
  id           = "env-type-${random_id.r.hex}"
}

resource "platform-orchestrator_environment" "env" {
  id           = "env-${random_id.r.hex}"
  project_id   = platform-orchestrator_project.project.id
  env_type_id  = platform-orchestrator_environment_type.env_type.id
  depends_on   = [platform-orchestrator_runner_rule.rrule]
}

resource "platform-orchestrator_deployment" "deployment" {
  project_id   = platform-orchestrator_project.project.id
  env_type_id  = platform-orchestrator_environment.env.id
  manifest = jsonencode({
    workloads = {
      main = {
        variables = {
          ANIMAL = "cat"
        }
      }
    }
  })
}
`

func TestAccDeploymentResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:   deploymentScenario,
				PlanOnly: true,
			},
			{
				Config: deploymentScenario,
			},
		},
	})
}
