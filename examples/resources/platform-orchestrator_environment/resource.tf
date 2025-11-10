resource "platform-orchestrator_environment" "example" {
  id           = "my-environment"
  project_id   = "my-project"
  env_type_id  = "development"
  display_name = "My Development Environment"
  delete_rules = true
}
