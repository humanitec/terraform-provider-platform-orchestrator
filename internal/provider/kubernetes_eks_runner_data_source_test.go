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

func TestAccKubernetesEksRunnerDataSource(t *testing.T) {
	var runnerId = fmt.Sprint("runner-", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create runner resource and read via data source
			{
				Config: testAccKubernetesEksRunnerDataSourceConfig(runnerId),
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct runner
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test EKS Runner for data source"),
					),
					// Verify runner configuration is correctly read
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("name"),
						knownvalue.StringExact("eks-cluster-name"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("region"),
						knownvalue.StringExact("us-west-2"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("auth").AtMapKey("role_arn"),
						knownvalue.StringExact("arn:aws:iam::123456789012:role/EksRunnerRole"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("auth").AtMapKey("session_name"),
						knownvalue.StringExact("eks-runner-session"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("auth").AtMapKey("sts_region"),
						knownvalue.StringExact("us-west-2"),
					),
					// Verify job configuration
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("namespace"),
						knownvalue.StringExact("runner-namespace"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("service_account"),
						knownvalue.StringExact("eks-runner-sa"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("pod_template"),
						knownvalue.StringExact(`{"metadata":{"labels":{"app.kubernetes.io/name":"eks-runner-test"}}}`),
					),
					// Verify state storage configuration
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("state_storage_configuration").AtMapKey("type"),
						knownvalue.StringExact("kubernetes"),
					),
					statecheck.ExpectKnownValue(
						"data.platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("state_storage_configuration").AtMapKey("kubernetes_configuration").AtMapKey("namespace"),
						knownvalue.StringExact("state-namespace"),
					),
				},
			},
		},
	})
}

func testAccKubernetesEksRunnerDataSourceConfig(runnerId string) string {
	return `
resource "platform-orchestrator_kubernetes_eks_runner" "test" {
  id = "` + runnerId + `"
  description = "Test EKS Runner for data source"
  
  runner_configuration = {
    cluster = {
      name = "eks-cluster-name"
      region = "us-west-2"
      auth = {
        role_arn = "arn:aws:iam::123456789012:role/EksRunnerRole"
        session_name = "eks-runner-session"
        sts_region = "us-west-2"
      }
    }
    job = {
      namespace = "runner-namespace"
      service_account = "eks-runner-sa"
      pod_template = jsonencode({
        metadata = {
          labels = {
            "app.kubernetes.io/name" = "eks-runner-test"
          }
        }
      })
    }
  }
  
  state_storage_configuration = {
    type = "kubernetes"
    kubernetes_configuration = {
      namespace = "state-namespace"
    }
  }
}

data "platform-orchestrator_kubernetes_eks_runner" "test" {
  id = platform-orchestrator_kubernetes_eks_runner.test.id
}
`
}
