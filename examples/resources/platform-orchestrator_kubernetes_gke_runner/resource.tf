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
        gcp_audience        = "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/humanitec-runner-pool/providers/humanitec-runner"
        gcp_service_account = "humanitec-runner@my-account.iam.gserviceaccount.com"
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
  state_storage_configuration = {
    type = "kubernetes"
    kubernetes_configuration = {
      namespace = "humanitec"
    }
  }
}
