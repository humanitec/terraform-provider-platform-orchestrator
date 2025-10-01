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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KubernetesGkeRunnerResource{}
var _ resource.ResourceWithImportState = &KubernetesGkeRunnerResource{}

func NewKubernetesGkeRunnerResource() resource.Resource {
	return &KubernetesGkeRunnerResource{}
}

// KubernetesGkeRunner defines the resource implementation.
type KubernetesGkeRunnerResource struct {
	baseRunnerResource
}

// KubernetesGkeRunnerConfiguration describes the runner configuration structure following SecretRef pattern.
type KubernetesGkeRunnerConfiguration struct {
	Cluster KubernetesGkeRunnerCluster `tfsdk:"cluster"`
	Job     KubernetesGkeRunnerJob     `tfsdk:"job"`
}

type KubernetesGkeRunnerCluster struct {
	Name       types.String                   `tfsdk:"name"`
	ProjectId  types.String                   `tfsdk:"project_id"`
	ProxyUrl   types.String                   `tfsdk:"proxy_url"`
	Location   types.String                   `tfsdk:"location"`
	InternalIp types.Bool                     `tfsdk:"internal_ip"`
	Auth       KubernetesGkeRunnerClusterAuth `tfsdk:"auth"`
}

type KubernetesGkeRunnerClusterAuth struct {
	GcpAudience       types.String `tfsdk:"gcp_audience"`
	GcpServiceAccount types.String `tfsdk:"gcp_service_account"`
}

type KubernetesGkeRunnerJob struct {
	Namespace      types.String         `tfsdk:"namespace"`
	ServiceAccount types.String         `tfsdk:"service_account"`
	PodTemplate    jsontypes.Normalized `tfsdk:"pod_template"`
}

func KubernetesGkeRunnerConfigurationAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cluster": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":        types.StringType,
				"project_id":  types.StringType,
				"location":    types.StringType,
				"proxy_url":   types.StringType,
				"internal_ip": types.BoolType,
				"auth": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"gcp_audience":        types.StringType,
						"gcp_service_account": types.StringType,
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

func (r *KubernetesGkeRunnerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_gke_runner"
}

func (r *KubernetesGkeRunnerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Kubernetes GKE Runner resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the Kubernetes GKE Runner.",
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
				MarkdownDescription: "The description of the Kubernetes GKE Runner.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"runner_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration of the Kubernetes GKE cluster.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"cluster": schema.SingleNestedAttribute{
						MarkdownDescription: "The cluster configuration for the Kubernetes GKE Runner.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								MarkdownDescription: "The name of the Kubernetes GKE cluster.",
								Required:            true,
							},
							"project_id": schema.StringAttribute{
								MarkdownDescription: "The project ID where the GKE cluster is located.",
								Required:            true,
							},
							"location": schema.StringAttribute{
								MarkdownDescription: "The location of the GKE cluster.",
								Required:            true,
							},
							"proxy_url": schema.StringAttribute{
								MarkdownDescription: "The proxy URL for the Kubernetes GKE cluster.",
								Optional:            true,
							},
							"internal_ip": schema.BoolAttribute{
								MarkdownDescription: "Whether to use internal IP for the Kubernetes GKE cluster.",
								Optional:            true,
								Computed:            true,
							},
							"auth": schema.SingleNestedAttribute{
								MarkdownDescription: "The authentication configuration for the Kubernetes GKE cluster.",
								Required:            true,
								Sensitive:           true,
								Attributes: map[string]schema.Attribute{
									"gcp_audience": schema.StringAttribute{
										MarkdownDescription: "The GCP audience to authenticate to the GKE cluster.",
										Required:            true,
									},
									"gcp_service_account": schema.StringAttribute{
										MarkdownDescription: "The GCP service account to authenticate to the GKE cluster.",
										Required:            true,
									},
								},
							},
						},
					},
					"job": schema.SingleNestedAttribute{
						MarkdownDescription: "The job configuration for the Kubernetes GKE Runner.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"namespace": schema.StringAttribute{
								MarkdownDescription: "The namespace for the Kubernetes GKE Runner job.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.LengthAtMost(63),
								},
							},
							"service_account": schema.StringAttribute{
								MarkdownDescription: "The service account for the Kubernetes GKE Runner job.",
								Required:            true,
							},
							"pod_template": schema.StringAttribute{
								MarkdownDescription: "JSON encoded pod template for the Kubernetes GKE Runner job.",
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

func (r *KubernetesGkeRunnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RunnerResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	runnerConfigurationFromObject, err := createKubernetesGKERunnerConfigurationFromObject(ctx, data.RunnerConfiguration)
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

	if data, err = toKubernetesGkeRunnerResourceModel(*httpResp.JSON201, &data); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesGkeRunnerResourceModel: %s", err))
		return
	} else {
		// Save data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}

}

func (r *KubernetesGkeRunnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

	if data, err = toKubernetesGkeRunnerResourceModel(*httpResp.JSON200, &data); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesGkeRunnerResourceModel: %s", err))
		return
	} else {
		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}

}

func (r *KubernetesGkeRunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state RunnerResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateRunnerConfigurationBodyFromObject, err := updateKubernetesGkeRunnerConfigurationFromObject(ctx, data.RunnerConfiguration)
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

	if data, err = toKubernetesGkeRunnerResourceModel(*httpResp.JSON200, &data); err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert API response to KubernetesGkeRunnerResourceModel: %s", err))
		return
	} else {
		// Save data info into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}
}

