resource "humanitec_provider" "aws" {
  id                 = "my-aws-provider"
  description        = "Provider using default runner environment variables for AWS"
  provider_type      = "aws"
  source             = "hashicorp/aws"
  version_constraint = ">= 3.0.0"
  configuration = jsonencode({
    region = "us-west-2"
  })
}
