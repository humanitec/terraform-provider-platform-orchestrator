package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func NewKubernetesRunnerDataSource() datasource.DataSource {
	return &commonRunnerDataSource{
		SubType: "kubernetes_runner",
		SchemaDef: schema.Schema{
			// This description is used by the documentation generator and the language server.
			MarkdownDescription: "Kubernetes Runner data source",

			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "Kubernetes Runner ID",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
							"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
						),
					},
				},
				"description": schema.StringAttribute{
					MarkdownDescription: "Kubernetes Runner description",
					Computed:            true,
				},
				"runner_configuration": schema.SingleNestedAttribute{
					MarkdownDescription: "The configuration of the Kubernetes Runner cluster",
					Computed:            true,
					Attributes: map[string]schema.Attribute{
						"cluster": schema.SingleNestedAttribute{
							MarkdownDescription: "The cluster configuration for the Kubernetes Runner cluster",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"cluster_data": schema.SingleNestedAttribute{
									MarkdownDescription: "The cluster data for the Kubernetes Runner cluster",
									Computed:            true,
									Attributes: map[string]schema.Attribute{
										"certificate_authority_data": schema.StringAttribute{
											MarkdownDescription: "The certificate authority data for the Kubernetes Runner cluster",
											Computed:            true,
										},
										"server": schema.StringAttribute{
											MarkdownDescription: "The server URL for the Kubernetes Runner cluster",
											Computed:            true,
										},
										"proxy_url": schema.StringAttribute{
											MarkdownDescription: "The proxy URL for the Kubernetes Runner cluster",
											Computed:            true,
										},
									},
								},
								"auth": schema.SingleNestedAttribute{
									MarkdownDescription: "The authentication configuration for the Kubernetes Runner cluster",
									Computed:            true,
									Sensitive:           true,
									Attributes: map[string]schema.Attribute{
										"client_certificate_data": schema.StringAttribute{
											MarkdownDescription: "The client certificate data for the Kubernetes Runner cluster",
											Computed:            true,
										},
										"client_key_data": schema.StringAttribute{
											MarkdownDescription: "The client key data for the Kubernetes Runner cluster",
											Computed:            true,
										},
										"service_account_token": schema.StringAttribute{
											MarkdownDescription: "The service account token for the Kubernetes Runner cluster",
											Computed:            true,
										},
									},
								},
							},
						},
						"job": schema.SingleNestedAttribute{
							MarkdownDescription: "The job configuration for the Kubernetes Runner",
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
				"state_storage_configuration": commonRunnerStateStorageDataSourceSchema,
			},
		},
		ReadApiResponseIntoModel: toKubernetesRunnerResourceModel,
	}
}
