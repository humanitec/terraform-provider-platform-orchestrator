package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
	"terraform-provider-humanitec-v2/internal/ref"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ModuleResource{}
var _ resource.ResourceWithImportState = &ModuleResource{}

func NewModuleResource() resource.Resource {
	return &ModuleResource{}
}

// ModuleResource defines the resource implementation.
type ModuleResource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

type ModuleResourceModel struct {
	Id               types.String         `tfsdk:"id"`
	Description      types.String         `tfsdk:"description"`
	ResourceType     types.String         `tfsdk:"resource_type"`
	ModuleSource     types.String         `tfsdk:"module_source"`
	ModuleSourceCode types.String         `tfsdk:"module_source_code"`
	ModuleInputs     jsontypes.Normalized `tfsdk:"module_inputs"`
	ModuleParams     basetypes.MapValue   `tfsdk:"module_params"`
	ProviderMapping  basetypes.MapValue   `tfsdk:"provider_mapping"`
	Coprovisioned    types.List           `tfsdk:"coprovisioned"`
	Dependencies     basetypes.MapValue   `tfsdk:"dependencies"`
}

type ModuleCoprovisionedModel struct {
	Class                     types.String         `tfsdk:"class"`
	CopyDependentsFromCurrent types.Bool           `tfsdk:"copy_dependents_from_current"`
	Id                        types.String         `tfsdk:"id"`
	IsDependentOnCurrent      types.Bool           `tfsdk:"is_dependent_on_current"`
	Params                    jsontypes.Normalized `tfsdk:"params"`
	Type                      types.String         `tfsdk:"type"`
}

type ModuleDependenciesModel struct {
	Class  types.String         `tfsdk:"class"`
	Id     types.String         `tfsdk:"id"`
	Params jsontypes.Normalized `tfsdk:"params"`
	Type   types.String         `tfsdk:"type"`
}

type ModuleParamModel struct {
	Type        types.String `tfsdk:"type"`
	IsOptional  types.Bool   `tfsdk:"is_optional"`
	Description types.String `tfsdk:"description"`
}

func ModuleCoprovisionedModelAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"class":                        types.StringType,
		"copy_dependents_from_current": types.BoolType,
		"id":                           types.StringType,
		"is_dependent_on_current":      types.BoolType,
		"params":                       types.StringType,
		"type":                         types.StringType,
	}
}

func ModuleDependenciesModelAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"class":  types.StringType,
		"id":     types.StringType,
		"params": types.StringType,
		"type":   types.StringType,
	}
}

func ModuleParamsModelAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":        types.StringType,
		"is_optional": types.BoolType,
		"description": types.StringType,
	}
}

func (r *ModuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_module"
}

