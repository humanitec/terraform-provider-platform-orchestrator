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

func NewKubernetesRunnerResource() resource.Resource {
	return &commonRunnerResource{
		SubType: "kubernetes_runner",
		SchemaDef: schema.Schema{
			// This description is used by the documentation generator and the language server.
			MarkdownDescription: "Kubernetes Runner resource",

			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "The unique identifier for the Kubernetes Runner.",
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
					MarkdownDescription: "The description of the Kubernetes Runner cluster.",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.LengthAtMost(200),
					},
				},
				"runner_configuration": schema.SingleNestedAttribute{
					MarkdownDescription: "The configuration of the Kubernetes Runner cluster.",
					Required:            true,
					Attributes: map[string]schema.Attribute{
						"cluster": schema.SingleNestedAttribute{
							MarkdownDescription: "The cluster configuration for the Kubernetes Runner cluster.",
							Required:            true,
							Attributes: map[string]schema.Attribute{
								"cluster_data": schema.SingleNestedAttribute{
									MarkdownDescription: "The cluster data for the Kubernetes Runner cluster.",
									Required:            true,
									Attributes: map[string]schema.Attribute{
										"certificate_authority_data": schema.StringAttribute{
											MarkdownDescription: "The certificate authority data for the Kubernetes Runner cluster.",
											Required:            true,
										},
										"server": schema.StringAttribute{
											MarkdownDescription: "The server URL for the Kubernetes Runner cluster.",
											Required:            true,
										},
										"proxy_url": schema.StringAttribute{
											MarkdownDescription: "The proxy URL for the Kubernetes Runner cluster.",
											Optional:            true,
										},
									},
								},
								"auth": schema.SingleNestedAttribute{
									MarkdownDescription: "The authentication configuration for the Kubernetes Runner cluster.",
									Required:            true,
									Attributes: map[string]schema.Attribute{
										"client_certificate_data": schema.StringAttribute{
											MarkdownDescription: "The client certificate data for the Kubernetes Runner cluster.",
											Optional:            true,
											Sensitive:           true,
										},
										"client_key_data": schema.StringAttribute{
											MarkdownDescription: "The client key data for the Kubernetes Runner cluster.",
											Optional:            true,
											Sensitive:           true,
										},
										"service_account_token": schema.StringAttribute{
											MarkdownDescription: "The service account token for the Kubernetes Runner cluster.",
											Optional:            true,
											Sensitive:           true,
										},
									},
								},
							},
						},
						"job": schema.SingleNestedAttribute{
							MarkdownDescription: "The job configuration for the Kubernetes Runner.",
							Required:            true,
							Attributes: map[string]schema.Attribute{
								"namespace": schema.StringAttribute{
									MarkdownDescription: "The namespace for the Kubernetes Runner job.",
									Required:            true,
									Validators: []validator.String{
										stringvalidator.LengthAtMost(63),
									},
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
				"state_storage_configuration": commonRunnerStateStorageResourceSchema,
			},
		},
		ReadApiResponseIntoModel:         toKubernetesRunnerResourceModel,
		ConvertRunnerConfigIntoCreateApi: createKubernetesRunnerConfigurationFromObject,
		ConvertRunnerConfigIntoUpdateApi: updateKubernetesRunnerConfigurationFromObject,
	}
}

// KubernetesRunnerConfiguration describes the runner configuration structure following SecretRef pattern.
type KubernetesRunnerConfiguration struct {
	Cluster KubernetesRunnerCluster `tfsdk:"cluster"`
	Job     KubernetesRunnerJob     `tfsdk:"job"`
}

type KubernetesRunnerCluster struct {
	ClusterData KubernetesRunnerClusterData `tfsdk:"cluster_data"`
	Auth        KubernetesRunnerClusterAuth `tfsdk:"auth"`
}

type KubernetesRunnerClusterData struct {
	CertificateAuthorityData types.String `tfsdk:"certificate_authority_data"`
	Server                   types.String `tfsdk:"server"`
	ProxyUrl                 types.String `tfsdk:"proxy_url"`
}

type KubernetesRunnerClusterAuth struct {
	ClientCertificateData types.String `tfsdk:"client_certificate_data"`
	ClientKeyData         types.String `tfsdk:"client_key_data"`
	ServiceAccountToken   types.String `tfsdk:"service_account_token"`
}

type KubernetesRunnerJob struct {
	Namespace      types.String         `tfsdk:"namespace"`
	ServiceAccount types.String         `tfsdk:"service_account"`
	PodTemplate    jsontypes.Normalized `tfsdk:"pod_template"`
}

func KubernetesRunnerConfigurationAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cluster": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"cluster_data": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"certificate_authority_data": types.StringType,
						"server":                     types.StringType,
						"proxy_url":                  types.StringType,
					},
				},
				"auth": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"client_certificate_data": types.StringType,
						"client_key_data":         types.StringType,
						"service_account_token":   types.StringType,
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

