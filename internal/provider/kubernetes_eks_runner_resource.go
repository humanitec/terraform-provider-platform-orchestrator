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
var _ resource.Resource = &KubernetesEksRunnerResource{}
var _ resource.ResourceWithImportState = &KubernetesEksRunnerResource{}

func NewKubernetesEksRunnerResource() resource.Resource {
	return &KubernetesEksRunnerResource{}
}

// KubernetesEksRunner defines the resource implementation.
type KubernetesEksRunnerResource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// KubernetesEksRunnerConfiguration describes the runner configuration structure following SecretRef pattern.
type KubernetesEksRunnerConfiguration struct {
	Cluster KubernetesEksRunnerCluster `tfsdk:"cluster"`
	Job     KubernetesEksRunnerJob     `tfsdk:"job"`
}

type KubernetesEksRunnerCluster struct {
	Name   types.String     `tfsdk:"name"`
	Region types.String     `tfsdk:"region"`
	Auth   AwsTemporaryAuth `tfsdk:"auth"`
}

type KubernetesEksRunnerJob struct {
	Namespace      types.String         `tfsdk:"namespace"`
	ServiceAccount types.String         `tfsdk:"service_account"`
	PodTemplate    jsontypes.Normalized `tfsdk:"pod_template"`
}

func KubernetesEksRunnerConfigurationAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cluster": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":   types.StringType,
				"region": types.StringType,
				"auth": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"role_arn":     types.StringType,
						"session_name": types.StringType,
						"sts_region":   types.StringType,
					},
				},
			},
		},
		"job": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"namespace":       types.StringType,
				"service_account": types.StringType,
				"pod_template":    types.StringType,
			},
		},
	}
}

func (r *KubernetesEksRunnerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_eks_runner"
}

func (r *KubernetesEksRunnerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Kubernetes EKS Runner resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the Kubernetes EKS Runner.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
						"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
					),
					stringvalidator.LengthAtMost(100),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Kubernetes EKS Runner.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"runner_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration of the Kubernetes EKS cluster.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"cluster": schema.SingleNestedAttribute{
						MarkdownDescription: "The cluster configuration for the Kubernetes EKS Runner.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								MarkdownDescription: "The name of the Kubernetes EKS cluster.",
								Required:            true,
							},
							"region": schema.StringAttribute{
								MarkdownDescription: "The AWS region where the EKS cluster is located.",
								Required:            true,
							},
							"auth": schema.SingleNestedAttribute{
								MarkdownDescription: "Configuration to obtain temporary AWS security credentials by assuming an IAM role.",
								Required:            true,
								Attributes: map[string]schema.Attribute{
									"role_arn": schema.StringAttribute{
										MarkdownDescription: "The ARN of the role to assume.",
										Required:            true,
										Validators: []validator.String{
											stringvalidator.RegexMatches(
												regexp.MustCompile(`^arn:aws:iam::[0-9]{12}:role\/[a-zA-Z_0-9+=,.@\-_/]+$`),
												"must be a valid IAM Role ARN",
											),
										},
									},
									"session_name": schema.StringAttribute{
										MarkdownDescription: "Session name to be used when assuming the role. If not provided, a default session name will be \"{org_id}-{runner_id}\".",
										Optional:            true,
										Validators: []validator.String{
											stringvalidator.LengthBetween(3, 64),
											stringvalidator.RegexMatches(
												regexp.MustCompile(`^[a-zA-Z0-9+=,.@\-_/]+$`),
												"must contain only valid characters (letters, digits, and +=,.@-_/)",
											),
										},
									},
									"sts_region": schema.StringAttribute{
										MarkdownDescription: "The AWS region identifier for the Security Token Service (STS) endpoint. If not provided, the cluster region will be used.",
										Optional:            true,
										Validators: []validator.String{
											stringvalidator.RegexMatches(
												regexp.MustCompile(`^[a-z]{2}-[a-z]+-\d$`),
												"must be a valid AWS region",
											),
										},
									},
								},
							},
						},
					},
					"job": schema.SingleNestedAttribute{
						MarkdownDescription: "The job configuration for the Kubernetes EKS Runner.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"namespace": schema.StringAttribute{
								MarkdownDescription: "The namespace for the Kubernetes EKS Runner job.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.LengthAtMost(63),
								},
							},
							"service_account": schema.StringAttribute{
								MarkdownDescription: "The service account for the Kubernetes EKS Runner job.",
								Required:            true,
							},
							"pod_template": schema.StringAttribute{
								MarkdownDescription: "JSON encoded pod template for the Kubernetes EKS Runner job.",
								Optional:            true,
								CustomType:          jsontypes.NormalizedType{},
								Computed:            true,
							},
						},
					},
				},
			},
			"state_storage_configuration": RunnerStateStorageResourceSchema,
		},
	}
}

func (r *KubernetesEksRunnerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KubernetesEksRunnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RunnerResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	runnerConfigurationFromObject, err := createKubernetesEksRunnerConfigurationFromObject(ctx, data.RunnerConfiguration)
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

	if data, err = toKubernetesEksRunnerResourceModel(*httpResp.JSON201); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesGkeRunnerResourceModel: %s", err))
		return
	} else {
		// Save data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}

}

