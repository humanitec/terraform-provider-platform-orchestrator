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

func NewKubernetesGkeRunnerDataSource() datasource.DataSource {
	return &commonRunnerDataSource{
		SubType: "kubernetes_gke_runner",
		SchemaDef: schema.Schema{
			// This description is used by the documentation generator and the language server.
			MarkdownDescription: "Kubernetes GKE Runner data source",

			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "Kubernetes GKE Runner ID",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
							"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
						),
					},
				},
				"description": schema.StringAttribute{
					MarkdownDescription: "Kubernetes GKE Runner description",
					Computed:            true,
				},
				"runner_configuration": schema.SingleNestedAttribute{
					MarkdownDescription: "The configuration of the Kubernetes GKE cluster",
					Computed:            true,
					Attributes: map[string]schema.Attribute{
						"cluster": schema.SingleNestedAttribute{
							MarkdownDescription: "The cluster configuration for the Kubernetes GKE Runner",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									MarkdownDescription: "The name of the Kubernetes GKE cluster",
									Computed:            true,
								},
								"project_id": schema.StringAttribute{
									MarkdownDescription: "The project ID where the GKE cluster is located",
									Computed:            true,
								},
								"location": schema.StringAttribute{
									MarkdownDescription: "The location of the GKE cluster",
									Computed:            true,
								},
								"proxy_url": schema.StringAttribute{
									MarkdownDescription: "The proxy URL for the Kubernetes GKE cluster",
									Computed:            true,
								},
								"internal_ip": schema.BoolAttribute{
									MarkdownDescription: "Whether to use internal IP for the Kubernetes GKE cluster",
									Computed:            true,
								},
								"auth": schema.SingleNestedAttribute{
									MarkdownDescription: "The authentication configuration for the Kubernetes GKE cluster",
									Computed:            true,
									Sensitive:           true,
									Attributes: map[string]schema.Attribute{
										"gcp_audience": schema.StringAttribute{
											MarkdownDescription: "The GCP audience to authenticate to the GKE cluster",
											Computed:            true,
										},
										"gcp_service_account": schema.StringAttribute{
											MarkdownDescription: "The GCP service account to authenticate to the GKE cluster",
											Computed:            true,
										},
									},
								},
							},
						},
						"job": schema.SingleNestedAttribute{
							MarkdownDescription: "The job configuration for the Kubernetes GKE Runner",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"namespace": schema.StringAttribute{
									MarkdownDescription: "The namespace for the Kubernetes GKE Runner job",
									Computed:            true,
								},
								"service_account": schema.StringAttribute{
									MarkdownDescription: "The service account for the Kubernetes GKE Runner job",
									Computed:            true,
								},
								"pod_template": schema.StringAttribute{
									MarkdownDescription: "JSON encoded pod template for the Kubernetes GKE Runner job",
									Computed:            true,
									CustomType:          jsontypes.NormalizedType{},
								},
							},
						},
					},
				},
				"state_storage_configuration": schema.SingleNestedAttribute{
					MarkdownDescription: "The state storage configuration for the Kubernetes GKE Runner",
					Computed:            true,
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of state storage configuration for the Kubernetes GKE Runner",
							Computed:            true,
						},
						"kubernetes_configuration": schema.SingleNestedAttribute{
							MarkdownDescription: "The Kubernetes state storage configuration for the Kubernetes GKE Runner",
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
		ReadApiResponseIntoModel: func(runner canyoncp.Runner, model commonRunnerModel) (commonRunnerModel, error) {
			x, err := toKubernetesGkeRunnerResourceModel(runner)
			return commonRunnerModel(x), err
		},
	}
}
