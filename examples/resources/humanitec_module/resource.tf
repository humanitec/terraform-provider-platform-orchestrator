resource "humanitec_module" "minio" {
  id            = "my-minio"
  description   = "Module for a minio bucket"
  resource_type = "minio"
  module_source = "git::https://github.com/humanitec/module-definition-library//minio?ref=preview"
  provider_mapping = {
    minio = "minio.default"
  }
  module_inputs = jsonencode({
    provider_region = "my-minio-bucket"
    bucket_prefix   = "bucket-$${context.res_id}-"
  })
  dependencies = {
    postgres = {
      type   = "postgres"
      class  = "classic"
      id     = "standard.common-postgres"
      params = {}
    }
  }
  coprovisioned = [
    {
      type                         = "custom-type"
      class                        = "classic"
      id                           = "standard.custom-resource"
      params                       = {}
      is_dependent_on_current      = false
      copy_dependents_from_current = false
    }
  ]

}