type KubernetesRunnerStateStorageConfigurationModel struct {
	Type                    string                                                   `tfsdk:"type"`
	KubernetesConfiguration KubernetesRunnerKubernetesStateStorageConfigurationModel `tfsdk:"kubernetes_configuration"`
}

type KubernetesRunnerKubernetesStateStorageConfigurationModel struct {
	Namespace string `tfsdk:"namespace"`
}

func parseKubernetesRunnerConfigurationResponse(ctx context.Context, k8sRunnerConfiguration canyoncp.K8sRunnerConfiguration, data *commonRunnerModel) (basetypes.ObjectValue, error) {
	var runnerConfig KubernetesRunnerConfiguration
	if data.RunnerConfiguration.IsUnknown() || data.RunnerConfiguration.IsNull() {
		runnerConfig = KubernetesRunnerConfiguration{}
	} else {
		diags := data.RunnerConfiguration.As(ctx, &runnerConfig, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return basetypes.ObjectValue{}, fmt.Errorf("failed to build runner configuration from model parsing API response: %v", diags.Errors())
		}
	}

	// Update cluster data from API response
	runnerConfig.Cluster.ClusterData.CertificateAuthorityData = types.StringValue(k8sRunnerConfiguration.Cluster.ClusterData.CertificateAuthorityData)
	runnerConfig.Cluster.ClusterData.Server = types.StringValue(k8sRunnerConfiguration.Cluster.ClusterData.Server)
	runnerConfig.Cluster.ClusterData.ProxyUrl = types.StringPointerValue(k8sRunnerConfiguration.Cluster.ClusterData.ProxyUrl)

	// Handle auth fields: these are sensitive so preserve the user's configuration unless they are unknown
	if runnerConfig.Cluster.Auth.ClientCertificateData.IsUnknown() || runnerConfig.Cluster.Auth.ClientCertificateData.IsNull() {
		if k8sRunnerConfiguration.Cluster.Auth.ClientCertificateData != nil {
			runnerConfig.Cluster.Auth.ClientCertificateData = types.StringValue(*k8sRunnerConfiguration.Cluster.Auth.ClientCertificateData)
		} else {
			runnerConfig.Cluster.Auth.ClientCertificateData = types.StringNull()
		}
	}

	if runnerConfig.Cluster.Auth.ClientKeyData.IsUnknown() || runnerConfig.Cluster.Auth.ClientKeyData.IsNull() {
		if k8sRunnerConfiguration.Cluster.Auth.ClientKeyData != nil {
			runnerConfig.Cluster.Auth.ClientKeyData = types.StringValue(*k8sRunnerConfiguration.Cluster.Auth.ClientKeyData)
		} else {
			runnerConfig.Cluster.Auth.ClientKeyData = types.StringNull()
		}
	}

	if runnerConfig.Cluster.Auth.ServiceAccountToken.IsUnknown() || runnerConfig.Cluster.Auth.ServiceAccountToken.IsNull() {
		if k8sRunnerConfiguration.Cluster.Auth.ServiceAccountToken != nil {
			runnerConfig.Cluster.Auth.ServiceAccountToken = types.StringValue(*k8sRunnerConfiguration.Cluster.Auth.ServiceAccountToken)
		} else {
			runnerConfig.Cluster.Auth.ServiceAccountToken = types.StringNull()
		}
	}

	// Update job config from API response
	runnerConfig.Job.Namespace = types.StringValue(k8sRunnerConfiguration.Job.Namespace)
	runnerConfig.Job.ServiceAccount = types.StringValue(k8sRunnerConfiguration.Job.ServiceAccount)
	if k8sRunnerConfiguration.Job.PodTemplate != nil {
		podTemplate, _ := json.Marshal(k8sRunnerConfiguration.Job.PodTemplate)
		runnerConfig.Job.PodTemplate = jsontypes.NewNormalizedValue(string(podTemplate))
	} else {
		runnerConfig.Job.PodTemplate = jsontypes.NewNormalizedNull()
	}

	objectValue, diags := types.ObjectValueFrom(ctx, KubernetesRunnerConfigurationAttributeTypes(), runnerConfig)
	if diags.HasError() {
		return basetypes.ObjectValue{}, fmt.Errorf("failed to build runner configuration from model parsing API response: %v", diags.Errors())
	}
	return objectValue, nil
}

