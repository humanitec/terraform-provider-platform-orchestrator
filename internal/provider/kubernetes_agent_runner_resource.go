package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
	"terraform-provider-humanitec-v2/internal/ref"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KubernetesAgentRunnerResource{}
var _ resource.ResourceWithImportState = &KubernetesAgentRunnerResource{}

func NewKubernetesAgentRunnerResource() resource.Resource {
	return &KubernetesAgentRunnerResource{}
}

// KubernetesAgentRunner defines the resource implementation.
type KubernetesAgentRunnerResource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// KubernetesAgentRunnerModel describes the resource data model.
type KubernetesAgentRunnerResourceModel struct {
	Id                        types.String `tfsdk:"id"`
	Description               types.String `tfsdk:"description"`
	RunnerConfiguration       types.Object `tfsdk:"runner_configuration"`
	StateStorageConfiguration types.Object `tfsdk:"state_storage_configuration"`
}

// KubernetesAgentRunnerConfiguration describes the runner configuration structure following SecretRef pattern.
type KubernetesAgentRunnerConfiguration struct {
	Key types.String             `tfsdk:"key"`
	Job KubernetesAgentRunnerJob `tfsdk:"job"`
}

type KubernetesAgentRunnerJob struct {
	Namespace      types.String         `tfsdk:"namespace"`
	ServiceAccount types.String         `tfsdk:"service_account"`
	PodTemplate    jsontypes.Normalized `tfsdk:"pod_template"`
}

func KubernetesAgentRunnerConfigurationAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key": types.StringType,
		"job": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"namespace":       types.StringType,
				"service_account": types.StringType,
				"pod_template":    types.StringType,
			},
		},
	}
}

type KubernetesAgentRunnerStateStorageConfigurationModel struct {
	Type                    string                                                        `tfsdk:"type"`
	KubernetesConfiguration KubernetesAgentRunnerKubernetesStateStorageConfigurationModel `tfsdk:"kubernetes_configuration"`
}

type KubernetesAgentRunnerKubernetesStateStorageConfigurationModel struct {
	Namespace string `tfsdk:"namespace"`
}

func KubernetesAgentRunnerStateStorageConfigurationAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type": types.StringType,
		"kubernetes_configuration": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"namespace": types.StringType,
			},
		},
	}
}

func (r *KubernetesAgentRunnerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_agent_runner"
}

func (r *KubernetesAgentRunnerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Kubernetes Agent Runner resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the Kubernetes Agent Runner.",
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
				MarkdownDescription: "The description of the Kubernetes Agent Runner.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"runner_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration of the Kubernetes Agent Runner.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"key": schema.StringAttribute{
						MarkdownDescription: "The public ed25519 key in PEM format used to identify the caller identity. The caller must hold the matching private key.",
						Required:            true,
					},
					"job": schema.SingleNestedAttribute{
						MarkdownDescription: "The job configuration for the Kubernetes Job triggered by the Kubernetes Agent Runner.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"namespace": schema.StringAttribute{
								MarkdownDescription: "The namespace for the Kubernetes Runner job.",
								Required:            true,
							},
							"service_account": schema.StringAttribute{
								MarkdownDescription: "The service account for the Kubernetes Runner job.",
								Required:            true,
							},
							"pod_template": schema.StringAttribute{
								MarkdownDescription: "JSON encoded pod template for the Kubernetes Runner job.",
								Optional:            true,
								CustomType:          jsontypes.NormalizedType{},
								Computed:            true,
							},
						},
					},
				},
			},
			"state_storage_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The state storage configuration for the Kubernetes Agent Runner.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "The type of state storage configuration for the Kubernetes Agent Runner.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("kubernetes"),
						},
					},
					"kubernetes_configuration": schema.SingleNestedAttribute{
						MarkdownDescription: "The Kubernetes state storage configuration for the Kubernetes Agent Runner.",
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
			},
		},
	}
}

func (r *KubernetesAgentRunnerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KubernetesAgentRunnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KubernetesAgentRunnerResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	runnerConfigurationFromObject, err := createKubernetesAgentRunnerConfigurationFromObject(ctx, data.RunnerConfiguration)
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

	if httpResp.StatusCode() != 201 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to create runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	if data, err = toKubernetesAgentRunnerResourceModel(*httpResp.JSON201); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesAgentRunnerResourceModel: %s", err))
		return
	} else {
		// Save data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}

}

func (r *KubernetesAgentRunnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KubernetesAgentRunnerResourceModel

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

	if httpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	if data, err = toKubernetesAgentRunnerResourceModel(*httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesAgentRunnerResourceModel: %s", err))
		return
	} else {
		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}

}

