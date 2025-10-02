package provider

import (
	"regexp"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func NewKubernetesAgentRunnerDataSource() datasource.DataSource {
	return &commonRunnerDataSource{
		SubType: "kubernetes_agent_runner",
		SchemaDef: schema.Schema{
			// This description is used by the documentation generator and the language server.
			MarkdownDescription: "Kubernetes Agent Runner data source",

			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "Kubernetes Agent Runner ID",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
							"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
						),
					},
				},
				"description": schema.StringAttribute{
					MarkdownDescription: "Kubernetes Agent Runner description",
					Computed:            true,
				},
				"runner_configuration": schema.SingleNestedAttribute{
					MarkdownDescription: "The configuration of the Kubernetes Agent Runner",
					Computed:            true,
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							MarkdownDescription: "The public ed25519 key in PEM format used to identify the caller identity",
							Computed:            true,
						},
						"job": schema.SingleNestedAttribute{
							MarkdownDescription: "The job configuration for the Kubernetes Job triggered by the Kubernetes Agent Runner",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"namespace": schema.StringAttribute{
									MarkdownDescription: "The namespace for the Kubernetes Runner job",
									Computed:            true,
								},
								"service_account": schema.StringAttribute{
									MarkdownDescription: "The service account for the Kubernetes Runner job",
									Computed:            true,
								},
								"pod_template": schema.StringAttribute{
									MarkdownDescription: "JSON encoded pod template for the Kubernetes Runner job",
									Computed:            true,
									CustomType:          jsontypes.NormalizedType{},
								},
							},
						},
					},
				},
				"state_storage_configuration": schema.SingleNestedAttribute{
					MarkdownDescription: "The state storage configuration for the Kubernetes Agent Runner",
					Computed:            true,
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of state storage configuration for the Kubernetes Agent Runner",
							Computed:            true,
						},
						"kubernetes_configuration": schema.SingleNestedAttribute{
							MarkdownDescription: "The Kubernetes state storage configuration for the Kubernetes Agent Runner",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"namespace": schema.StringAttribute{
									MarkdownDescription: "The namespace for the Kubernetes state storage configuration",
									Computed:            true,
								},
							},
						},
					},
				},
			},
		},
		ReadApiResponseIntoModel: func(item canyoncp.Runner, model commonRunnerModel) (commonRunnerModel, error) {
			x, err := toKubernetesAgentRunnerResourceModel(item)
			return commonRunnerModel(x), err
		},
	}
}
