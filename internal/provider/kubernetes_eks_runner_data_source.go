package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &KubernetesEksRunnerDataSource{}

func NewKubernetesEksRunnerDataSource() datasource.DataSource {
	return &KubernetesEksRunnerDataSource{baseRunnerDataSource: baseRunnerDataSource{
		readApiResponseIntoModel: toKubernetesEksRunnerResourceModel,
	}}
}

// KubernetesEksRunnerDataSource defines the data source implementation.
type KubernetesEksRunnerDataSource struct {
	baseRunnerDataSource
}

func (d *KubernetesEksRunnerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_eks_runner"
}

func (d *KubernetesEksRunnerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Kubernetes EKS Runner data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Kubernetes EKS Runner ID",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
						"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
					),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Kubernetes EKS Runner.",
				Computed:            true,
			},
			"runner_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration of the Kubernetes EKS cluster.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"cluster": schema.SingleNestedAttribute{
						MarkdownDescription: "The cluster configuration for the Kubernetes EKS Runner.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								MarkdownDescription: "The name of the Kubernetes EKS cluster.",
								Computed:            true,
							},
							"region": schema.StringAttribute{
								MarkdownDescription: "The AWS region where the EKS cluster is located.",
								Computed:            true,
							},
							"auth": schema.SingleNestedAttribute{
								MarkdownDescription: "Configuration to obtain temporary AWS security credentials by assuming an IAM role.",
								Computed:            true,
								Attributes: map[string]schema.Attribute{
									"role_arn": schema.StringAttribute{
										MarkdownDescription: "The ARN of the role to assume.",
										Computed:            true,
									},
									"session_name": schema.StringAttribute{
										MarkdownDescription: "Session name to be used when assuming the role. If not provided, a default session name will be \"{org_id}-{runner_id}\"",
										Computed:            true,
									},
									"sts_region": schema.StringAttribute{
										MarkdownDescription: "The AWS region identifier for the Security Token Service (STS) endpoint. If not provided, the cluster region will be used.",
										Computed:            true,
									},
								},
							},
						},
					},
					"job": schema.SingleNestedAttribute{
						MarkdownDescription: "The job configuration for the Kubernetes EKS Runner.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"namespace": schema.StringAttribute{
								MarkdownDescription: "The namespace for the Kubernetes EKS Runner job.",
								Computed:            true,
							},
							"service_account": schema.StringAttribute{
								MarkdownDescription: "The service account for the Kubernetes EKS Runner job.",
								Computed:            true,
							},
							"pod_template": schema.StringAttribute{
								MarkdownDescription: "JSON encoded pod template for the Kubernetes EKS Runner job.",
								Computed:            true,
								CustomType:          jsontypes.NormalizedType{},
							},
						},
					},
				},
			},
			"state_storage_configuration": RunnerStateStorageDataSourceSchema,
		},
	}
}
