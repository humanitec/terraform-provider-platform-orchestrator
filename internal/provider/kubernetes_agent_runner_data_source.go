package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &KubernetesAgentRunnerDataSource{}

func NewKubernetesAgentRunnerDataSource() datasource.DataSource {
	return &KubernetesAgentRunnerDataSource{}
}

// KubernetesAgentRunnerDataSource defines the data source implementation.
type KubernetesAgentRunnerDataSource struct {
	baseRunnerDataSource
}

func (d *KubernetesAgentRunnerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_agent_runner"
}

func (d *KubernetesAgentRunnerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
			"state_storage_configuration": RunnerStateStorageDataSourceSchema,
		},
	}
}

func (d *KubernetesAgentRunnerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RunnerResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetRunnerWithResponse(ctx, d.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read kubernetes agent runner, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddError(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Kubernetes agent runner with ID %s not found in org %s", data.Id.ValueString(), d.orgId))
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read kubernetes agent runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	runner := httpResp.JSON200

	data.Id = types.StringValue(runner.Id)
	data.Description = types.StringPointerValue(runner.Description)

	// Convert the runner to the data source model
	if convertedData, err := toKubernetesAgentRunnerResourceModel(*runner); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesAgentRunnerDataSourceModel: %s", err))
		return
	} else {
		data.RunnerConfiguration = convertedData.RunnerConfiguration
		data.StateStorageConfiguration = convertedData.StateStorageConfiguration
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
