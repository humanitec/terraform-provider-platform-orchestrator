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
var _ datasource.DataSource = &ModuleRuleDataSource{}

func NewModuleRuleDataSource() datasource.DataSource {
	return &ModuleRuleDataSource{}
}

// ModuleRuleDataSource defines the data source implementation.
type ModuleRuleDataSource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// ModuleRuleDataSourceModel describes the data source data model.
type ModuleRuleDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	ModuleId      types.String `tfsdk:"module_id"`
	ResourceClass types.String `tfsdk:"resource_class"`
	ResourceType  types.String `tfsdk:"resource_type"`
	ResourceId    types.String `tfsdk:"resource_id"`
	EnvTypeId     types.String `tfsdk:"env_type_id"`
	EnvId         types.String `tfsdk:"env_id"`
	ProjectId     types.String `tfsdk:"project_id"`
}

func (d *ModuleRuleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_module_rule"
}

func (d *ModuleRuleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Module Rule data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the Module Rule.",
				Required:            true,
			},
			"module_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Module this rule applies to.",
				Computed:            true,
			},
			"resource_class": schema.StringAttribute{
				MarkdownDescription: "A resource class requested by the resource graph.",
				Computed:            true,
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "A specific resource id requested by the resource graph.",
				Computed:            true,
			},
			"resource_type": schema.StringAttribute{
				MarkdownDescription: "The resource type matched by this rule.",
				Computed:            true,
			},
			"env_type_id": schema.StringAttribute{
				MarkdownDescription: "The environment type to match this rule.",
				Computed:            true,
			},
			"env_id": schema.StringAttribute{
				MarkdownDescription: "The environment id to match this rule.",
				Computed:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The project id that this rule matches.",
				Computed:            true,
			},
		},
	}
}

func (d *ModuleRuleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ModuleRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ModuleRuleDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetModuleRuleInOrgWithResponse(ctx, d.orgId, uuid.MustParse(data.Id.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read module rule, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read module rule, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	moduleRule := httpResp.JSON200

	// Convert the module rule to the data source model using the existing helper function
	moduleRuleModel := toModuleRuleResourceModel(*moduleRule)

	data.Id = moduleRuleModel.Id
	data.ModuleId = moduleRuleModel.ModuleId
	data.ResourceClass = moduleRuleModel.ResourceClass
	data.ResourceType = moduleRuleModel.ResourceType
	data.ResourceId = moduleRuleModel.ResourceId
	data.EnvTypeId = moduleRuleModel.EnvTypeId
	data.EnvId = moduleRuleModel.EnvId
	data.ProjectId = moduleRuleModel.ProjectId

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
