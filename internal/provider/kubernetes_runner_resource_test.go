package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccKubernetesRunnerResource(t *testing.T) {
	var runnerId = fmt.Sprint("runner-", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccKubernetesRunnerResource(runnerId, KubernetesRunnerClusterAuth{
					ClientCertificateData: types.StringValue("client-certificate-data"),
				}, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_kubernetes_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"humanitec_kubernetes_runner.test",
						tfjsonpath.New("runner_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"cluster": knownvalue.MapExact(map[string]knownvalue.Check{
								"cluster_data": knownvalue.MapExact(map[string]knownvalue.Check{
									"certificate_authority_data": knownvalue.StringExact("certificate-authority-data"),
									"server":                     knownvalue.StringExact("10.0.1:6443"),
									"proxy_url":                  knownvalue.Null(),
								}),
								"auth": knownvalue.MapExact(map[string]knownvalue.Check{
									"client_certificate_data": knownvalue.StringExact("client-certificate-data"),
									"client_key_data":         knownvalue.Null(),
									"service_account_token":   knownvalue.Null(),
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
						"humanitec_kubernetes_runner.test",
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
				Config: testAccKubernetesRunnerResource(runnerId, KubernetesRunnerClusterAuth{
					ServiceAccountToken: types.StringValue("service-account-token"),
				}, `pod_template = jsonencode({
	metadata = {
		labels = {
			"app.kubernetes.io/name" = "humanitec-runner"
		}
	}	
})`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_kubernetes_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"humanitec_kubernetes_runner.test",
						tfjsonpath.New("runner_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"cluster": knownvalue.MapExact(map[string]knownvalue.Check{
								"cluster_data": knownvalue.MapExact(map[string]knownvalue.Check{
									"certificate_authority_data": knownvalue.StringExact("certificate-authority-data"),
									"server":                     knownvalue.StringExact("10.0.1:6443"),
									"proxy_url":                  knownvalue.Null(),
								}),
								"auth": knownvalue.MapExact(map[string]knownvalue.Check{
									"client_certificate_data": knownvalue.Null(),
									"client_key_data":         knownvalue.Null(),
									"service_account_token":   knownvalue.StringExact("service-account-token"),
								}),
							}),
							"job": knownvalue.MapExact(map[string]knownvalue.Check{
								"namespace":       knownvalue.StringExact("default"),
								"service_account": knownvalue.StringExact("humanitec-runner"),
								"pod_template":    knownvalue.StringExact(`{"metadata":{"labels":{"app.kubernetes.io/name":"humanitec-runner"}}}`),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"humanitec_kubernetes_runner.test",
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
			{
				ResourceName:      "humanitec_kubernetes_runner.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"runner_configuration.cluster.auth.client_certificate_data",
					"runner_configuration.cluster.auth.client_key_data",
					"runner_configuration.cluster.auth.service_account_token",
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccKubernetesRunnerResource(id string, auth KubernetesRunnerClusterAuth, podTemplate string) string {
	var authString string
	if auth.ClientCertificateData.ValueString() != "" {
		authString = `
	  client_certificate_data = "` + auth.ClientCertificateData.ValueString() + `"`
	} else {
		authString = `
	  service_account_token = "` + auth.ServiceAccountToken.ValueString() + `"`
	}

	return `
resource "humanitec_kubernetes_runner" "test" {
  id = "` + id + `"
  runner_configuration = {
    cluster = {
      cluster_data = {
        certificate_authority_data = "certificate-authority-data"
        server = "10.0.1:6443"
      }
      auth = {
` + authString + `
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
	  namespace = "humanitec-runner"
    }
  }
}
`
}