func (r *ModuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Module resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique identifier for a module",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z](?:-?[a-z0-9]+)+$"), ""),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "An optional text description for this module",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"resource_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The resource type that this module provisions. Changing this will force a recreation of the resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(2),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"module_source": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The source of the OpenTofu module backing this module. Required. Must be set to 'inline' if module_source_code is set.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 200),
				},
			},
			"module_source_code": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The source code of the OpenTofu module backing this module. Required, if module source is not defined.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 2000),
				},
			},
			"module_inputs": schema.StringAttribute{
				MarkdownDescription: "The JSON encoded string which represents the inputs to the module. These may contain expressions referencing the modules context.",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(2000),
				},
				Optional:   true,
				Computed:   true,
				CustomType: jsontypes.NormalizedType{},
			},
			"module_params": schema.MapNestedAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "A mapping of module parameters available when provisioning using this module.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The type of the module parameter. string, number, bool, map, list, or any",
						},
						"is_optional": schema.BoolAttribute{
							Computed:            true,
							Optional:            true,
							MarkdownDescription: "If true, this module parameter is optional",
						},
						"description": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "An optional text description for this module parameter",
						},
					},
				},
			},
			"provider_mapping": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A mapping of module providers to use when provisioning using this module.",
			},
			"coprovisioned": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A set of resources to provision after or in parallel with the resource of the current module.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"class": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "A resource class requested by the resource graph. 'default' is the default value.",
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 63),
								stringvalidator.RegexMatches(regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9-]{0,61}[A-Za-z0-9]$"), ""),
							},
						},
						"copy_dependents_from_current": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "If true, all resources that depend on the current resource will also depend on (be provisioned after) this coprovisioned resource.\n",
						},
						"id": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "A specific resource id requested by the resource graph",
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 63),
								stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+(?:-+[a-z0-9]+)*(?:\.[a-z0-9]+(?:-+[a-z0-9]+)*)*$`), ""),
							},
						},
						"is_dependent_on_current": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "If true, this coprovisioned resource will have a dependency on the current resource so that the current\nresource must be successfully provisioned before the coprovisioned one is.\n",
						},
						"params": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "A JSON encoded string representing the parameters to pass for provisioning.",
							Computed:            true,
							CustomType:          jsontypes.NormalizedType{},
						},
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The resource type to provision",
							Validators: []validator.String{
								stringvalidator.LengthBetween(2, 63),
								stringvalidator.RegexMatches(regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9-]{0,61}[A-Za-z0-9]$"), ""),
							},
						},
					},
				},
			},
			"dependencies": schema.MapNestedAttribute{
				Optional:            true,
				MarkdownDescription: "A mapping of alias to resource dependencies that must be provisioned with this module",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"class": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "A resource class requested by the resource graph. 'default' is the default value.",
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 63),
								stringvalidator.RegexMatches(regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9-]{0,61}[A-Za-z0-9]$"), ""),
							},
						},
						"id": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "A specific resource id requested by the resource graph",
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 63),
								stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+(?:-+[a-z0-9]+)*(?:\.[a-z0-9]+(?:-+[a-z0-9]+)*)*$`), ""),
							},
						},
						"params": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "A JSON encoded string representing the parameters to pass for provisioning.",
							CustomType:          jsontypes.NormalizedType{},
							Computed:            true,
						},
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The resource type to provision",
							Validators: []validator.String{
								stringvalidator.LengthBetween(2, 63),
								stringvalidator.RegexMatches(regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9-]{0,61}[A-Za-z0-9]$"), ""),
							},
						},
					},
				},
			},
		},
	}
}
func (r *ModuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ModuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ModuleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	coprovisioned, err := toCoprovisionedFromModel(ctx, data.Coprovisioned)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to parse coprovisioned from model: %s", err))
		return
	}

	dependencies, err := toDependenciesFromModel(ctx, data.Dependencies)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to parse dependencies from model: %s", err))
		return
	}

	inputs := make(map[string]interface{})
	if !data.ModuleInputs.IsNull() && !data.ModuleInputs.IsUnknown() {
		if diags := data.ModuleInputs.Unmarshal(&inputs); diags.HasError() {
			resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Module inputs is not a valid object: %s", diags.Errors()))
			return
		}
	}

	providerMappings := make(map[string]string)
	if !data.ProviderMapping.IsNull() && !data.ProviderMapping.IsUnknown() {
		for key, value := range data.ProviderMapping.Elements() {
			if strValue, ok := value.(basetypes.StringValue); ok {
				providerMappings[key] = strValue.ValueString()
			} else {
				resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Provider mapping for key %s is not a string value", key))
				return
			}
		}
	}

	moduleParams, err := toModuleParamsFromModel(ctx, data.ModuleParams)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to parse module params from model: %s", err))
		return
	}

	httpResp, err := r.cpClient.CreateModuleWithResponse(ctx, r.orgId, canyoncp.CreateModuleJSONRequestBody{
		Id:               data.Id.ValueString(),
		Description:      ref.RefStringEmptyNil(data.Description.ValueString()),
		Coprovisioned:    coprovisioned,
		Dependencies:     dependencies,
		ModuleSource:     data.ModuleSource.ValueString(),
		ModuleSourceCode: fromStringValueToStringPointer(data.ModuleSourceCode),
		ModuleInputs:     inputs,
		ModuleParams:     moduleParams,
		ProviderMapping:  providerMappings,
		ResourceType:     data.ResourceType.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to create module, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != 201 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to create module, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	if data, err = toModuleResourceModel(ctx, *httpResp.JSON201); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to ModuleResourceModel: %s", err))
		return
	} else {
		// Save data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}

}

func (r *ModuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ModuleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.cpClient.GetModuleWithResponse(ctx, r.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Unable to read module, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddWarning(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Module with ID %s not found, assuming it has been deleted.", data.Id.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read module, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	if moduleModel, err := toModuleResourceModel(ctx, *httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert module response to model: %s", err))
		return
	} else {
		resp.Diagnostics.Append(resp.State.Set(ctx, ref.Ref(moduleModel))...)
	}
}

