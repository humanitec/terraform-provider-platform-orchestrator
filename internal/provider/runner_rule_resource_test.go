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

func TestAccRunnerRuleResourceBasic(t *testing.T) {
	var (
		runnerId  = fmt.Sprintf("test-runner-%d", time.Now().UnixNano())
		envTypeId = fmt.Sprintf("test-env-type-%d", time.Now().UnixNano())
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - minimal configuration
			{
				Config: testAccRunnerRuleResourceBasic(runnerId, envTypeId),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_runner_rule.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"humanitec_runner_rule.test",
						tfjsonpath.New("runner_id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"humanitec_runner_rule.test",
						tfjsonpath.New("env_type_id"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"humanitec_runner_rule.test",
						tfjsonpath.New("project_id"),
						knownvalue.Null(),
					),
				},
			},
			{
				ResourceName:      "humanitec_runner_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccRunnerRuleResourceWithEnvType(t *testing.T) {
	var (
		runnerId  = fmt.Sprintf("test-runner-%d", time.Now().UnixNano())
		envTypeId = fmt.Sprintf("test-env-type-%d", time.Now().UnixNano())
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - with environment type
			{
				Config: testAccRunnerRuleResourceWithEnvType(runnerId, envTypeId),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_runner_rule.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"humanitec_runner_rule.test",
						tfjsonpath.New("runner_id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"humanitec_runner_rule.test",
						tfjsonpath.New("env_type_id"),
						knownvalue.StringExact(envTypeId),
					),
					statecheck.ExpectKnownValue(
						"humanitec_runner_rule.test",
						tfjsonpath.New("project_id"),
						knownvalue.Null(),
					),
				},
			},
			{
				ResourceName:      "humanitec_runner_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccRunnerRuleResourceBasic(runnerId, envTypeId string) string {
	return `
resource "humanitec_environment_type" "test" {
  id = "` + envTypeId + `"
}

resource "humanitec_kubernetes_agent_runner" "test" {
  id = "` + runnerId + `"
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

resource "humanitec_runner_rule" "test" {
  runner_id = humanitec_kubernetes_agent_runner.test.id
}
`
}

func testAccRunnerRuleResourceWithEnvType(runnerId, envTypeId string) string {
	return `
resource "humanitec_environment_type" "test" {
  id = "` + envTypeId + `"
}

resource "humanitec_kubernetes_agent_runner" "test" {
  id = "` + runnerId + `"
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

resource "humanitec_runner_rule" "test" {
  runner_id   = humanitec_kubernetes_agent_runner.test.id
  env_type_id = humanitec_environment_type.test.id
}
`
}
