resource "platform-orchestrator_serverless_ecs_runner" "example" {
  id          = "my-ecs-runner"
  description = "This is a sample ECS runner configuration."

  runner_configuration = {
    auth = {
      role_arn = "arn:aws:iam::123456789012:role/humanitec_role"
    }
    job = {
      region             = "eu-central-1"
      cluster            = "my-ecs-cluster-name"
      execution_role_arn = "arn:aws:iam::123456789012:role/execution_role"
      subnets            = ["my-subnet-1"]

      task_role_arn        = "arn:aws:iam::123456789012:role/task_role"
      is_public_ip_enabled = false
      security_groups      = []

      environment = {
        "EXAMPLE_ENVIRONMENT_VARIABLE" = "value"
      }

      secrets = {
        "SECRET_ENVIRONMENT_VARIABLE"   = "arn:aws:secretsmanager:eu-central-1:123456789012:secret:myapp/api-key-XyZ9Qw"
        "PROPERTY_ENVIRONMENT_VARIABLE" = "arn:aws:ssm:eu-central-1:123456789012:parameter/app/config/api-endpoint"
      }
    }
  }

  state_storage_configuration = {
    type = "s3"
    s3_configuration = {
      bucket      = "humanitec-ecs-runner-state"
      path_prefix = "state-files"
    }
  }
}