func (r *KubernetesEksRunnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RunnerResourceModel

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

	if httpResp.StatusCode() != 200 {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read runner, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	if data, err = toKubernetesEksRunnerResourceModel(*httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesGkeRunnerResourceModel: %s", err))
		return
	} else {
		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}

}

func (r *KubernetesEksRunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state RunnerResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateRunnerConfigurationBodyFromObject, err := updateKubernetesEksRunnerConfigurationFromObject(ctx, data.RunnerConfiguration)
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

	if data, err = toKubernetesEksRunnerResourceModel(*httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesGkeRunnerResourceModel: %s", err))
		return
	} else {
		// Save data info into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}
}

func (r *KubernetesEksRunnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RunnerResourceModel

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

func (r *KubernetesEksRunnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func parseKubernetesEksRunnerConfigurationResponse(ctx context.Context, k8sEksRunnerConfiguration canyoncp.K8sEksRunnerConfiguration) (basetypes.ObjectValue, error) {
	runnerConfig := KubernetesEksRunnerConfiguration{
		Cluster: KubernetesEksRunnerCluster{
			Name:   types.StringValue(k8sEksRunnerConfiguration.Cluster.Name),
			Region: types.StringValue(k8sEksRunnerConfiguration.Cluster.Region),
			Auth:   NewAwsTemporaryAuth(k8sEksRunnerConfiguration.Cluster.Auth),
		},
		Job: KubernetesEksRunnerJob{
			Namespace:      types.StringValue(k8sEksRunnerConfiguration.Job.Namespace),
			ServiceAccount: types.StringValue(k8sEksRunnerConfiguration.Job.ServiceAccount),
		},
	}

	if k8sEksRunnerConfiguration.Job.PodTemplate != nil {
		podTemplate, _ := json.Marshal(k8sEksRunnerConfiguration.Job.PodTemplate)
		runnerConfig.Job.PodTemplate = jsontypes.NewNormalizedValue(string(podTemplate))
	} else {
		runnerConfig.Job.PodTemplate = jsontypes.NewNormalizedNull()
	}

	objectValue, diags := types.ObjectValueFrom(ctx, KubernetesEksRunnerConfigurationAttributeTypes(), runnerConfig)
	if diags.HasError() {
		return basetypes.ObjectValue{}, fmt.Errorf("failed to build runner configuration from model parsing API response: %v", diags.Errors())
	}
	return objectValue, nil
}

func toKubernetesEksRunnerResourceModel(item canyoncp.Runner) (RunnerResourceModel, error) {
	k8sRunnerConfiguration, _ := item.RunnerConfiguration.AsK8sEksRunnerConfiguration()

	runnerConfigurationModel, err := parseKubernetesEksRunnerConfigurationResponse(context.Background(), k8sRunnerConfiguration)
	if err != nil {
		return RunnerResourceModel{}, err
	}

	stateStorageConfigurationModel, err := parseStateStorageConfigurationResponse(context.Background(), item.StateStorageConfiguration)
	if err != nil {
		return RunnerResourceModel{}, err
	}

	return RunnerResourceModel{
		Id:                        types.StringValue(item.Id),
		Description:               types.StringPointerValue(item.Description),
		StateStorageConfiguration: *stateStorageConfigurationModel,
		RunnerConfiguration:       runnerConfigurationModel,
	}, nil
}

func createKubernetesEksRunnerConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.RunnerConfiguration, error) {
	var runnerConfig KubernetesEksRunnerConfiguration
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
	_ = runnerConfiguration.FromK8sEksRunnerConfiguration(canyoncp.K8sEksRunnerConfiguration{
		Cluster: canyoncp.K8sRunnerEksCluster{
			Name:   runnerConfig.Cluster.Name.ValueString(),
			Region: runnerConfig.Cluster.Region.ValueString(),
			Auth:   runnerConfig.Cluster.Auth.ToApiModel(),
		},
		Job: canyoncp.K8sRunnerJobConfig{
			Namespace:      runnerConfig.Job.Namespace.ValueString(),
			ServiceAccount: runnerConfig.Job.ServiceAccount.ValueString(),
			PodTemplate:    jobPodTemplate,
		},
	})
	return *runnerConfiguration, nil
}

func updateKubernetesEksRunnerConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.RunnerConfigurationUpdate, error) {
	var runnerConfig KubernetesEksRunnerConfiguration
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
	_ = updateRunnerConfiguration.FromK8sEksRunnerConfigurationUpdateBody(canyoncp.K8sEksRunnerConfigurationUpdateBody{
		Cluster: &canyoncp.K8sRunnerEksCluster{
			Name:   runnerConfig.Cluster.Name.ValueString(),
			Region: runnerConfig.Cluster.Region.ValueString(),
			Auth:   runnerConfig.Cluster.Auth.ToApiModel(),
		},
		Job: &canyoncp.K8sRunnerJobConfig{
			Namespace:      runnerConfig.Job.Namespace.ValueString(),
			ServiceAccount: runnerConfig.Job.ServiceAccount.ValueString(),
			PodTemplate:    jobPodTemplate,
		},
	})
	return *updateRunnerConfiguration, nil
}