func parseKubernetesGKERunnerConfigurationResponse(ctx context.Context, k8sGKERunnerConfiguration canyoncp.K8sGkeRunnerConfiguration) (basetypes.ObjectValue, error) {
	runnerConfig := KubernetesGkeRunnerConfiguration{
		Cluster: KubernetesGkeRunnerCluster{
			Name:       types.StringValue(k8sGKERunnerConfiguration.Cluster.Name),
			ProjectId:  types.StringValue(k8sGKERunnerConfiguration.Cluster.ProjectId),
			Location:   types.StringValue(k8sGKERunnerConfiguration.Cluster.Location),
			InternalIp: types.BoolValue(ref.DerefOr(k8sGKERunnerConfiguration.Cluster.InternalIp, false)),
			ProxyUrl:   types.StringPointerValue(k8sGKERunnerConfiguration.Cluster.ProxyUrl),
			Auth: KubernetesGkeRunnerClusterAuth{
				GcpAudience:       types.StringValue(k8sGKERunnerConfiguration.Cluster.Auth.GcpAudience),
				GcpServiceAccount: types.StringValue(k8sGKERunnerConfiguration.Cluster.Auth.GcpServiceAccount),
			},
		},
		Job: KubernetesGkeRunnerJob{
			Namespace:      types.StringValue(k8sGKERunnerConfiguration.Job.Namespace),
			ServiceAccount: types.StringValue(k8sGKERunnerConfiguration.Job.ServiceAccount),
		},
	}

	if k8sGKERunnerConfiguration.Job.PodTemplate != nil {
		podTemplate, _ := json.Marshal(k8sGKERunnerConfiguration.Job.PodTemplate)
		runnerConfig.Job.PodTemplate = jsontypes.NewNormalizedValue(string(podTemplate))
	} else {
		runnerConfig.Job.PodTemplate = jsontypes.NewNormalizedNull()
	}

	objectValue, diags := types.ObjectValueFrom(ctx, KubernetesGkeRunnerConfigurationAttributeTypes(), runnerConfig)
	if diags.HasError() {
		return basetypes.ObjectValue{}, fmt.Errorf("failed to build runner configuration from model parsing API response: %v", diags.Errors())
	}
	return objectValue, nil
}

func toKubernetesGkeRunnerResourceModel(item canyoncp.Runner, data *RunnerResourceModel) (RunnerResourceModel, error) {
	k8sRunnerConfiguration, _ := item.RunnerConfiguration.AsK8sGkeRunnerConfiguration()

	runnerConfigurationModel, err := parseKubernetesGKERunnerConfigurationResponse(context.Background(), k8sRunnerConfiguration)
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

func createKubernetesGKERunnerConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.RunnerConfiguration, error) {
	var runnerConfig KubernetesGkeRunnerConfiguration
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
	_ = runnerConfiguration.FromK8sGkeRunnerConfiguration(canyoncp.K8sGkeRunnerConfiguration{
		Cluster: canyoncp.K8sRunnerGkeCluster{
			InternalIp: ref.Ref(runnerConfig.Cluster.InternalIp.ValueBool()),
			Name:       runnerConfig.Cluster.Name.ValueString(),
			ProjectId:  runnerConfig.Cluster.ProjectId.ValueString(),
			Location:   runnerConfig.Cluster.Location.ValueString(),
			ProxyUrl:   fromStringValueToStringPointer(runnerConfig.Cluster.ProxyUrl),
			Auth: canyoncp.K8sRunnerGcpTemporaryAuth{
				GcpAudience:       runnerConfig.Cluster.Auth.GcpAudience.ValueString(),
				GcpServiceAccount: runnerConfig.Cluster.Auth.GcpServiceAccount.ValueString(),
			},
		},
		Job: canyoncp.K8sRunnerJobConfig{
			Namespace:      runnerConfig.Job.Namespace.ValueString(),
			ServiceAccount: runnerConfig.Job.ServiceAccount.ValueString(),
			PodTemplate:    jobPodTemplate,
		},
	})
	return *runnerConfiguration, nil
}

func updateKubernetesGkeRunnerConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.RunnerConfigurationUpdate, error) {
	var runnerConfig KubernetesGkeRunnerConfiguration
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
	_ = updateRunnerConfiguration.FromK8sGkeRunnerConfigurationUpdateBody(canyoncp.K8sGkeRunnerConfigurationUpdateBody{
		Cluster: &canyoncp.K8sRunnerGkeCluster{
			Name:       runnerConfig.Cluster.Name.ValueString(),
			ProjectId:  runnerConfig.Cluster.ProjectId.ValueString(),
			Location:   runnerConfig.Cluster.Location.ValueString(),
			ProxyUrl:   fromStringValueToStringPointer(runnerConfig.Cluster.ProxyUrl),
			InternalIp: ref.Ref(runnerConfig.Cluster.InternalIp.ValueBool()),
			Auth: canyoncp.K8sRunnerGcpTemporaryAuth{
				GcpAudience:       runnerConfig.Cluster.Auth.GcpAudience.ValueString(),
				GcpServiceAccount: runnerConfig.Cluster.Auth.GcpServiceAccount.ValueString(),
			},
		},
		Job: &canyoncp.K8sRunnerJobConfig{
			Namespace:      runnerConfig.Job.Namespace.ValueString(),
			ServiceAccount: runnerConfig.Job.ServiceAccount.ValueString(),
			PodTemplate:    jobPodTemplate,
		},
	})
	return *updateRunnerConfiguration, nil
}
