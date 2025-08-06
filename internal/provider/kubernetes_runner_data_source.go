package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &KubernetesRunnerDataSource{}

func NewKubernetesRunnerDataSource() datasource.DataSource {
	return &KubernetesRunnerDataSource{}
}

// KubernetesRunnerDataSource defines the data source implementation.
type KubernetesRunnerDataSource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// KubernetesRunnerDataSourceModel describes the data source data model.
type KubernetesRunnerDataSourceModel struct {
	Id                        types.String `tfsdk:"id"`
	Description               types.String `tfsdk:"description"`
	RunnerConfiguration       types.Object `tfsdk:"runner_configuration"`
	StateStorageConfiguration types.Object `tfsdk:"state_storage_configuration"`
}

func (d *KubernetesRunnerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_runner"
}

func (d *KubernetesRunnerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
							},
						},
					},
				},
			},
			"state_storage_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The state storage configuration for the Kubernetes Runner",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "The type of state storage configuration for the Kubernetes Runner",
						Computed:            true,
					},
					"kubernetes_configuration": schema.SingleNestedAttribute{
						MarkdownDescription: "The Kubernetes state storage configuration for the Kubernetes Runner",
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
	}
}

func (d *KubernetesRunnerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *KubernetesRunnerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KubernetesRunnerDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetRunnerWithResponse(ctx, d.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read kubernetes runner, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read kubernetes runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	runner := httpResp.JSON200

	data.Id = types.StringValue(runner.Id)
	data.Description = types.StringPointerValue(runner.Description)

	// For the kubernetes_runner data source, we need to use a more generic approach
	// since it could be any type of runner configuration. We'll create a temporary
	// KubernetesRunnerResourceModel to extract the data.
	dummyData := KubernetesRunnerResourceModel{
		Id:          types.StringValue(runner.Id),
		Description: types.StringPointerValue(runner.Description),
	}

	// Convert the runner using the existing resource model conversion
	if convertedData, err := toKubernetesRunnerResourceModel(*runner, dummyData); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesRunnerDataSourceModel: %s", err))
		return
	} else {
		data.RunnerConfiguration = convertedData.RunnerConfiguration
		data.StateStorageConfiguration = convertedData.StateStorageConfiguration
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}