package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccKubernetesAgentRunnerDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create runner resource and read via data source
			{
				Config: testAccKubernetesAgentRunnerDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct runner
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_agent_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("tf-provider-agent-data-test"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_agent_runner.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test Agent Runner for data source"),
					),
					// Verify runner configuration is correctly read
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_agent_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("key"),
						knownvalue.StringExact(`-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAc5dgCx4ano39JT0XgTsHnts3jej+5xl7ZAwSIrKpef0=
-----END PUBLIC KEY-----
`),
					),
					// Verify job configuration
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_agent_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("namespace"),
						knownvalue.StringExact("agent-namespace"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_agent_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("service_account"),
						knownvalue.StringExact("agent-runner-sa"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_agent_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("pod_template"),
						knownvalue.StringExact(`{"metadata":{"labels":{"app.kubernetes.io/name":"agent-runner-test","environment":"test"}}}`),
					),
					// Verify state storage configuration
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_agent_runner.test",
						tfjsonpath.New("state_storage_configuration").AtMapKey("type"),
						knownvalue.StringExact("kubernetes"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_agent_runner.test",
						tfjsonpath.New("state_storage_configuration").AtMapKey("kubernetes_configuration").AtMapKey("namespace"),
						knownvalue.StringExact("agent-state-namespace"),
					),
				},
			},
		},
	})
}

const testAccKubernetesAgentRunnerDataSourceConfig = `
resource "humanitec_kubernetes_agent_runner" "test" {
  id = "tf-provider-agent-data-test"
  description = "Test Agent Runner for data source"
  
  runner_configuration = {
    key = <<EOT
-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAc5dgCx4ano39JT0XgTsHnts3jej+5xl7ZAwSIrKpef0=
-----END PUBLIC KEY-----
EOT
    job = {
      namespace = "agent-namespace"
      service_account = "agent-runner-sa"
      pod_template = jsonencode({
        metadata = {
          labels = {
            "app.kubernetes.io/name" = "agent-runner-test"
            "environment" = "test"
          }
        }
      })
    }
  }
  
  state_storage_configuration = {
    type = "kubernetes"
    kubernetes_configuration = {
      namespace = "agent-state-namespace"
    }
  }
}

data "humanitec_kubernetes_agent_runner" "test" {
  id = humanitec_kubernetes_agent_runner.test.id
}
`