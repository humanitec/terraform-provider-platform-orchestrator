package provider

import (
	"context"
	"fmt"
	"net/http"
	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RunnerRuleDataSource{}

func NewRunnerRuleDataSource() datasource.DataSource {
	return &RunnerRuleDataSource{}
}

// RunnerRuleDataSource defines the data source implementation.
type RunnerRuleDataSource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// RunnerRuleDataSourceModel describes the data source data model.
type RunnerRuleDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	RunnerId  types.String `tfsdk:"runner_id"`
	ProjectId types.String `tfsdk:"project_id"`
	EnvTypeId types.String `tfsdk:"env_type_id"`
}

func (d *RunnerRuleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_rule"
}

func (d *RunnerRuleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Runner Rule data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the Runner Rule.",
				Required:            true,
			},
			"runner_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Runner this rule applies to.",
				Computed:            true,
			},
			"env_type_id": schema.StringAttribute{
				MarkdownDescription: "The environment type to match this rule.",
				Computed:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The project id that this rule matches.",
				Computed:            true,
			},
		},
	}
}

func (d *RunnerRuleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RunnerRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RunnerRuleDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetRunnerRuleInOrgWithResponse(ctx, d.orgId, uuid.MustParse(data.Id.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read runner rule, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read runner rule, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	runnerRule := httpResp.JSON200

	// Convert the runner rule to the data source model using the existing helper function
	runnerRuleModel := toRunnerRuleResourceModel(*runnerRule)

	data.Id = runnerRuleModel.Id
	data.RunnerId = runnerRuleModel.RunnerId
	data.EnvTypeId = runnerRuleModel.EnvTypeId
	data.ProjectId = runnerRuleModel.ProjectId

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
