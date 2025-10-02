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

func TestAccKubernetesEksRunnerResource(t *testing.T) {
	var runnerId = fmt.Sprint("runner-", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccKubernetesEksRunnerResource(runnerId, "humanitec-runner", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("runner_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"cluster": knownvalue.MapExact(map[string]knownvalue.Check{
								"name":   knownvalue.StringExact("eks-cluster-name"),
								"region": knownvalue.StringExact("us-west-2"),
								"auth": knownvalue.MapExact(map[string]knownvalue.Check{
									"role_arn":     knownvalue.StringExact("arn:aws:iam::123456789012:role/EksRunnerRole"),
									"session_name": knownvalue.Null(),
									"sts_region":   knownvalue.Null(),
								}),
							}),
							"job": knownvalue.MapExact(map[string]knownvalue.Check{
								"namespace":       knownvalue.StringExact("default"),
								"service_account": knownvalue.StringExact("humanitec-runner"),
								"pod_template":    knownvalue.Null(),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
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
				Config: testAccKubernetesEksRunnerResource(runnerId, "default", `pod_template = null`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("runner_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"cluster": knownvalue.MapExact(map[string]knownvalue.Check{
								"name":   knownvalue.StringExact("eks-cluster-name"),
								"region": knownvalue.StringExact("us-west-2"),
								"auth": knownvalue.MapExact(map[string]knownvalue.Check{
									"role_arn":     knownvalue.StringExact("arn:aws:iam::123456789012:role/EksRunnerRole"),
									"session_name": knownvalue.Null(),
									"sts_region":   knownvalue.Null(),
								}),
							}),
							"job": knownvalue.MapExact(map[string]knownvalue.Check{
								"namespace":       knownvalue.StringExact("default"),
								"service_account": knownvalue.StringExact("humanitec-runner"),
								"pod_template":    knownvalue.Null(),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
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
				ResourceName:      "platform-orchestrator_kubernetes_eks_runner.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccKubernetesEksRunnerResource(id, stateNamespace, podTemplate string) string {
	return `
resource "platform-orchestrator_kubernetes_eks_runner" "test" {
  id = "` + id + `"
  runner_configuration = {
    cluster = {
      name = "eks-cluster-name"
      region = "us-west-2"
      auth = {
        role_arn = "arn:aws:iam::123456789012:role/EksRunnerRole"
      }
    }
    job = {
      namespace = "default"
      service_account = "humanitec-runner"
      ` + podTemplate + `
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
