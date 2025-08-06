package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccModuleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccModuleResource("test-module", "s3://my-bucket/module.zip", "{}"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("test-module"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact("custom-type"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("module_source"),
						knownvalue.StringExact("s3://my-bucket/module.zip"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("module_inputs"),
						knownvalue.StringExact("{}"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("dependencies"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							"database": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":   knownvalue.StringExact("postgres"),
								"class":  knownvalue.StringExact("default"),
								"id":     knownvalue.Null(),
								"params": knownvalue.Null(),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test module description"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("provider_mapping"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							"aws": knownvalue.StringExact("aws.my-aws-account"),
						}),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("coprovisioned"),
						knownvalue.ListPartial(map[int]knownvalue.Check{
							0: knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":                         knownvalue.StringExact("metrics"),
								"class":                        knownvalue.StringExact("default"),
								"id":                           knownvalue.Null(),
								"params":                       knownvalue.StringExact(`{"level":"info"}`),
								"copy_dependents_from_current": knownvalue.Bool(false),
								"is_dependent_on_current":      knownvalue.Bool(true),
							}),
						}),
					),
				},
			},
			// Update testing
			{
				Config: testAccModuleResourceWithUpdate("test-module", "s3://my-bucket/module-v2.zip", "jsonencode({ region = \"us-east-1\" })", "Updated test module description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("test-module"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact("custom-type"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("module_source"),
						knownvalue.StringExact("s3://my-bucket/module-v2.zip"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("module_inputs"),
						knownvalue.StringExact(`{"region":"us-east-1"}`),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("dependencies"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							"database": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":   knownvalue.StringExact("custom-type"),
								"class":  knownvalue.StringExact("production"),
								"id":     knownvalue.StringExact("main-db"),
								"params": knownvalue.StringExact(`{"version":"14"}`),
							}),
							"cache": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":   knownvalue.StringExact("postgres"),
								"class":  knownvalue.StringExact("default"),
								"id":     knownvalue.Null(),
								"params": knownvalue.Null(),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated test module description"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("provider_mapping"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							"aws": knownvalue.StringExact("aws.my-updated-aws-account"),
						}),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("coprovisioned"),
						knownvalue.ListPartial(map[int]knownvalue.Check{
							0: knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":                         knownvalue.StringExact("metrics"),
								"class":                        knownvalue.StringExact("advanced"),
								"id":                           knownvalue.StringExact("mon-1"),
								"params":                       knownvalue.Null(),
								"copy_dependents_from_current": knownvalue.Bool(true),
								"is_dependent_on_current":      knownvalue.Bool(false),
							}),
						}),
					),
				},
			},
			{
				ResourceName:      "humanitec_module.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccModuleResourceWithSourceCode(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccModuleResourceWithSourceCode("test-module-code", `resource "aws_db_instance" "default" { engine = "postgres" }`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("test-module-code"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact("custom-type"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("module_source_code"),
						knownvalue.StringExact(`resource "aws_db_instance" "default" { engine = "postgres" }
`),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("coprovisioned"),
						knownvalue.ListSizeExact(0),
					),
				},
			},
			// Update testing
			{
				Config: testAccModuleResourceWithSourceCodeUpdate("test-module-code", `resource "aws_db_instance" "default" { engine = "mysql" }`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("test-module-code"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("resource_type"),
						knownvalue.StringExact("custom-type"),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("module_source_code"),
						knownvalue.StringExact(`resource "aws_db_instance" "default" { engine = "mysql" }
`),
					),
					statecheck.ExpectKnownValue(
						"humanitec_module.test",
						tfjsonpath.New("coprovisioned"),
						knownvalue.ListPartial(map[int]knownvalue.Check{
							0: knownvalue.ObjectExact(map[string]knownvalue.Check{
								"type":                         knownvalue.StringExact("postgres"),
								"class":                        knownvalue.StringExact("advanced"),
								"id":                           knownvalue.StringExact("postgres-1"),
								"params":                       knownvalue.StringExact(`{"interval":"5m"}`),
								"copy_dependents_from_current": knownvalue.Bool(true),
								"is_dependent_on_current":      knownvalue.Bool(false),
							}),
						}),
					),
				},
			},
		},
	})
}

