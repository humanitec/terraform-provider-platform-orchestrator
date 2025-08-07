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

func TestAccKubernetesGkeRunnerResource(t *testing.T) {
	var runnerId = fmt.Sprint("runner-", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccKubernetesGkeRunnerResource(runnerId, "humanitec-runner", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("runner_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"cluster": knownvalue.MapExact(map[string]knownvalue.Check{
								"name":        knownvalue.StringExact("gke-cluster-name"),
								"project_id":  knownvalue.StringExact("gke-project-id"),
								"location":    knownvalue.StringExact("gke-location"),
								"internal_ip": knownvalue.Bool(false),
								"proxy_url":   knownvalue.Null(),
								"auth": knownvalue.MapExact(map[string]knownvalue.Check{
									"gcp_audience":        knownvalue.StringExact("https://gke.googleapis.com/"),
									"gcp_service_account": knownvalue.StringExact("account@example.com"),
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
						"humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("kubernetes"),
							"kubernetes_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"namespace": knownvalue.StringExact("humanitec-runner"),
							}),
						}),
					),
				},
			},
			// Update testing
			{
				Config: testAccKubernetesGkeRunnerResource(runnerId, "default", `pod_template = null`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("runner_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"cluster": knownvalue.MapExact(map[string]knownvalue.Check{
								"name":        knownvalue.StringExact("gke-cluster-name"),
								"project_id":  knownvalue.StringExact("gke-project-id"),
								"location":    knownvalue.StringExact("gke-location"),
								"internal_ip": knownvalue.Bool(false),
								"proxy_url":   knownvalue.Null(),
								"auth": knownvalue.MapExact(map[string]knownvalue.Check{
									"gcp_audience":        knownvalue.StringExact("https://gke.googleapis.com/"),
									"gcp_service_account": knownvalue.StringExact("account@example.com"),
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
						"humanitec_kubernetes_gke_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("kubernetes"),
							"kubernetes_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"namespace": knownvalue.StringExact("default"),
							}),
						}),
					),
				},
			},
			{
				ResourceName:      "humanitec_kubernetes_gke_runner.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccKubernetesGkeRunnerResource(id, stateNamespace, podTemplate string) string {
	return `
resource "humanitec_kubernetes_gke_runner" "test" {
  id = "` + id + `"
  runner_configuration = {
    cluster = {
      name = "gke-cluster-name"
	  project_id = "gke-project-id"
	  location = "gke-location"
      auth = {
		gcp_audience = "https://gke.googleapis.com/"
		gcp_service_account = "account@example.com"
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
