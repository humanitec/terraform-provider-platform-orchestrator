package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSourceWithConfigure = &commonRunnerDataSource{}

type commonRunnerDataSource struct {
	SubType                  string
	SchemaDef                schema.Schema
	ReadApiResponseIntoModel func(canyoncp.Runner, commonRunnerModel) (commonRunnerModel, error)

	// params set during Configure()
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

type commonRunnerModel struct {
	Id                        types.String `tfsdk:"id"`
	Description               types.String `tfsdk:"description"`
	RunnerConfiguration       types.Object `tfsdk:"runner_configuration"`
	StateStorageConfiguration types.Object `tfsdk:"state_storage_configuration"`
}

func (d *commonRunnerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.SubType
}

var commonRunnerStateStorageDataSourceSchema = schema.SingleNestedAttribute{
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
}

func (d *commonRunnerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = d.SchemaDef
}

func (d *commonRunnerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *commonRunnerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data commonRunnerModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetRunnerWithResponse(ctx, d.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read %s, got error: %s", d.SubType, err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddError(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("%s with ID %s not found in org %s", d.SubType, data.Id.ValueString(), d.orgId))
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read %s, unexpected status code: %d, body: %s", d.SubType, httpResp.StatusCode(), httpResp.Body))
		return
	}

	runner := httpResp.JSON200

	data.Id = types.StringValue(runner.Id)
	data.Description = types.StringPointerValue(runner.Description)

	// Convert the runner to the data source model
	if convertedData, err := d.ReadApiResponseIntoModel(*runner, data); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to %s: %s", d.SubType, err))
		return
	} else {
		data.RunnerConfiguration = convertedData.RunnerConfiguration
		data.StateStorageConfiguration = convertedData.StateStorageConfiguration
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
