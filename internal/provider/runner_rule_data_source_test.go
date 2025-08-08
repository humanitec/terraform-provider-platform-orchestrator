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

func TestAccRunnerRuleDataSource(t *testing.T) {
	var (
		runnerId = fmt.Sprintf("test-runner-%d", time.Now().UnixNano())
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create runner rule resource and read via data source - basic configuration
			{
				Config: testAccRunnerRuleDataSourceBasic(runnerId),
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct runner rule
					statecheck.ExpectKnownValue(
						"data.humanitec_runner_rule.test",
						tfjsonpath.New("runner_id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_runner_rule.test",
						tfjsonpath.New("env_type_id"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_runner_rule.test",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(""),
					),
				},
			},
		},
	})
}

func TestAccRunnerRuleDataSourceWithEnvType(t *testing.T) {
	var (
		runnerId  = fmt.Sprintf("test-runner-%d", time.Now().UnixNano())
		envTypeId = fmt.Sprintf("test-env-type-%d", time.Now().UnixNano())
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create runner rule resource with env_type_id and read via data source
			{
				Config: testAccRunnerRuleDataSourceWithEnvType(runnerId, envTypeId),
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify the data source reads the correct runner rule
					statecheck.ExpectKnownValue(
						"data.humanitec_runner_rule.test",
						tfjsonpath.New("runner_id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_runner_rule.test",
						tfjsonpath.New("env_type_id"),
						knownvalue.StringExact(envTypeId),
					),
					statecheck.ExpectKnownValue(
						"data.humanitec_runner_rule.test",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(""),
					),
				},
			},
		},
	})
}

func testAccRunnerRuleDataSourceBasic(runnerId string) string {
	return `
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

data "humanitec_runner_rule" "test" {
  id = humanitec_runner_rule.test.id
}
`
}

func testAccRunnerRuleDataSourceWithEnvType(runnerId, envTypeId string) string {
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

data "humanitec_runner_rule" "test" {
  id = humanitec_runner_rule.test.id
}
`
}
