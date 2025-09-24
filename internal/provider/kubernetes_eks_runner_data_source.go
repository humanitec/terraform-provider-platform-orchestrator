package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &KubernetesEksRunnerDataSource{}

func NewKubernetesEksRunnerDataSource() datasource.DataSource {
	return &KubernetesEksRunnerDataSource{}
}

// KubernetesEksRunnerDataSource defines the data source implementation.
type KubernetesEksRunnerDataSource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// KubernetesEksRunnerDataSourceModel describes the data source data model.
type KubernetesEksRunnerDataSourceModel struct {
	Id                        types.String `tfsdk:"id"`
	Description               types.String `tfsdk:"description"`
	RunnerConfiguration       types.Object `tfsdk:"runner_configuration"`
	StateStorageConfiguration types.Object `tfsdk:"state_storage_configuration"`
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
			"state_storage_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The state storage configuration for the Kubernetes EKS Runner.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "The type of state storage configuration for the Kubernetes EKS Runner.",
						Computed:            true,
					},
					"kubernetes_configuration": schema.SingleNestedAttribute{
						MarkdownDescription: "The Kubernetes state storage configuration for the Kubernetes EKS Runner.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"namespace": schema.StringAttribute{
								MarkdownDescription: "The namespace for the Kubernetes state storage configuration.",
								Computed:            true,
							},
						},
					},
				},
			},
		},
	}
}

func (d *KubernetesEksRunnerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*HumanitecProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			HUM_PROVIDER_ERR,
			fmt.Sprintf("Expected *HumanitecProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.cpClient = providerData.CpClient
	d.orgId = providerData.OrgId
}

func (d *KubernetesEksRunnerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KubernetesEksRunnerDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetRunnerWithResponse(ctx, d.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read kubernetes eks runner, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddError(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Kubernetes eks runner with ID %s not found in org %s", data.Id.ValueString(), d.orgId))
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read kubernetes eks runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	runner := httpResp.JSON200

	data.Id = types.StringValue(runner.Id)
	data.Description = types.StringPointerValue(runner.Description)

	// Convert the runner to the data source model
	if convertedData, err := toKubernetesEksRunnerResourceModel(*runner); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesEksRunnerDataSourceModel: %s", err))
		return
	} else {
		data.RunnerConfiguration = convertedData.RunnerConfiguration
		data.StateStorageConfiguration = convertedData.StateStorageConfiguration
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
