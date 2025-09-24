package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceTypeResource{}
var _ resource.ResourceWithImportState = &ResourceTypeResource{}

func NewResourceTypeResource() resource.Resource {
	return &ResourceTypeResource{}
}

// ResourceTypeResource defines the resource implementation.
type ResourceTypeResource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// ResourceTypeResourceModel describes the resource data model.
type ResourceTypeResourceModel struct {
	Id                    types.String         `tfsdk:"id"`
	Description           types.String         `tfsdk:"description"`
	OutputSchema          jsontypes.Normalized `tfsdk:"output_schema"`
	IsDeveloperAccessible types.Bool           `tfsdk:"is_developer_accessible"`
}

func (r *ResourceTypeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_type"
}

func (r *ResourceTypeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Resource Type resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the Resource Type.",
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
				MarkdownDescription: "The description of the Resource Type.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"output_schema": schema.StringAttribute{
				MarkdownDescription: "The JSON schema for output parameters.",
				Required:            true,
				CustomType:          jsontypes.NormalizedType{},
			},
			"is_developer_accessible": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this resource type is for developers to use in the manifest. Resource types with this flag set to false, will not be available as types of resources in a manifest.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

func (r *ResourceTypeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ResourceTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceTypeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var description *string
	if v := data.Description.ValueString(); v != "" {
		description = &v
	}

	var outputSchema map[string]interface{}
	diags := data.OutputSchema.Unmarshal(&outputSchema)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	isDeveloperAccessible := data.IsDeveloperAccessible.ValueBool()

	httpResp, err := r.cpClient.CreateResourceTypeWithResponse(ctx, r.orgId, canyoncp.CreateResourceTypeJSONRequestBody{
		Id:                    data.Id.ValueString(),
		Description:           description,
		OutputSchema:          outputSchema,
		IsDeveloperAccessible: &isDeveloperAccessible,
	})
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to create resource type, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != 201 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to create resource type, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	data, err = toResourceTypeModel(*httpResp.JSON201)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Unable to convert resource type response, got error: %s", err))
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceTypeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.cpClient.GetResourceTypeWithResponse(ctx, r.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Unable to read resource type, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddWarning(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Resource type with ID %s not found in org %s", data.Id.ValueString(), r.orgId))
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read resource type, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	data, err = toResourceTypeModel(*httpResp.JSON200)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Unable to convert resource type response, got error: %s", err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceTypeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var outputSchema map[string]interface{}
	diags := data.OutputSchema.Unmarshal(&outputSchema)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	httpResp, err := r.cpClient.UpdateResourceTypeWithResponse(ctx, r.orgId, data.Id.ValueString(), canyoncp.UpdateResourceTypeJSONRequestBody{
		Description:           data.Description.ValueStringPointer(),
		OutputSchema:          &outputSchema,
		IsDeveloperAccessible: data.IsDeveloperAccessible.ValueBoolPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to update resource type, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to update resource type, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	data, err = toResourceTypeModel(*httpResp.JSON200)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Unable to convert resource type response, got error: %s", err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceTypeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.cpClient.DeleteResourceTypeWithResponse(ctx, r.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to delete resource type, got error: %s", err))
		return
	}

	switch httpResp.StatusCode() {
	case 204:
		// Successfully deleted, no further action needed.
	case 404:
		// If the resource is not found, we can consider it deleted.
		resp.Diagnostics.AddWarning(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Resource Type with ID %s not found, assuming it has been deleted.", data.Id.ValueString()))
	default:
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to delete resource type, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *ResourceTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func toResourceTypeModel(item canyoncp.ResourceType) (ResourceTypeResourceModel, error) {
	var description types.String
	if item.Description != nil {
		description = types.StringValue(*item.Description)
	} else {
		description = types.StringNull()
	}

	outputSchemaBytes, err := json.Marshal(item.OutputSchema)
	if err != nil {
		return ResourceTypeResourceModel{}, fmt.Errorf("unable to marshal output schema: %w", err)
	}

	return ResourceTypeResourceModel{
		Id:                    types.StringValue(item.Id),
		Description:           description,
		OutputSchema:          jsontypes.NewNormalizedValue(string(outputSchemaBytes)),
		IsDeveloperAccessible: types.BoolValue(item.IsDeveloperAccessible),
	}, nil
}
