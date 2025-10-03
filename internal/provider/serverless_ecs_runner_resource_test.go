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

func TestAccEcsRunnerResourceCreateUpdateDelete(t *testing.T) {
	var runnerId = fmt.Sprintf("test-runner-%d", time.Now().UnixNano())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - minimal configuration
			{
				Config: `
resource "platform-orchestrator_serverless_ecs_runner" "test" {
  id          = "` + runnerId + `"
  runner_configuration = {
    auth = {
      role_arn = "arn:aws:iam::123456789012:role/humanitec_role"
    }
    job = {
      region             = "eu-central-1"
      cluster            = "my-ecs-cluster-name"
      execution_role_arn = "arn:aws:iam::123456789012:role/execution_role"
      subnets            = ["my-subnet-1"]
    }
  }
  state_storage_configuration = {
    type        = "s3"
    s3_configuration = {
	  bucket      = "humanitec-ecs-runner-state"
    }
  }
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_serverless_ecs_runner.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(runnerId),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_serverless_ecs_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("auth"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"role_arn":     knownvalue.StringExact("arn:aws:iam::123456789012:role/humanitec_role"),
							"session_name": knownvalue.Null(),
							"sts_region":   knownvalue.Null(),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_serverless_ecs_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							"region":               knownvalue.StringExact("eu-central-1"),
							"cluster":              knownvalue.StringExact("my-ecs-cluster-name"),
							"execution_role_arn":   knownvalue.StringExact("arn:aws:iam::123456789012:role/execution_role"),
							"subnets":              knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("my-subnet-1")}),
							"security_groups":      knownvalue.ListExact([]knownvalue.Check{}),
							"task_role_arn":        knownvalue.Null(),
							"is_public_ip_enabled": knownvalue.Bool(false),
							"environment":          knownvalue.MapExact(map[string]knownvalue.Check{}),
							"secrets":              knownvalue.MapExact(map[string]knownvalue.Check{}),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_serverless_ecs_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("s3"),
							"s3_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"bucket":      knownvalue.StringExact("humanitec-ecs-runner-state"),
								"path_prefix": knownvalue.Null(),
							}),
							"kubernetes_configuration": knownvalue.Null(),
						}),
					),
				},
			},
			{
				Config: `
resource "platform-orchestrator_serverless_ecs_runner" "test" {
  id          = "` + runnerId + `"
  runner_configuration = {
    auth = {
      role_arn = "arn:aws:iam::123456789012:role/humanitec_role"
      session_name = "ecs-runner-session"
      sts_region = "eu-central-1"
    }
    job = {
      region             = "eu-central-1"
      cluster            = "my-ecs-cluster-name"
      execution_role_arn = "arn:aws:iam::123456789012:role/execution_role"
      subnets            = ["my-subnet-1"]
      security_groups    = ["my-security-group"]
      task_role_arn      = "arn:aws:iam::123456789012:role/task_role"
      is_public_ip_enabled = true
      environment = {
        "MY_ENV_VAR" = "my-env-var-value"
      }
      secrets = {
        "MY_SECRET" = "arn:aws:secretsmanager:eu-west-1:123456789012:secret:myapp/api-key-XyZ9Qw"
      }
    }
  }
  state_storage_configuration = {
    type        = "s3"
    s3_configuration = {
	  bucket      = "humanitec-ecs-runner-state"
      path_prefix = "prefix"
    }
  }
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"platform-orchestrator_serverless_ecs_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("auth"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"role_arn":     knownvalue.StringExact("arn:aws:iam::123456789012:role/humanitec_role"),
							"session_name": knownvalue.StringExact("ecs-runner-session"),
							"sts_region":   knownvalue.StringExact("eu-central-1"),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_serverless_ecs_runner.test",
						tfjsonpath.New("runner_configuration").AtMapKey("job"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"region":               knownvalue.StringExact("eu-central-1"),
							"cluster":              knownvalue.StringExact("my-ecs-cluster-name"),
							"execution_role_arn":   knownvalue.StringExact("arn:aws:iam::123456789012:role/execution_role"),
							"subnets":              knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("my-subnet-1")}),
							"security_groups":      knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("my-security-group")}),
							"task_role_arn":        knownvalue.StringExact("arn:aws:iam::123456789012:role/task_role"),
							"is_public_ip_enabled": knownvalue.Bool(true),
							"environment": knownvalue.MapExact(map[string]knownvalue.Check{
								"MY_ENV_VAR": knownvalue.StringExact("my-env-var-value"),
							}),
							"secrets": knownvalue.MapExact(map[string]knownvalue.Check{
								"MY_SECRET": knownvalue.StringExact("arn:aws:secretsmanager:eu-west-1:123456789012:secret:myapp/api-key-XyZ9Qw"),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"platform-orchestrator_serverless_ecs_runner.test",
						tfjsonpath.New("state_storage_configuration"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"type": knownvalue.StringExact("s3"),
							"s3_configuration": knownvalue.MapExact(map[string]knownvalue.Check{
								"bucket":      knownvalue.StringExact("humanitec-ecs-runner-state"),
								"path_prefix": knownvalue.StringExact("prefix"),
							}),
							"kubernetes_configuration": knownvalue.Null(),
						}),
					),
				},
			},
			{
				ResourceName:      "platform-orchestrator_serverless_ecs_runner.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
