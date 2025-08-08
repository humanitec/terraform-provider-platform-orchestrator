package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccRunnerRuleResourceBasic(t *testing.T) {
	var runnerId = fmt.Sprintf("test-runner-%d", time.Now().UnixNano())
	var envTypeId = fmt.Sprintf("test-env-type-%d", time.Now().UnixNano())
	var ruleId uuid.UUID
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - minimal configuration
			{
				Config: testAccRunnerRuleResourceBasic(runnerId),
				Check: func(s *terraform.State) error {
					ruleId = uuid.Must(uuid.Parse(s.RootModule().Resources["humanitec_runner_rule.test"].Primary.ID))
					return nil
				},
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
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"humanitec_runner_rule.test",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(""),
					),
				},
			},
			// Update any fields means re-creating the resource
			{
				Config: testAccRunnerRuleResourceWithEnvType(runnerId, envTypeId),
				Check: func(s *terraform.State) error {
					newRuleId := s.RootModule().Resources["humanitec_runner_rule.test"].Primary.ID
					if newRuleId == ruleId.String() {
						return fmt.Errorf("expected new rule ID after update, got same ID: %s", newRuleId)
					}
					return nil
				},
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
						knownvalue.StringExact(""),
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

func testAccRunnerRuleResourceBasic(runnerId string) string {
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
