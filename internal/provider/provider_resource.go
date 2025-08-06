package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
	"terraform-provider-humanitec-v2/internal/ref"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProviderResource{}
var _ resource.ResourceWithImportState = &ProviderResource{}

func NewProviderResource() resource.Resource {
	return &ProviderResource{}
}

// ProviderResource defines the resource implementation.
type ProviderResource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// ProviderResourceModel describes the resource data model.
type ProviderResourceModel struct {
	Id                types.String         `tfsdk:"id"`
	Description       types.String         `tfsdk:"description"`
	ProviderType      types.String         `tfsdk:"provider_type"`
	Source            types.String         `tfsdk:"source"`
	VersionConstraint types.String         `tfsdk:"version_constraint"`
	Configuration     jsontypes.Normalized `tfsdk:"configuration"`
}

func (r *ProviderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_provider"
}

func (r *ProviderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Provider resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the Provider.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
						"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Module Provider.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"provider_type": schema.StringAttribute{
				MarkdownDescription: "The type of the provider, e.g. `aws`, `gcp`, `azure`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z][a-z0-9_-]+$`),
						"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
					),
					stringvalidator.LengthAtMost(100),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "The source of the provider, e.g. `hashicorp/aws`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^([a-z0-9.-]+/)?([a-z][a-z0-9_-]+)/([a-z][a-z0-9_-]+)$`),
						"must start with a lowercase letter, can contain lowercase letters, numbers, underscores, slashes and dots and can not be empty.",
					),
					stringvalidator.LengthAtMost(100),
				},
			},
			"version_constraint": schema.StringAttribute{
				MarkdownDescription: "The version constraint for the provider, e.g. `~> 2.0`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^((=|!=|>|>=|<|<=|~>) ([0-9.]+(-.+)?))(, (=|!=|>|>=|<|<=|~>) ([0-9.]+(-.+)?))*$`),
						"must be a valid version constraint, e.g. `~> 2.0`.",
					),
					stringvalidator.LengthAtMost(100),
				},
			},
			"configuration": schema.StringAttribute{
				MarkdownDescription: "JSON encoded configuration of the provider.",
				Optional:            true,
				CustomType:          jsontypes.NormalizedType{},
				Computed:            true,
			},
		},
	}
}

func (r *ProviderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProviderResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var configuration map[string]interface{}
	if !data.Configuration.IsNull() && !data.Configuration.IsUnknown() {
		if diags := data.Configuration.Unmarshal(&configuration); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	httpResp, err := r.cpClient.CreateModuleProviderWithResponse(ctx, r.orgId, canyoncp.CreateModuleProviderJSONRequestBody{
		Id:                data.Id.ValueString(),
		Description:       ref.RefStringEmptyNil(data.Description.ValueString()),
		ProviderType:      data.ProviderType.ValueString(),
		Source:            data.Source.ValueString(),
		VersionConstraint: data.VersionConstraint.ValueString(),
		Configuration:     configuration,
	})
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to create provider, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != 201 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to create provider, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, ref.Ref(toProviderResourceModel(*httpResp.JSON201)))...)
}

func (r *ProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProviderResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.cpClient.GetModuleProviderWithResponse(ctx, r.orgId, data.ProviderType.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Unable to read module provider, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read module provider, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ref.Ref(toProviderResourceModel(*httpResp.JSON200)))...)
}

func (r *ProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ProviderResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var configuration map[string]interface{}
	if !data.Configuration.IsNull() && !data.Configuration.IsUnknown() {
		if diags := data.Configuration.Unmarshal(&configuration); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	id := state.Id.ValueString()
	providerType := state.ProviderType.ValueString()
	var updateBody = canyoncp.UpdateModuleProviderJSONRequestBody{
		Description:       ref.RefStringEmptyNil(data.Description.ValueString()),
		VersionConstraint: ref.RefStringEmptyNil(data.VersionConstraint.ValueString()),
		Configuration:     ref.Ref(configuration),
	}

	httpResp, err := r.cpClient.UpdateModuleProviderWithResponse(ctx, r.orgId, providerType, id, updateBody)
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to update module provider, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to update runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ref.Ref(toProviderResourceModel(*httpResp.JSON200)))...)
}

func (r *ProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProviderResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.cpClient.DeleteModuleProviderWithResponse(ctx, r.orgId, data.ProviderType.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to delete module provider, got error: %s", err))
		return
	}

	switch httpResp.StatusCode() {
	case 204:
		// Successfully deleted, no further action needed.
	case 404:
		// If the resource is not found, we can consider it deleted.
		resp.Diagnostics.AddWarning(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Module provider with ID %s not found, assuming it has been deleted.", data.Id.ValueString()))
	default:
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to delete module provider, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *ProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ".")
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID to be in the format 'provider_type.id', got: %s", req.ID),
		)
		return
	}

	providerTypeValue := idParts[0]
	idValue := idParts[1]
	if providerTypeValue == "" || idValue == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID to have non-empty provider type and ID, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("provider_type"), providerTypeValue)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idValue)...)
}

func toProviderResourceModel(item canyoncp.ModuleProvider) ProviderResourceModel {
	var configuration jsontypes.Normalized
	if item.Configuration != nil {
		configJson, _ := json.Marshal(item.Configuration)
		configuration = jsontypes.NewNormalizedValue(string(configJson))
	}

	return ProviderResourceModel{
		Id:                types.StringValue(item.Id),
		Description:       types.StringPointerValue(item.Description),
		ProviderType:      types.StringValue(item.ProviderType),
		Source:            types.StringValue(item.Source),
		VersionConstraint: types.StringValue(item.VersionConstraint),
		Configuration:     configuration,
	}
}
