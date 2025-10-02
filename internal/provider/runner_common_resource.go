package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
	"terraform-provider-humanitec-v2/internal/ref"
)

var _ resource.Resource = &commonRunnerResource{}
var _ resource.ResourceWithImportState = &commonRunnerResource{}

type commonRunnerResource struct {
	SubType                          string
	SchemaDef                        schema.Schema
	ReadApiResponseIntoModel         func(canyoncp.Runner, commonRunnerModel) (commonRunnerModel, error)
	ConvertRunnerConfigIntoCreateApi func(ctx context.Context, obj types.Object) (canyoncp.RunnerConfiguration, error)
	ConvertRunnerConfigIntoUpdateApi func(ctx context.Context, obj types.Object) (canyoncp.RunnerConfigurationUpdate, error)

	// params set during Configure()
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

func (r *commonRunnerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.SubType
}

var commonRunnerStateStorageResourceSchema = schema.SingleNestedAttribute{
	MarkdownDescription: "The state storage configuration for the Kubernetes Runner.",
	Required:            true,
	Attributes: map[string]schema.Attribute{
		"type": schema.StringAttribute{
			MarkdownDescription: "The type of state storage configuration for the Kubernetes Runner.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("kubernetes"),
			},
		},
		"kubernetes_configuration": schema.SingleNestedAttribute{
			MarkdownDescription: "The Kubernetes state storage configuration for the Kubernetes Runner.",
			Required:            true,
			Attributes: map[string]schema.Attribute{
				"namespace": schema.StringAttribute{
					MarkdownDescription: "The namespace for the Kubernetes state storage configuration.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.LengthAtMost(63),
					},
				},
			},
		},
	},
}

func (r *commonRunnerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.SchemaDef
}

func (r *commonRunnerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *commonRunnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data commonRunnerModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	runnerConfigurationFromObject, err := r.ConvertRunnerConfigIntoCreateApi(ctx, data.RunnerConfiguration)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to parse runner configuration from model: %s", err))
		return
	}

	stateStorageConfigurationFromObject, err := createStateStorageConfigurationFromObject(ctx, data.StateStorageConfiguration)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to parse state storage configuration from model: %s", err))
		return
	}

	httpResp, err := r.cpClient.CreateRunnerWithResponse(ctx, r.orgId, canyoncp.CreateRunnerJSONRequestBody{
		Id:                        data.Id.ValueString(),
		Description:               ref.RefStringEmptyNil(data.Description.ValueString()),
		RunnerConfiguration:       runnerConfigurationFromObject,
		StateStorageConfiguration: stateStorageConfigurationFromObject,
	})
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to create runner, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to create runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	if data, err = r.ReadApiResponseIntoModel(*httpResp.JSON201, data); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to %s: %s", r.SubType, err))
		return
	} else {
		// Save data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}

}

func (r *commonRunnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data commonRunnerModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.cpClient.GetRunnerWithResponse(ctx, r.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Unable to read runner, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddWarning(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Runner with ID %s not found, assuming it has been deleted.", data.Id.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	if data, err = r.ReadApiResponseIntoModel(*httpResp.JSON200, data); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to %s: %s", r.SubType, err))
		return
	} else {
		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}

}

func (r *commonRunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state commonRunnerModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateRunnerConfigurationBodyFromObject, err := r.ConvertRunnerConfigIntoUpdateApi(ctx, data.RunnerConfiguration)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to parse runner configuration from model: %s", err))
		return
	}

	updateStateStorageBodyConfigurationFromObject, err := createStateStorageConfigurationFromObject(ctx, data.StateStorageConfiguration)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to parse state storage configuration from model: %s", err))
		return
	}

	id := state.Id.ValueString()
	var updateBody = canyoncp.UpdateRunnerJSONRequestBody{
		Description:               ref.RefStringEmptyNil(data.Description.ValueString()),
		RunnerConfiguration:       &updateRunnerConfigurationBodyFromObject,
		StateStorageConfiguration: &updateStateStorageBodyConfigurationFromObject,
	}

	httpResp, err := r.cpClient.UpdateRunnerWithResponse(ctx, r.orgId, id, updateBody)
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to update runner, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to update runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	if data, err = r.ReadApiResponseIntoModel(*httpResp.JSON200, data); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to %s: %s", r.SubType, err))
		return
	} else {
		// Save data info into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}
}

func (r *commonRunnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data commonRunnerModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.cpClient.DeleteRunnerWithResponse(ctx, r.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to delete runner, got error: %s", err))
		return
	}

	switch httpResp.StatusCode() {
	case http.StatusNoContent:
		// Successfully deleted, no further action needed.
	case http.StatusNotFound:
		// If the resource is not found, we can consider it deleted.
		resp.Diagnostics.AddWarning(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Runner with ID %s not found, assuming it has been deleted.", data.Id.ValueString()))
	default:
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to delete runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *commonRunnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