func toKubernetesRunnerResourceModel(item canyoncp.Runner, data commonRunnerModel) (commonRunnerModel, error) {
	k8sRunnerConfiguration, _ := item.RunnerConfiguration.AsK8sRunnerConfiguration()
	k8sStateStorageConfiguration, _ := item.StateStorageConfiguration.AsK8sStorageConfiguration()

	runnerConfigurationModel, err := parseKubernetesRunnerConfigurationResponse(context.Background(), k8sRunnerConfiguration, &data)
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

func createKubernetesRunnerConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.RunnerConfiguration, error) {
	var runnerConfig KubernetesRunnerConfiguration
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
	_ = runnerConfiguration.FromK8sRunnerConfiguration(canyoncp.K8sRunnerConfiguration{
		Cluster: canyoncp.K8sRunnerK8sCluster{
			ClusterData: canyoncp.K8sRunnerK8sClusterClusterData{
				CertificateAuthorityData: runnerConfig.Cluster.ClusterData.CertificateAuthorityData.ValueString(),
				Server:                   runnerConfig.Cluster.ClusterData.Server.ValueString(),
				ProxyUrl:                 fromStringValueToStringPointer(runnerConfig.Cluster.ClusterData.ProxyUrl),
			},
			Auth: canyoncp.K8sRunnerK8sClusterAuth{
				ClientCertificateData: fromStringValueToStringPointer(runnerConfig.Cluster.Auth.ClientCertificateData),
				ClientKeyData:         fromStringValueToStringPointer(runnerConfig.Cluster.Auth.ClientKeyData),
				ServiceAccountToken:   fromStringValueToStringPointer(runnerConfig.Cluster.Auth.ServiceAccountToken),
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

func updateKubernetesRunnerConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.RunnerConfigurationUpdate, error) {
	var runnerConfig KubernetesRunnerConfiguration
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
	_ = updateRunnerConfiguration.FromK8sRunnerConfigurationUpdateBody(canyoncp.K8sRunnerConfigurationUpdateBody{
		Cluster: &canyoncp.K8sRunnerK8sCluster{
			ClusterData: canyoncp.K8sRunnerK8sClusterClusterData{
				CertificateAuthorityData: runnerConfig.Cluster.ClusterData.CertificateAuthorityData.ValueString(),
				Server:                   runnerConfig.Cluster.ClusterData.Server.ValueString(),
				ProxyUrl:                 fromStringValueToStringPointer(runnerConfig.Cluster.ClusterData.ProxyUrl),
			},
			Auth: canyoncp.K8sRunnerK8sClusterAuth{
				ClientCertificateData: fromStringValueToStringPointer(runnerConfig.Cluster.Auth.ClientCertificateData),
				ClientKeyData:         fromStringValueToStringPointer(runnerConfig.Cluster.Auth.ClientKeyData),
				ServiceAccountToken:   fromStringValueToStringPointer(runnerConfig.Cluster.Auth.ServiceAccountToken),
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
