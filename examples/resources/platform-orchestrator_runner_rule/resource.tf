resource "platform-orchestrator_runner_rule" "my_runner_my_project_development" {
  runner_id   = "my-runner"
  project_id  = "my-project"
  env_type_id = "development"
}
