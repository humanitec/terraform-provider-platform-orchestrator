package provider

import (
	"context"
	"fmt"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &EnvironmentDataSource{}

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

// EnvironmentDataSource defines the data source implementation.
type EnvironmentDataSource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// EnvironmentDataSourceModel describes the data source data model.
type EnvironmentDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	ProjectId     types.String `tfsdk:"project_id"`
	EnvTypeId     types.String `tfsdk:"env_type_id"`
	DisplayName   types.String `tfsdk:"display_name"`
	Uuid          types.String `tfsdk:"uuid"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
	Status        types.String `tfsdk:"status"`
	StatusMessage types.String `tfsdk:"status_message"`
	RunnerId      types.String `tfsdk:"runner_id"`
}

func (d *EnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *EnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Environment data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the Environment.",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project this environment belongs to.",
				Required:            true,
			},
			"env_type_id": schema.StringAttribute{
				MarkdownDescription: "The environment type for the environment.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the Environment.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the Environment.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The date and time when the environment was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The date and time when the environment was updated.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the environment (active, deleting, delete_failed).",
				Computed:            true,
			},
			"status_message": schema.StringAttribute{
				MarkdownDescription: "An optional message associated with the status.",
				Computed:            true,
			},
			"runner_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the runner to be used to deploy this environment.",
				Computed:            true,
			},
		},
	}
}

func (d *EnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetEnvironmentWithResponse(ctx, d.orgId, data.ProjectId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == 404 {
		resp.Diagnostics.AddError(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Environment with ID %s not found in project %s", data.Id.ValueString(), data.ProjectId.ValueString()))
		return
	}

	if httpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read environment, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	// Convert API response to data source model
	environment := *httpResp.JSON200
	displayName := types.StringValue(environment.Id)
	if environment.DisplayName != "" {
		displayName = types.StringValue(environment.DisplayName)
	}

	statusMessage := types.StringNull()
	if environment.StatusMessage != nil && *environment.StatusMessage != "" {
		statusMessage = types.StringValue(*environment.StatusMessage)
	}

	data = EnvironmentDataSourceModel{
		Id:            types.StringValue(environment.Id),
		ProjectId:     types.StringValue(environment.ProjectId),
		EnvTypeId:     types.StringValue(environment.EnvTypeId),
		DisplayName:   displayName,
		Uuid:          types.StringValue(environment.Uuid.String()),
		CreatedAt:     types.StringValue(environment.CreatedAt.String()),
		UpdatedAt:     types.StringValue(environment.UpdatedAt.String()),
		Status:        types.StringValue(string(environment.Status)),
		StatusMessage: statusMessage,
		RunnerId:      types.StringValue(environment.RunnerId),
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
