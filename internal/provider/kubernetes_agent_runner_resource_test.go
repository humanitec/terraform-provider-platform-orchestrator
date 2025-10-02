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

func TestAccKubernetesAgentRunnerResource(t *testing.T) {
	var runnerId = fmt.Sprint("runner-", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccKubernetesAgentRunnerResource(runnerId, `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAc5dgCx4ano39JT0XgTsHnts3jej+5xl7ZAwSIrKpef0=
-----END PUBLIC KEY-----`, "humanitec-runner"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_agent_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_agent_runner.test",
						tfjsonpath.New("description"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_agent_runner.test",
						tfjsonpath.New("runner_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"key": knownvalue.StringExact(`-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAc5dgCx4ano39JT0XgTsHnts3jej+5xl7ZAwSIrKpef0=
-----END PUBLIC KEY-----
`),
							"job": knownvalue.MapExact(map[string]knownvalue.Check{
								"namespace":       knownvalue.StringExact("default"),
								"service_account": knownvalue.StringExact("humanitec-runner"),
								"pod_template":    knownvalue.StringExact(`{"metadata":{"labels":{"app.kubernetes.io/name":"humanitec-runner"}}}`),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_agent_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("kubernetes"),
							"kubernetes_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"namespace": knownvalue.StringExact("humanitec-runner"),
							}),
							"s3": knownvalue.Null(),
						}),
					),
				},
			},
			// Update testing
			{
				Config: testAccKubernetesAgentRunnerResourceUpdateNoPodTemplate(runnerId, `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAc5dgCx4ano39JT0XgTsHnts3jej+5xl7ZAwSIrKpeg0=
-----END PUBLIC KEY-----`, "default"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_agent_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_agent_runner.test",
						tfjsonpath.New("runner_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"key": knownvalue.StringExact(`-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAc5dgCx4ano39JT0XgTsHnts3jej+5xl7ZAwSIrKpeg0=
-----END PUBLIC KEY-----
`),
							"job": knownvalue.MapExact(map[string]knownvalue.Check{
								"namespace":       knownvalue.StringExact("default"),
								"service_account": knownvalue.StringExact("humanitec-runner"),
								"pod_template":    knownvalue.StringExact("{}"),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_agent_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("kubernetes"),
							"kubernetes_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"namespace": knownvalue.StringExact("default"),
							}),
							"s3": knownvalue.Null(),
						}),
					),
				},
			},
			{
				ResourceName:      "platform-orchestrator_kubernetes_agent_runner.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccKubernetesAgentRunnerResource(id, key, stateNamespace string) string {
	return `
resource "platform-orchestrator_kubernetes_agent_runner" "test" {
  id = "` + id + `"
  runner_configuration = {
	key = <<EOT
` + key + `
EOT
	job = {
		namespace = "default"
		service_account = "humanitec-runner"
		pod_template = jsonencode({
			metadata = {
				labels = {
					"app.kubernetes.io/name" = "humanitec-runner"
				}
			}
		})
	}
  }
  state_storage_configuration = {
	type = "kubernetes"
	kubernetes_configuration = {
	  namespace = "` + stateNamespace + `"
    }
  }
}
`
}

func testAccKubernetesAgentRunnerResourceUpdateNoPodTemplate(id, key, stateNamespace string) string {
	return `
resource "platform-orchestrator_kubernetes_agent_runner" "test" {
  id = "` + id + `"
  runner_configuration = {
	key = <<EOT
` + key + `
EOT
	job = {
	  namespace = "default"
      service_account = "humanitec-runner"
	  pod_template = "{}"
	}
  }
  state_storage_configuration = {
	type = "kubernetes"
	kubernetes_configuration = {
	  namespace = "` + stateNamespace + `"
	}
  }
}
`
}

func TestAccKubernetesAgentRunnerResource_s3_state(t *testing.T) {
	var runnerId = fmt.Sprint("runner-", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
resource "platform-orchestrator_kubernetes_agent_runner" "test" {
  id = "` + runnerId + `"
  runner_configuration = {
	key = <<EOT
-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAc5dgCx4ano39JT0XgTsHnts3jej+5xl7ZAwSIrKpeg0=
-----END PUBLIC KEY-----
EOT
	job = {
	  namespace = "default"
      service_account = "humanitec-runner"
	  pod_template = "{}"
	}
  }
  state_storage_configuration = {
	type = "s3"
	s3_configuration = {
	  bucket = "some-bucket"
      path_prefix = "some/prefix"
	}
  }
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_agent_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_agent_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("s3"),
							"s3": knownvalue.MapExact(map[string]knownvalue.Check{
								"bucket":      knownvalue.StringExact("some-bucket"),
								"path_prefix": knownvalue.StringExact("some/prefix"),
							}),
							"kubernetes_configuration": knownvalue.Null(),
						}),
					),
				},
			},
			{
				ResourceName:      "platform-orchestrator_kubernetes_agent_runner.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