func testAccModuleResource(id, moduleSource, moduleInputs string) string {
	return `
resource "humanitec_resource_type" "custom_type" {
  id           =  "custom-type"
  output_schema = "{}"
}

resource "humanitec_resource_type" "metrics" {
  id           =  "metrics"
  output_schema = "{}"
}

resource "humanitec_resource_type" "postgres" {
  id           =  "postgres"
  output_schema = "{}"
}

resource "humanitec_provider" "test_aws" {
  id = "my-aws-account"
  provider_type = "aws"
  source = "hashicorp/aws"
  version_constraint = ">= 4.0.0"
}

resource "humanitec_module" "test" {
  id             = "` + id + `"
  description    = "Test module description"
  resource_type  = humanitec_resource_type.custom_type.id
  module_source  = "` + moduleSource + `"
  module_inputs  = "` + moduleInputs + `"
  
  provider_mapping = {
    aws = "${humanitec_provider.test_aws.provider_type}.${humanitec_provider.test_aws.id}"
  }

  dependencies = {
    database = {
      type  = humanitec_resource_type.postgres.id
      class = "default"
    }
  }
  
  coprovisioned = [
    {
      type                         = humanitec_resource_type.metrics.id
      class                        = "default"
      params                       = jsonencode({"level": "info"})
      copy_dependents_from_current = false
      is_dependent_on_current      = true
    }
  ]
}
`
}

func testAccModuleResourceWithUpdate(id, moduleSource, moduleInputs, description string) string {
	return `
resource "humanitec_resource_type" "custom_type" {
  id           =  "custom-type"
  output_schema = "{}"
}

resource "humanitec_resource_type" "metrics" {
  id           =  "metrics"
  output_schema = "{}"
}

resource "humanitec_resource_type" "postgres" {
  id           =  "postgres"
  output_schema = "{}"
}

resource "humanitec_provider" "test_aws" {
  id = "my-aws-account"
  provider_type = "aws"
  source = "hashicorp/aws"
  version_constraint = ">= 4.0.0"
}

resource "humanitec_provider" "test_aws_updated" {
  id = "my-updated-aws-account"
  provider_type = "aws"
  source = "hashicorp/aws"
  version_constraint = ">= 4.0.0"
}

resource "humanitec_module" "test" {
  id             = "` + id + `"
  description    = "` + description + `"
  resource_type  = humanitec_resource_type.custom_type.id
  module_source  = "` + moduleSource + `"
  module_inputs  = ` + moduleInputs + `
  
  provider_mapping = {
    aws = "${humanitec_provider.test_aws_updated.provider_type}.${humanitec_provider.test_aws_updated.id}"
  }

  dependencies = {
    database = {
      type   = humanitec_resource_type.custom_type.id
      class  = "production"
      id     = "main-db"
      params = jsonencode({"version": "14"})
    }
    cache = {
      type  = humanitec_resource_type.postgres.id
      class = "default"
    }
  }
  
  coprovisioned = [
    {
      type                         = humanitec_resource_type.metrics.id
      class                        = "advanced"
      id                          = "mon-1"
      params                       = null
      copy_dependents_from_current = true
      is_dependent_on_current      = false
    }
  ]
}
`
}

func testAccModuleResourceWithSourceCode(id, sourceCode string) string {
	rs := `
resource "humanitec_resource_type" "custom_type" {
  id           =  "custom-type"
  output_schema = "{}"
}

resource "humanitec_module" "test" {
  id                 = "` + id + `"
  resource_type      = humanitec_resource_type.custom_type.id
  module_source_code =<<EOT
` + sourceCode + `
EOT
  
  coprovisioned = []
}
`
	return rs
}

func testAccModuleResourceWithSourceCodeUpdate(id, sourceCode string) string {
	rs := `
resource "humanitec_resource_type" "custom_type" {
  id           =  "custom-type"
  output_schema = "{}"
}

resource "humanitec_resource_type" "postgres" {
  id           =  "postgres"
  output_schema = "{}"
}

resource "humanitec_module" "test" {
  id                 = "` + id + `"
  resource_type      = humanitec_resource_type.custom_type.id
  module_source_code =<<EOT
` + sourceCode + `
EOT
  
  coprovisioned = [
    {
      type                         = humanitec_resource_type.postgres.id
      class                        = "advanced"
      id                          = "postgres-1"
      params                       = jsonencode({"interval": "5m"})
      copy_dependents_from_current = true
      is_dependent_on_current      = false
    }
  ]
}
`
	return rs
}

// prepareProvidersAndResourceTypes sets up the necessary providers and resource types for the tests.
// func prepareProvidersAndResourceTypes(t *testing.T, canyonCpClient canyoncp.ClientWithResponsesInterface, orgId string) {
// 	t.Helper()

