resource "platform-orchestrator_kubernetes_agent_runner" "my_runner" {
  id          = "my-runner"
  description = "runner for all the envs"
  runner_configuration = {
    key = <<EOT
-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAc5dgCx4ano39JT0XgTsHnts3jej+5xl7ZAwSIrKpef0=
-----END PUBLIC KEY-----
EOT
    job = {
      namespace       = "default"
      service_account = "humanitec-runner"
      pod_template = jsonencode({
        metadata = {
          labels = {
            "app.kubernetes.io/name" = "humanitec-runner"
          }
        }
      })
    }
  }
  state_storage_configuration = {
    type = "kubernetes"
    kubernetes_configuration = {
      namespace = "humanitec"
    }
  }
}
