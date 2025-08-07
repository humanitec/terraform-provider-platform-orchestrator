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

func TestAccKubernetesRunnerDataSource(t *testing.T) {
	var runnerId = fmt.Sprint("runner-", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create runner resource and read via data source
			{
				Config: testAccKubernetesRunnerDataSourceConfig(runnerId),
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct runner
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test Kubernetes Runner for data source"),
					),
					// Verify cluster configuration is correctly read
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("cluster_data").AtMapKey("certificate_authority_data"),
						knownvalue.StringExact("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t..."),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("cluster_data").AtMapKey("server"),
						knownvalue.StringExact("https://kubernetes.example.com:6443"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("cluster_data").AtMapKey("proxy_url"),
						knownvalue.StringExact("http://proxy.example.com:8080"),
					),
					// Verify auth configuration (service account token)
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("auth").AtMapKey("service_account_token"),
						knownvalue.StringExact("SECRET"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("auth").AtMapKey("client_certificate_data"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("auth").AtMapKey("client_key_data"),
						knownvalue.Null(),
					),
					// Verify job configuration
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("namespace"),
						knownvalue.StringExact("k8s-runner-namespace"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("service_account"),
						knownvalue.StringExact("k8s-runner-sa"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("pod_template"),
						knownvalue.StringExact(`{"metadata":{"labels":{"app.kubernetes.io/name":"k8s-runner-test","runner-type":"kubernetes"}},"spec":{"containers":[{"image":"humanitec/runner:latest","name":"runner"}]}}`),
					),
					// Verify state storage configuration
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("state_storage_configuration").AtMapKey("type"),
						knownvalue.StringExact("kubernetes"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_runner.test",
						tfjsonpath.New("state_storage_configuration").AtMapKey("kubernetes_configuration").AtMapKey("namespace"),
						knownvalue.StringExact("k8s-state-namespace"),
					),
				},
			},
		},
	})
}

func testAccKubernetesRunnerDataSourceConfig(runnerId string) string {
	return `
resource "humanitec_kubernetes_runner" "test" {
  id = "` + runnerId + `"
  description = "Test Kubernetes Runner for data source"
  
  runner_configuration = {
    cluster = {
      cluster_data = {
        certificate_authority_data = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t..."
        server = "https://kubernetes.example.com:6443"
        proxy_url = "http://proxy.example.com:8080"
      }
      auth = {
        service_account_token = "eyJhbGciOiJSUzI1NiIsImtpZCI6Ii..."
      }
    }
    job = {
      namespace = "k8s-runner-namespace"
      service_account = "k8s-runner-sa"
      pod_template = jsonencode({
        metadata = {
          labels = {
            "app.kubernetes.io/name" = "k8s-runner-test"
            "runner-type" = "kubernetes"
          }
        }
        spec = {
          containers = [
            {
              name = "runner"
              image = "humanitec/runner:latest"
            }
          ]
        }
      })
    }
  }
  
  state_storage_configuration = {
    type = "kubernetes"
    kubernetes_configuration = {
      namespace = "k8s-state-namespace"
    }
  }
}

data "humanitec_kubernetes_runner" "test" {
  id = humanitec_kubernetes_runner.test.id
}
`
}
