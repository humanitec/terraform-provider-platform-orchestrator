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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func NewKubernetesGkeRunnerResource() resource.Resource {
	return &commonRunnerResource{
		SubType: "kubernetes_gke_runner",
		SchemaDef: schema.Schema{
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
				"state_storage_configuration": commonRunnerStateStorageResourceSchema,
			},
		},
		ReadApiResponseIntoModel: func(runner canyoncp.Runner, model commonRunnerModel) (commonRunnerModel, error) {
			x, err := toKubernetesGkeRunnerResourceModel(runner)
			return commonRunnerModel(x), err
		},
		ConvertRunnerConfigIntoCreateApi: func(ctx context.Context, obj types.Object) (canyoncp.RunnerConfiguration, error) {
			x, err := createKubernetesGKERunnerConfigurationFromObject(ctx, obj)
			return x, err
		},
		ConvertRunnerConfigIntoUpdateApi: func(ctx context.Context, obj types.Object) (canyoncp.RunnerConfigurationUpdate, error) {
			x, err := updateKubernetesGkeRunnerConfigurationFromObject(ctx, obj)
			return x, err
		},
	}
}

// KubernetesGkeRunnerModel describes the resource data model.
type KubernetesGkeRunnerResourceModel struct {
	Id                        types.String `tfsdk:"id"`
	Description               types.String `tfsdk:"description"`
	RunnerConfiguration       types.Object `tfsdk:"runner_configuration"`
	StateStorageConfiguration types.Object `tfsdk:"state_storage_configuration"`
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

type KubernetesGkeRunnerStateStorageConfigurationModel struct {
	Type                    string                                                      `tfsdk:"type"`
	KubernetesConfiguration KubernetesGkeRunnerKubernetesStateStorageConfigurationModel `tfsdk:"kubernetes_configuration"`
}

type KubernetesGkeRunnerKubernetesStateStorageConfigurationModel struct {
	Namespace string `tfsdk:"namespace"`
}

func KubernetesGkeRunnerStateStorageConfigurationAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type": types.StringType,
		"kubernetes_configuration": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"namespace": types.StringType,
			},
		},
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

func toKubernetesGkeRunnerResourceModel(item canyoncp.Runner) (KubernetesGkeRunnerResourceModel, error) {
	k8sRunnerConfiguration, _ := item.RunnerConfiguration.AsK8sGkeRunnerConfiguration()
	k8sStateStorageConfiguration, _ := item.StateStorageConfiguration.AsK8sStorageConfiguration()

	runnerConfigurationModel, err := parseKubernetesGKERunnerConfigurationResponse(context.Background(), k8sRunnerConfiguration)
	if err != nil {
		return KubernetesGkeRunnerResourceModel{}, err
	}

	stateStorageConfigurationModel := parseStateStorageConfigurationResponse(context.Background(), k8sStateStorageConfiguration)
	if stateStorageConfigurationModel == nil {
		return KubernetesGkeRunnerResourceModel{}, errors.New("failed to parse state storage configuration")
	}

	return KubernetesGkeRunnerResourceModel{
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
