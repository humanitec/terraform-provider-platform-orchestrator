resource "platform-orchestrator_kubernetes_gke_runner" "my_runner" {
  id          = "my-runner"
  description = "runner for all the envs"
  runner_configuration = {
    cluster = {
      name        = "my-cluster"
      project_id  = "my-gcp-project"
      location    = "europe-west3"
      internal_ip = false
      auth = {
        gcp_audience        = "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/platform-orchestrator-runner-pool/providers/platform-orchestrator-runner"
        gcp_service_account = "platform-orchestrator-runner@my-account.iam.gserviceaccount.com"
      }
    }
    job = {
      namespace       = "default"
      service_account = "platform-orchestrator-runner"
      pod_template = jsonencode({
        metadata = {
          labels = {
            "app.kubernetes.io/name" = "platform-orchestrator-runner"
          }
        }
      })
    }
  }
  state_storage_configuration = {
    type = "kubernetes"
    kubernetes_configuration = {
      namespace = "platform-orchestrator"
    }
  }
}
