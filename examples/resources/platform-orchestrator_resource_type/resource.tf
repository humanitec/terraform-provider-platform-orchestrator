resource "platform-orchestrator_resource_type" "resource_type" {
  id          = "my_resource"
  description = "This is a sample resource type."
  output_schema = jsonencode({
    type = "object"
    properties = {
      example_property = {
        type        = "string"
        description = "An example property in the output schema."
      }
    }
  })
  is_developer_accessible = true
}
