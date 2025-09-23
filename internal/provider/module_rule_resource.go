package provider

import (
	"context"
	"fmt"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
	"terraform-provider-humanitec-v2/internal/ref"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ModuleRuleResource{}
var _ resource.ResourceWithImportState = &ModuleRuleResource{}

func NewModuleRuleResource() resource.Resource {
	return &ModuleRuleResource{}
}

// ModuleRuleResource defines the resource implementation.
type ModuleRuleResource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// ModuleRuleResourceModel describes the resource data model.
type ModuleRuleResourceModel struct {
	Id            types.String `tfsdk:"id"`
	ModuleId      types.String `tfsdk:"module_id"`
	ResourceClass types.String `tfsdk:"resource_class"`
	ResourceType  types.String `tfsdk:"resource_type"`
	ResourceId    types.String `tfsdk:"resource_id"`
	EnvTypeId     types.String `tfsdk:"env_type_id"`
	EnvId         types.String `tfsdk:"env_id"`
	ProjectId     types.String `tfsdk:"project_id"`
}

func (r *ModuleRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_module_rule"
}

func (r *ModuleRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Module Rule resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the Module Rule.",
				Computed:            true,
			},
			"module_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Module this rule applies to.",
				Required:            true,
			},
			"resource_class": schema.StringAttribute{
				MarkdownDescription: "A resource class requested by the resource graph. 'default' is the default value.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "A specific resource id requested by the resource graph.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_type": schema.StringAttribute{
				MarkdownDescription: "The resource type matched by this rule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"env_type_id": schema.StringAttribute{
				MarkdownDescription: "The environment type to match this rule. This environment type must exist in the org. Mutually exclusive with env_id.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"env_id": schema.StringAttribute{
				MarkdownDescription: "The environment id to match this rule. This environment id must exist in the org. Mutually exclusive with env_type_id.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The optional project id that this rule matches.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ModuleRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.cpClient = providerData.CpClient
	r.orgId = providerData.OrgId
}

func (r *ModuleRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ModuleRuleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.cpClient.CreateModuleRuleInOrgWithResponse(ctx, r.orgId, canyoncp.CreateModuleRuleInOrgJSONRequestBody{
		ModuleId:      data.ModuleId.ValueString(),
		ResourceClass: ref.RefStringEmptyNil(data.ResourceClass.ValueString()),
		ResourceId:    ref.RefStringEmptyNil(data.ResourceId.ValueString()),
		EnvTypeId:     ref.RefStringEmptyNil(data.EnvTypeId.ValueString()),
		ProjectId:     ref.RefStringEmptyNil(data.ProjectId.ValueString()),
		EnvId:         ref.RefStringEmptyNil(data.EnvId.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to create module rule, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != 201 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to create module rule, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, ref.Ref(toModuleRuleResourceModel(*httpResp.JSON201)))...)
}

func (r *ModuleRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ModuleRuleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.cpClient.GetModuleRuleInOrgWithResponse(ctx, r.orgId, uuid.MustParse(data.Id.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Unable to read module rule, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == 404 {
		resp.Diagnostics.AddWarning(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Module rule with ID %s not found in org %s", data.Id.ValueString(), r.orgId))
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read module rule, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ref.Ref(toModuleRuleResourceModel(*httpResp.JSON200)))...)
}

func (r *ModuleRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Not Supported", "The Module Rule resource does not support updates.")
}

func (r *ModuleRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ModuleRuleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.cpClient.DeleteModuleRuleInOrgWithResponse(ctx, r.orgId, uuid.MustParse(data.Id.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to delete module rule, got error: %s", err))
		return
	}

	switch httpResp.StatusCode() {
	case 204:
		// Successfully deleted, no further action needed.
	case 404:
		// If the resource is not found, we can consider it deleted.
		resp.Diagnostics.AddWarning(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Module rule with ID %s not found, assuming it has been deleted.", data.Id.ValueString()))
	default:
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to delete module rule, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *ModuleRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func toModuleRuleResourceModel(item canyoncp.Rule) ModuleRuleResourceModel {
	return ModuleRuleResourceModel{
		Id:            types.StringValue(item.Id.String()),
		ModuleId:      types.StringValue(item.ModuleId),
		ResourceClass: types.StringValue(item.ResourceClass),
		ResourceType:  types.StringValue(item.ResourceType),
		ResourceId:    toStringValueOrNil(item.ResourceId),
		EnvTypeId:     toStringValueOrNil(item.EnvTypeId),
		ProjectId:     toStringValueOrNil(item.ProjectId),
		EnvId:         toStringValueOrNil(item.EnvId),
	}
}
