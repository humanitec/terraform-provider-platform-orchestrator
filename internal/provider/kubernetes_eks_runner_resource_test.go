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
							"s3_configuration":  knownvalue.Null(),
							"gcs_configuration": knownvalue.Null(),
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
							"s3_configuration":  knownvalue.Null(),
							"gcs_configuration": knownvalue.Null(),
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

func TestAccKubernetesEksRunnerResource_gcs_state(t *testing.T) {
	var runnerId = fmt.Sprint("runner-", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with GCS state storage
			{
				Config: testAccKubernetesEksRunnerResourceGCSState(runnerId, "my-bucket", "initial/prefix"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("gcs"),
							"gcs_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"bucket":      knownvalue.StringExact("my-bucket"),
								"path_prefix": knownvalue.StringExact("initial/prefix"),
							}),
							"kubernetes_configuration": knownvalue.Null(),
							"s3_configuration":         knownvalue.Null(),
						}),
					),
				},
			},
			// Update prefix
			{
				Config: testAccKubernetesEksRunnerResourceGCSState(runnerId, "my-bucket", "updated/prefix"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("gcs"),
							"gcs_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"bucket":      knownvalue.StringExact("my-bucket"),
								"path_prefix": knownvalue.StringExact("updated/prefix"),
							}),
							"kubernetes_configuration": knownvalue.Null(),
							"s3_configuration":         knownvalue.Null(),
						}),
					),
				},
			},
			// Remove prefix
			{
				Config: testAccKubernetesEksRunnerResourceGCSStateNoPrefix(runnerId, "my-bucket"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("gcs"),
							"gcs_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"bucket":      knownvalue.StringExact("my-bucket"),
								"path_prefix": knownvalue.Null(),
							}),
							"kubernetes_configuration": knownvalue.Null(),
							"s3_configuration":         knownvalue.Null(),
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

func TestAccKubernetesEksRunnerResource_s3_state(t *testing.T) {
	var runnerId = fmt.Sprint("runner-", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with S3 state storage
			{
				Config: testAccKubernetesEksRunnerResourceS3State(runnerId, "my-bucket", `path_prefix = "initial/prefix"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("s3"),
							"s3_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"bucket":      knownvalue.StringExact("my-bucket"),
								"path_prefix": knownvalue.StringExact("initial/prefix"),
							}),
							"kubernetes_configuration": knownvalue.Null(),
							"gcs_configuration":        knownvalue.Null(),
						}),
					),
				},
			},
			// Update prefix
			{
				Config: testAccKubernetesEksRunnerResourceS3State(runnerId, "my-bucket", `path_prefix = "updated/prefix"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("s3"),
							"s3_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"bucket":      knownvalue.StringExact("my-bucket"),
								"path_prefix": knownvalue.StringExact("updated/prefix"),
							}),
							"kubernetes_configuration": knownvalue.Null(),
							"gcs_configuration":        knownvalue.Null(),
						}),
					),
				},
			},
			// Remove prefix
			{
				Config: testAccKubernetesEksRunnerResourceS3State(runnerId, "my-bucket", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_kubernetes_eks_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("s3"),
							"s3_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"bucket":      knownvalue.StringExact("my-bucket"),
								"path_prefix": knownvalue.Null(),
							}),
							"kubernetes_configuration": knownvalue.Null(),
							"gcs_configuration":        knownvalue.Null(),
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

func testAccKubernetesEksRunnerResourceS3State(id, bucket, pathPrefixAttr string) string {
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
    }
  }
  state_storage_configuration = {
    type = "s3"
    s3_configuration = {
      bucket = "` + bucket + `"
      ` + pathPrefixAttr + `
    }
  }
}
`
}

func testAccKubernetesEksRunnerResourceGCSState(id, bucket, pathPrefix string) string {
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
    }
  }
  state_storage_configuration = {
    type = "gcs"
    gcs_configuration = {
      bucket = "` + bucket + `"
      path_prefix = "` + pathPrefix + `"
    }
  }
}
`
}

func testAccKubernetesEksRunnerResourceGCSStateNoPrefix(id, bucket string) string {
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
    }
  }
  state_storage_configuration = {
    type = "gcs"
    gcs_configuration = {
      bucket = "` + bucket + `"
    }
  }
}
`
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
