resource "platform-orchestrator_deployment" "main" {
  project_id = "my-project"
  env_id     = "development"
  mode       = "deploy"
  manifest = jsonencode({
    workloads = {}
    shared    = {}
  })
  wait_for = true
}
