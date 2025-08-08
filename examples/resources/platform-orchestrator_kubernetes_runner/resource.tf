resource "platform-orchestrator_kubernetes_runner" "my_runner" {
  id          = "my_runner"
  description = "Development Kubernetes Runner"
  runner_configuration = {
    cluster = {
      cluster_data = {
        certificate_authority_data = "certificate-authority-data"
        server                     = "https://kubernetes.example.com"
      }
      auth = {
        service_account_token = "service-account-token"
      }
    }
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
  state_storage_configuration {
    type = "kubernetes"
    kubernetes_configuration {
      namespace = "humanitec"
    }
  }
}
