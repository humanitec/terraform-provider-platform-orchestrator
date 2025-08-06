package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccKubernetesGkeRunnerDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create runner resource and read via data source
			{
				Config: testAccKubernetesGkeRunnerDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct runner
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("tf-provider-gke-data-test"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test GKE Runner for data source"),
					),
					// Verify runner configuration is correctly read
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("name"),
						knownvalue.StringExact("gke-cluster-name"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("project_id"),
						knownvalue.StringExact("gke-project-id"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("location"),
						knownvalue.StringExact("gke-location"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("internal_ip"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("auth").AtMapKey("gcp_audience"),
						knownvalue.StringExact("https://gke.googleapis.com/"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("cluster").AtMapKey("auth").AtMapKey("gcp_service_account"),
						knownvalue.StringExact("account@example.com"),
					),
					// Verify job configuration
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("namespace"),
						knownvalue.StringExact("runner-namespace"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("service_account"),
						knownvalue.StringExact("gke-runner-sa"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job").AtMapKey("pod_template"),
						knownvalue.StringExact(`{"metadata":{"labels":{"app.kubernetes.io/name":"gke-runner-test"}}}`),
					),
					// Verify state storage configuration
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("state_storage_configuration").AtMapKey("type"),
						knownvalue.StringExact("kubernetes"),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("state_storage_configuration").AtMapKey("kubernetes_configuration").AtMapKey("namespace"),
						knownvalue.StringExact("state-namespace"),
					),
				},
			},
		},
	})
}

const testAccKubernetesGkeRunnerDataSourceConfig = `
resource "humanitec_kubernetes_gke_runner" "test" {
  id = "tf-provider-gke-data-test"
  description = "Test GKE Runner for data source"
  
  runner_configuration = {
    cluster = {
      name = "gke-cluster-name"
      project_id = "gke-project-id"
      location = "gke-location"
      internal_ip = true
      auth = {
        gcp_audience = "https://gke.googleapis.com/"
        gcp_service_account = "account@example.com"
      }
    }
    job = {
      namespace = "runner-namespace"
      service_account = "gke-runner-sa"
      pod_template = jsonencode({
        metadata = {
          labels = {
            "app.kubernetes.io/name" = "gke-runner-test"
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

data "humanitec_kubernetes_gke_runner" "test" {
  id = humanitec_kubernetes_gke_runner.test.id
}
`