// 	resp, err := canyonCpClient.CreateModuleProviderWithResponse(t.Context(), orgId, canyoncp.CreateModuleProviderJSONRequestBody{
// 		Id:                "my-aws-account",
// 		Source:            "hashicorp/aws",
// 		ProviderType:      "aws",
// 		VersionConstraint: "~> 3.0",
// 	})
// 	require.NoError(t, err, "Failed to create module provider `my-aws-account`")
// 	require.Contains(t, []int{http.StatusCreated, http.StatusConflict}, resp.StatusCode(), "Unexpected status code when creating module provider: %v", string(resp.Body))

// 	resp, err = canyonCpClient.CreateModuleProviderWithResponse(t.Context(), orgId, canyoncp.CreateModuleProviderJSONRequestBody{
// 		Id:                "my-updated-aws-account",
// 		Source:            "hashicorp/aws",
// 		ProviderType:      "aws",
// 		VersionConstraint: "~> 3.0",
// 	})
// 	require.NoError(t, err, "Failed to create module provider `my-updated-aws-account`")
// 	require.Contains(t, []int{http.StatusCreated, http.StatusConflict}, resp.StatusCode(), "Unexpected status code when creating module provider: %v", string(resp.Body))

// 	respType, err := canyonCpClient.CreateResourceTypeWithResponse(t.Context(), orgId, canyoncp.CreateResourceTypeJSONRequestBody{
// 		Id:           "custom-type",
// 		OutputSchema: map[string]interface{}{},
// 	})
// 	require.NoError(t, err, "Failed to create resource type `custom-type`")
// 	require.Contains(t, []int{http.StatusCreated, http.StatusConflict}, respType.StatusCode(), "Unexpected status code when creating resource type")

// 	respType, err = canyonCpClient.CreateResourceTypeWithResponse(t.Context(), orgId, canyoncp.CreateResourceTypeJSONRequestBody{
// 		Id:           "metrics",
// 		OutputSchema: map[string]interface{}{},
// 	})
// 	require.NoError(t, err, "Failed to create resource type `metrics`")
// 	require.Contains(t, []int{http.StatusCreated, http.StatusConflict}, respType.StatusCode(), "Unexpected status code when creating resource type")

// 	respType, err = canyonCpClient.CreateResourceTypeWithResponse(t.Context(), orgId, canyoncp.CreateResourceTypeJSONRequestBody{
// 		Id:           "postgres",
// 		OutputSchema: map[string]interface{}{},
// 	})
// 	require.NoError(t, err, "Failed to create resource type `environment`")
// 	require.Contains(t, []int{http.StatusCreated, http.StatusConflict}, respType.StatusCode(), "Unexpected status code when creating resource type")
// }

// // destroyProvidersAndResourceTypes cleans up the providers and resource types created during the tests.
// func destroyProvidersAndResourceTypes(t *testing.T, canyonCpClient canyoncp.ClientWithResponsesInterface, orgId string) {
// 	t.Helper()

// 	providerResp, err := canyonCpClient.DeleteModuleProviderWithResponse(t.Context(), orgId, "aws", "my-aws-account")
// 	require.NoError(t, err, "Failed to delete module provider `my-aws-account`")
// 	require.Equal(t, http.StatusNoContent, providerResp.StatusCode(), "Unexpected status code when deleting module provider: %v - %s", providerResp.StatusCode(), string(providerResp.Body))

// 	providerResp, err = canyonCpClient.DeleteModuleProviderWithResponse(t.Context(), orgId, "aws", "my-updated-aws-account")
// 	require.NoError(t, err, "Failed to delete module provider `my-updated-aws-account`")
// 	require.Equal(t, http.StatusNoContent, providerResp.StatusCode(), "Unexpected status code when deleting module provider: %v - %s", providerResp.StatusCode(), string(providerResp.Body))

// 	typeResp, err := canyonCpClient.DeleteResourceTypeWithResponse(t.Context(), orgId, "custom-type")
// 	require.NoError(t, err, "Failed to delete resource type `custom-type`")
// 	require.Equal(t, http.StatusNoContent, typeResp.StatusCode(), "Unexpected status code when deleting resource type: %v - %s", typeResp.StatusCode(), string(typeResp.Body))

// 	typeResp, err = canyonCpClient.DeleteResourceTypeWithResponse(t.Context(), orgId, "metrics")
// 	require.NoError(t, err, "Failed to delete resource type `metrics`")
// 	require.Equal(t, http.StatusNoContent, typeResp.StatusCode(), "Unexpected status code when deleting resource type: %v - %s", typeResp.StatusCode(), string(typeResp.Body))
// }