func (r *ModuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ModuleResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	coprovisioned, err := toCoprovisionedFromModel(ctx, data.Coprovisioned)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to parse coprovisioned from model: %s", err))
		return
	}

	dependencies, err := toDependenciesFromModel(ctx, data.Dependencies)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to parse dependencies from model: %s", err))
		return
	}

	inputs := make(map[string]interface{})
	if !data.ModuleInputs.IsNull() && !data.ModuleInputs.IsUnknown() {
		if err := json.Unmarshal([]byte(data.ModuleInputs.ValueString()), &inputs); err != nil {
			resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Data module inputs is not a valid object: %s", err))
			return
		}
	}

	providerMappings := make(map[string]string)
	if !data.ProviderMapping.IsNull() && !data.ProviderMapping.IsUnknown() {
		if diags := data.ProviderMapping.ElementsAs(ctx, &providerMappings, false); diags.HasError() {
			resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to parse provider mapping from model: %s", diags.Errors()))
			return
		}
	}

	moduleParams, err := toModuleParamsFromModel(ctx, data.ModuleParams)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to parse module params from model: %s", err))
		return
	}

	id := state.Id.ValueString()

	var updateBody = canyoncp.UpdateModuleJSONRequestBody{
		Description:      ref.RefStringEmptyNil(data.Description.ValueString()),
		Dependencies:     ref.Ref(dependencies),
		ModuleInputs:     ref.Ref(inputs),
		ModuleParams:     ref.Ref(moduleParams),
		ProviderMapping:  ref.Ref(providerMappings),
		ModuleSource:     fromStringValueToStringPointer(data.ModuleSource),
		ModuleSourceCode: fromStringValueToStringPointer(data.ModuleSourceCode),
		Coprovisioned:    ref.Ref(coprovisioned),
	}

	httpResp, err := r.cpClient.UpdateModuleWithResponse(ctx, r.orgId, id, updateBody)
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to update module, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddError(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Module with ID %s not found, assuming it has been deleted.", id))
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to update module, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	if data, err = toModuleResourceModel(ctx, *httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to ModuleResourceModel: %s", err))
		return
	} else {
		// Save data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}

}

func (r *ModuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ModuleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.cpClient.DeleteModuleWithResponse(ctx, r.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to delete module, got error: %s", err))
		return
	}

	switch httpResp.StatusCode() {
	case 204:
		// Successfully deleted, no further action needed.
	case 404:
		// If the resource is not found, we can consider it deleted.
		resp.Diagnostics.AddWarning(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Module with ID %s not found, assuming it has been deleted.", data.Id.ValueString()))
	default:
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to delete module, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *ModuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func toCoprovisionedFromModel(ctx context.Context, coprovisioned basetypes.ListValue) ([]canyoncp.ModuleCoProvisionManifest, error) {
	var result = make([]canyoncp.ModuleCoProvisionManifest, len(coprovisioned.Elements()))
	if !coprovisioned.IsNull() && !coprovisioned.IsUnknown() {
		for i, cp := range coprovisioned.Elements() {
			cpObj, ok := cp.(basetypes.ObjectValue)
			if !ok {
				return nil, fmt.Errorf("expected object value for coprovisioned item")
			}

			var cpModel ModuleCoprovisionedModel
			if diags := cpObj.As(ctx, &cpModel, basetypes.ObjectAsOptions{}); diags.HasError() {
				return nil, fmt.Errorf("failed to convert coprovisioned model: %v", diags.Errors())
			}

			params := make(map[string]interface{})
			if !cpModel.Params.IsNull() && !cpModel.Params.IsUnknown() {
				if err := json.Unmarshal([]byte(cpModel.Params.ValueString()), &params); err != nil {
					return nil, fmt.Errorf("failed to parse params: %s", err)
				}
			}

			result[i] = canyoncp.ModuleCoProvisionManifest{
				Type:                      cpModel.Type.ValueString(),
				CopyDependentsFromCurrent: cpModel.CopyDependentsFromCurrent.ValueBool(),
				IsDependentOnCurrent:      cpModel.IsDependentOnCurrent.ValueBool(),
				Class:                     ref.RefStringEmptyNil(cpModel.Class.ValueString()),
				Id:                        ref.RefStringEmptyNil(cpModel.Id.ValueString()),
				Params:                    params,
			}
		}

	}

	return result, nil
}

func toDependenciesFromModel(ctx context.Context, dependencies basetypes.MapValue) (map[string]canyoncp.ModuleDependencyManifest, error) {
	result := make(map[string]canyoncp.ModuleDependencyManifest)
	if !dependencies.IsNull() && !dependencies.IsUnknown() {
		for alias, dep := range dependencies.Elements() {
			depObj, ok := dep.(basetypes.ObjectValue)
			if !ok {
				return nil, fmt.Errorf("expected object value for dependency %s", alias)
			}

			var depModel ModuleDependenciesModel
			if diags := depObj.As(ctx, &depModel, basetypes.ObjectAsOptions{}); diags.HasError() {
				return nil, fmt.Errorf("failed to convert dependencies model: %v", diags.Errors())
			}

			params := make(map[string]interface{})
			if !depModel.Params.IsNull() && !depModel.Params.IsUnknown() {
				if err := json.Unmarshal([]byte(depModel.Params.ValueString()), &params); err != nil {
					return nil, fmt.Errorf("failed to parse params: %s", err)
				}
			}

			result[alias] = canyoncp.ModuleDependencyManifest{
				Type:   depModel.Type.ValueString(),
				Class:  ref.RefStringEmptyNil(depModel.Class.ValueString()),
				Id:     ref.RefStringEmptyNil(depModel.Id.ValueString()),
				Params: params,
			}
		}
	}
	return result, nil
}

func toModuleParamsFromModel(ctx context.Context, moduleParams basetypes.MapValue) (map[string]canyoncp.ModuleParamItem, error) {
	result := make(map[string]canyoncp.ModuleParamItem)
	if !moduleParams.IsNull() && !moduleParams.IsUnknown() {
		for key, value := range moduleParams.Elements() {
			paramObj, ok := value.(basetypes.ObjectValue)
			if !ok {
				return nil, fmt.Errorf("expected object value for module param %s", key)
			}

			var paramModel ModuleParamModel
			if diags := paramObj.As(ctx, &paramModel, basetypes.ObjectAsOptions{}); diags.HasError() {
				return nil, fmt.Errorf("failed to convert module param model: %v", diags.Errors())
			}

			result[key] = canyoncp.ModuleParamItem{
				Description: paramModel.Description.ValueStringPointer(),
				IsOptional:  paramModel.IsOptional.ValueBool(),
				Type:        canyoncp.ModuleParamItemType(paramModel.Type.ValueString()),
			}
		}
	}
	return result, nil
}

func toModuleResourceModel(ctx context.Context, item canyoncp.Module) (ModuleResourceModel, error) {
	var coprovisioned basetypes.ListValue
	var diags diag.Diagnostics
	if item.Coprovisioned != nil {
		coprovisionedList := make([]attr.Value, len(item.Coprovisioned))
		for i, cp := range item.Coprovisioned {
			var params jsontypes.Normalized
			if cp.Params != nil {
				paramJson, _ := json.Marshal(cp.Params)
				params = jsontypes.NewNormalizedValue(string(paramJson))
			} else {
				params = jsontypes.NewNormalizedNull()
			}

			cpModel := ModuleCoprovisionedModel{
				Class:                     toStringValueOrNil(cp.Class),
				CopyDependentsFromCurrent: types.BoolValue(cp.CopyDependentsFromCurrent),
				Id:                        toStringValueOrNil(cp.Id),
				IsDependentOnCurrent:      types.BoolValue(cp.IsDependentOnCurrent),
				Params:                    params,
				Type:                      types.StringValue(cp.Type),
			}
			objectValue, diags := types.ObjectValueFrom(ctx, ModuleCoprovisionedModelAttributeTypes(), cpModel)
			if diags.HasError() {
				return ModuleResourceModel{}, fmt.Errorf("failed to build coprovisioned model from API response: %v", diags.Errors())
			}
			coprovisionedList[i] = objectValue
		}
		coprovisioned, diags = types.ListValue(types.ObjectType{AttrTypes: ModuleCoprovisionedModelAttributeTypes()}, coprovisionedList)
		if diags.HasError() {
			return ModuleResourceModel{}, fmt.Errorf("failed to build coprovisioned list model from API response: %v", diags.Errors())
		}
	}

	var dependencies basetypes.MapValue
	if item.Dependencies != nil {
		dependenciesMap := make(map[string]attr.Value)
		for alias, dep := range item.Dependencies {
			var params jsontypes.Normalized
			if dep.Params != nil {
				paramJson, _ := json.Marshal(dep.Params)
				params = jsontypes.NewNormalizedValue(string(paramJson))
			} else {
				params = jsontypes.NewNormalizedNull()
			}
			depModel := ModuleDependenciesModel{
				Type:   types.StringValue(dep.Type),
				Class:  toStringValueOrNil(dep.Class),
				Id:     toStringValueOrNil(dep.Id),
				Params: params,
			}
			objectValue, diags := types.ObjectValueFrom(ctx, ModuleDependenciesModelAttributeTypes(), depModel)
			if diags.HasError() {
				return ModuleResourceModel{}, fmt.Errorf("failed to build dependencies model model parsing API response: %v", diags.Errors())
			}
			dependenciesMap[alias] = objectValue
		}
		dependencies, diags = types.MapValue(types.ObjectType{AttrTypes: ModuleDependenciesModelAttributeTypes()}, dependenciesMap)
		if diags.HasError() {
			return ModuleResourceModel{}, fmt.Errorf("failed to build dependencies map model parsing API response: %v", diags.Errors())
		}
	}

	var moduleParams basetypes.MapValue
	if item.ModuleParams != nil {
		moduleParamsMap := make(map[string]attr.Value)
		for key, def := range item.ModuleParams {
			objectValue, diags := types.ObjectValueFrom(ctx, ModuleParamsModelAttributeTypes(), ModuleParamModel{
				Type:        types.StringValue(string(def.Type)),
				IsOptional:  types.BoolValue(def.IsOptional),
				Description: toStringValueOrNil(def.Description),
			})
			if diags.HasError() {
				return ModuleResourceModel{}, fmt.Errorf("failed to build dependencies model model parsing API response: %v", diags.Errors())
			}
			moduleParamsMap[key] = objectValue
		}
		moduleParams, diags = types.MapValue(types.ObjectType{AttrTypes: ModuleParamsModelAttributeTypes()}, moduleParamsMap)
		if diags.HasError() {
			return ModuleResourceModel{}, fmt.Errorf("failed to build module params map model parsing API response: %v", diags.Errors())
		}
	}

	var inputs jsontypes.Normalized
	if item.ModuleInputs != nil {
		inputsJson, _ := json.Marshal(item.ModuleInputs)
		inputs = jsontypes.NewNormalizedValue(string(inputsJson))
	} else {
		inputs = jsontypes.NewNormalizedNull()
	}

	var providerMapping basetypes.MapValue
	if item.ProviderMapping != nil {
		var providerMappingMap = make(map[string]types.String)
		for key, value := range item.ProviderMapping {
			providerMappingMap[key] = types.StringValue(value)
		}
		providerMapping, diags = types.MapValueFrom(ctx, types.StringType, providerMappingMap)
		if diags.HasError() {
			return ModuleResourceModel{}, fmt.Errorf("failed to build provider mapping from model parsing API response: %v", diags.Errors())
		}
	}

	return ModuleResourceModel{
		Id:               types.StringValue(item.Id),
		Description:      types.StringPointerValue(item.Description),
		ResourceType:     types.StringValue(item.ResourceType),
		ModuleSource:     types.StringValue(item.ModuleSource),
		ModuleSourceCode: toStringValueOrNil(item.ModuleSourceCode),
		ModuleInputs:     inputs,
		ModuleParams:     moduleParams,
		ProviderMapping:  providerMapping,
		Coprovisioned:    coprovisioned,
		Dependencies:     dependencies,
	}, nil
}