func (r *KubernetesAgentRunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state KubernetesAgentRunnerResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateRunnerConfigurationBodyFromObject, err := updateK8sAgentRunnerConfigurationFromObject(ctx, data.RunnerConfiguration)
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

	if httpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to update runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	if data, err = toKubernetesAgentRunnerResourceModel(*httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesAgentRunnerResourceModel: %s", err))
		return
	} else {
		// Save data info into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}
}

func (r *KubernetesAgentRunnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KubernetesAgentRunnerResourceModel

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
	case 204:
		// Successfully deleted, no further action needed.
	case 404:
		// If the resource is not found, we can consider it deleted.
		resp.Diagnostics.AddWarning(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Runner with ID %s not found, assuming it has been deleted.", data.Id.ValueString()))
	default:
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to delete runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *KubernetesAgentRunnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func parseKubernetesAgentRunnerConfigurationResponse(ctx context.Context, k8sAgentRunnerConfiguration canyoncp.K8sAgentRunnerConfiguration) (basetypes.ObjectValue, error) {
	runnerConfig := KubernetesAgentRunnerConfiguration{
		Key: types.StringValue(k8sAgentRunnerConfiguration.Key),
		Job: KubernetesAgentRunnerJob{
			Namespace:      types.StringValue(k8sAgentRunnerConfiguration.Job.Namespace),
			ServiceAccount: types.StringValue(k8sAgentRunnerConfiguration.Job.ServiceAccount),
		},
	}

	if k8sAgentRunnerConfiguration.Job.PodTemplate != nil {
		podTemplate, _ := json.Marshal(k8sAgentRunnerConfiguration.Job.PodTemplate)
		runnerConfig.Job.PodTemplate = jsontypes.NewNormalizedValue(string(podTemplate))
	} else {
		runnerConfig.Job.PodTemplate = jsontypes.NewNormalizedNull()
	}

	objectValue, diags := types.ObjectValueFrom(ctx, KubernetesAgentRunnerConfigurationAttributeTypes(), runnerConfig)
	if diags.HasError() {
		return basetypes.ObjectValue{}, fmt.Errorf("failed to build runner configuration from model parsing API response: %v", diags.Errors())
	}
	return objectValue, nil
}

func toKubernetesAgentRunnerResourceModel(item canyoncp.Runner) (KubernetesAgentRunnerResourceModel, error) {
	k8sAgentRunnerConfiguration, _ := item.RunnerConfiguration.AsK8sAgentRunnerConfiguration()
	k8sStateStorageConfiguration, _ := item.StateStorageConfiguration.AsK8sStorageConfiguration()

	runnerConfigurationModel, err := parseKubernetesAgentRunnerConfigurationResponse(context.Background(), k8sAgentRunnerConfiguration)
	if err != nil {
		return KubernetesAgentRunnerResourceModel{}, err
	}

	stateStorageConfigurationModel := parseStateStorageConfigurationResponse(context.Background(), k8sStateStorageConfiguration)
	if stateStorageConfigurationModel == nil {
		return KubernetesAgentRunnerResourceModel{}, errors.New("failed to parse state storage configuration")
	}

	return KubernetesAgentRunnerResourceModel{
		Id:                        types.StringValue(item.Id),
		Description:               types.StringPointerValue(item.Description),
		StateStorageConfiguration: *stateStorageConfigurationModel,
		RunnerConfiguration:       runnerConfigurationModel,
	}, nil
}

func createKubernetesAgentRunnerConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.RunnerConfiguration, error) {
	var runnerConfig KubernetesAgentRunnerConfiguration
	diags := obj.As(ctx, &runnerConfig, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return canyoncp.RunnerConfiguration{}, fmt.Errorf("failed to parse runner configuration from model: %v", diags.Errors())
	}

	var jobPodTemplate *map[string]interface{}
	if runnerConfig.Job.PodTemplate.ValueString() != "" {
		if err := json.Unmarshal([]byte(runnerConfig.Job.PodTemplate.ValueString()), &jobPodTemplate); err != nil {
			return canyoncp.RunnerConfiguration{}, fmt.Errorf("failed to parse pod template from model: %v", err)
		}
	}

	var runnerConfiguration = new(canyoncp.RunnerConfiguration)
	_ = runnerConfiguration.FromK8sAgentRunnerConfiguration(canyoncp.K8sAgentRunnerConfiguration{
		Key: runnerConfig.Key.ValueString(),
		Job: canyoncp.K8sRunnerJobConfig{
			Namespace:      runnerConfig.Job.Namespace.ValueString(),
			ServiceAccount: runnerConfig.Job.ServiceAccount.ValueString(),
			PodTemplate:    jobPodTemplate,
		},
	})
	return *runnerConfiguration, nil
}

func updateK8sAgentRunnerConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.RunnerConfigurationUpdate, error) {
	var runnerConfig KubernetesAgentRunnerConfiguration
	diags := obj.As(ctx, &runnerConfig, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return canyoncp.RunnerConfigurationUpdate{}, fmt.Errorf("failed to parse runner configuration from model: %v", diags.Errors())
	}

	var jobPodTemplate *map[string]interface{}
	if runnerConfig.Job.PodTemplate.ValueString() != "" {
		if err := json.Unmarshal([]byte(runnerConfig.Job.PodTemplate.ValueString()), &jobPodTemplate); err != nil {
			return canyoncp.RunnerConfigurationUpdate{}, fmt.Errorf("failed to parse pod template from model: %v", err)
		}
	}

	var updateRunnerConfiguration = new(canyoncp.RunnerConfigurationUpdate)
	_ = updateRunnerConfiguration.FromK8sAgentRunnerConfigurationUpdateBody(canyoncp.K8sAgentRunnerConfigurationUpdateBody{
		Key: ref.Ref(runnerConfig.Key.ValueString()),
		Job: &canyoncp.K8sRunnerJobConfig{
			Namespace:      runnerConfig.Job.Namespace.ValueString(),
			ServiceAccount: runnerConfig.Job.ServiceAccount.ValueString(),
			PodTemplate:    jobPodTemplate,
		},
	})
	return *updateRunnerConfiguration, nil
}
