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
}

resource "humanitec_module_rule" "minio" {
  module_id      = humanitec_module.minio.id
  resource_class = "custom-class"
  env_type_id    = "development"
}
