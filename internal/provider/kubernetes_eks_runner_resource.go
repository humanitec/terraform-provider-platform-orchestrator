package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

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

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
)

func NewKubernetesEksRunnerResource() resource.Resource {
	return &commonRunnerResource{
		SubType: "kubernetes_eks_runner",
		SchemaDef: schema.Schema{
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
				"state_storage_configuration": commonRunnerStateStorageResourceSchema,
			},
		},
		ReadApiResponseIntoModel:         toKubernetesEksRunnerResourceModel,
		ConvertRunnerConfigIntoCreateApi: createKubernetesEksRunnerConfigurationFromObject,
		ConvertRunnerConfigIntoUpdateApi: updateKubernetesEksRunnerConfigurationFromObject,
	}
}

// KubernetesEksRunnerConfiguration describes the runner configuration structure following SecretRef pattern.
type KubernetesEksRunnerConfiguration struct {
	Cluster KubernetesEksRunnerCluster `tfsdk:"cluster"`
	Job     KubernetesEksRunnerJob     `tfsdk:"job"`
}

type KubernetesEksRunnerCluster struct {
	Name   types.String                   `tfsdk:"name"`
	Region types.String                   `tfsdk:"region"`
	Auth   KubernetesEksRunnerClusterAuth `tfsdk:"auth"`
}

type KubernetesEksRunnerClusterAuth struct {
	RoleArn     types.String `tfsdk:"role_arn"`
	SessionName types.String `tfsdk:"session_name"`
	StsRegion   types.String `tfsdk:"sts_region"`
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

type KubernetesEksRunnerStateStorageConfigurationModel struct {
	Type                    string                                                      `tfsdk:"type"`
	KubernetesConfiguration KubernetesEksRunnerKubernetesStateStorageConfigurationModel `tfsdk:"kubernetes_configuration"`
}

type KubernetesEksRunnerKubernetesStateStorageConfigurationModel struct {
	Namespace string `tfsdk:"namespace"`
}

func parseKubernetesEksRunnerConfigurationResponse(ctx context.Context, k8sEksRunnerConfiguration canyoncp.K8sEksRunnerConfiguration) (basetypes.ObjectValue, error) {
	runnerConfig := KubernetesEksRunnerConfiguration{
		Cluster: KubernetesEksRunnerCluster{
			Name:   types.StringValue(k8sEksRunnerConfiguration.Cluster.Name),
			Region: types.StringValue(k8sEksRunnerConfiguration.Cluster.Region),
			Auth: KubernetesEksRunnerClusterAuth{
				RoleArn:     types.StringValue(k8sEksRunnerConfiguration.Cluster.Auth.RoleArn),
				SessionName: types.StringPointerValue(k8sEksRunnerConfiguration.Cluster.Auth.SessionName),
				StsRegion:   types.StringPointerValue(k8sEksRunnerConfiguration.Cluster.Auth.StsRegion),
			},
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

func toKubernetesEksRunnerResourceModel(item canyoncp.Runner, _ commonRunnerModel) (commonRunnerModel, error) {
	k8sRunnerConfiguration, _ := item.RunnerConfiguration.AsK8sEksRunnerConfiguration()
	k8sStateStorageConfiguration, _ := item.StateStorageConfiguration.AsK8sStorageConfiguration()

	runnerConfigurationModel, err := parseKubernetesEksRunnerConfigurationResponse(context.Background(), k8sRunnerConfiguration)
	if err != nil {
		return commonRunnerModel{}, err
	}

	stateStorageConfigurationModel := parseStateStorageConfigurationResponse(context.Background(), k8sStateStorageConfiguration)
	if stateStorageConfigurationModel == nil {
		return commonRunnerModel{}, errors.New("failed to parse state storage configuration")
	}

	return commonRunnerModel{
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
			Auth: canyoncp.K8sRunnerAwsTemporaryAuth{
				RoleArn:     runnerConfig.Cluster.Auth.RoleArn.ValueString(),
				SessionName: fromStringValueToStringPointer(runnerConfig.Cluster.Auth.SessionName),
				StsRegion:   fromStringValueToStringPointer(runnerConfig.Cluster.Auth.StsRegion),
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
			Auth: canyoncp.K8sRunnerAwsTemporaryAuth{
				RoleArn:     runnerConfig.Cluster.Auth.RoleArn.ValueString(),
				SessionName: fromStringValueToStringPointer(runnerConfig.Cluster.Auth.SessionName),
				StsRegion:   fromStringValueToStringPointer(runnerConfig.Cluster.Auth.StsRegion),
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
