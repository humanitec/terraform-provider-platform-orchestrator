resource "platform-orchestrator_kubernetes_eks_runner" "my_runner" {
  id          = "my-runner"
  description = "runner for all the envs"
  runner_configuration = {
    cluster = {
      name   = "my-eks-cluster"
      region = "us-west-2"
      auth = {
        role_arn     = "arn:aws:iam::123456789012:role/EksRunnerRole"
        session_name = "humanitec-runner-session"
        sts_region   = "us-west-2"
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
