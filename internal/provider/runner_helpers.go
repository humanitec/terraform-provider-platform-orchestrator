package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
	"terraform-provider-humanitec-v2/internal/ref"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type RunnerResourceModel struct {
	Id                        types.String `tfsdk:"id"`
	Description               types.String `tfsdk:"description"`
	RunnerConfiguration       types.Object `tfsdk:"runner_configuration"`
	StateStorageConfiguration types.Object `tfsdk:"state_storage_configuration"`
}

type RunnerStateStorageConfigurationModel struct {
	Type                    string                                         `tfsdk:"type"`
	KubernetesConfiguration RunnerKubernetesStateStorageConfigurationModel `tfsdk:"kubernetes_configuration"`
	S3Configuration         RunnerS3StateStorageConfigurationModel         `tfsdk:"s3_configuration"`
}

type RunnerKubernetesStateStorageConfigurationModel struct {
	Namespace string `tfsdk:"namespace"`
}

var RunnerStateStorageResourceSchema = resschema.SingleNestedAttribute{
	MarkdownDescription: "The state storage configuration for the Kubernetes Runner.",
	Required:            true,
	Attributes: map[string]resschema.Attribute{
		"type": resschema.StringAttribute{
			MarkdownDescription: "The type of state storage configuration for the Kubernetes Runner.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("kubernetes"),
			},
		},
		"kubernetes_configuration": resschema.SingleNestedAttribute{
			MarkdownDescription: "The Kubernetes state storage configuration for the Kubernetes Runner.",
			Optional:            true,
			Attributes: map[string]resschema.Attribute{
				"namespace": resschema.StringAttribute{
					MarkdownDescription: "The namespace for the Kubernetes state storage configuration.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.LengthAtMost(63),
					},
				},
			},
		},
		"s3_configuration": dsschema.SingleNestedAttribute{
			MarkdownDescription: "The S3 state storage configuration for the Kubernetes Runner",
			Optional:            true,
			Attributes: map[string]dsschema.Attribute{
				"bucket": dsschema.StringAttribute{
					MarkdownDescription: "The bucket for the S3 state storage configuration",
					Required:            true,
				},
				"prefix_path": dsschema.StringAttribute{
					MarkdownDescription: "The prefix path for the S3 state storage configuration",
					Optional:            true,
				},
			},
		},
	},
}

var RunnerStateStorageDataSourceSchema = dsschema.SingleNestedAttribute{
	MarkdownDescription: "The state storage configuration for the Kubernetes Runner",
	Computed:            true,
	Attributes: map[string]dsschema.Attribute{
		"type": dsschema.StringAttribute{
			MarkdownDescription: "The type of state storage configuration for the Kubernetes Runner",
			Computed:            true,
		},
		"kubernetes_configuration": dsschema.SingleNestedAttribute{
			MarkdownDescription: "The Kubernetes state storage configuration for the Kubernetes Runner",
			Computed:            true,
			Attributes: map[string]dsschema.Attribute{
				"namespace": dsschema.StringAttribute{
					MarkdownDescription: "The namespace for the Kubernetes state storage configuration",
					Computed:            true,
				},
			},
		},
		"s3_configuration": dsschema.SingleNestedAttribute{
			MarkdownDescription: "The S3 state storage configuration for the Kubernetes Runner",
			Computed:            true,
			Attributes: map[string]dsschema.Attribute{
				"bucket": dsschema.StringAttribute{
					MarkdownDescription: "The bucket for the S3 state storage configuration",
					Computed:            true,
				},
				"prefix_path": dsschema.StringAttribute{
					MarkdownDescription: "The prefix path for the S3 state storage configuration",
					Computed:            true,
				},
			},
		},
	},
}

var RunnerStateStorageAttributeTypes = map[string]attr.Type{
	"type": types.StringType,
	"kubernetes_configuration": types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"namespace": types.StringType,
		},
	},
	"s3_configuration": types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"bucket":      types.StringType,
			"path_prefix": types.StringType,
		},
	},
}

type RunnerS3StateStorageConfigurationModel struct {
	Bucket     string `tfsdk:"bucket"`
	PathPrefix string `tfsdk:"path_prefix"`
}

type AwsTemporaryAuth struct {
	RoleArn     types.String `tfsdk:"role_arn"`
	SessionName types.String `tfsdk:"session_name"`
	StsRegion   types.String `tfsdk:"sts_region"`
}

func NewAwsTemporaryAuth(m canyoncp.AwsTemporaryAuth) AwsTemporaryAuth {
	return AwsTemporaryAuth{
		RoleArn:     types.StringValue(m.RoleArn),
		SessionName: types.StringPointerValue(m.SessionName),
		StsRegion:   types.StringPointerValue(m.StsRegion),
	}
}

func (a AwsTemporaryAuth) ToApiModel() canyoncp.AwsTemporaryAuth {
	return canyoncp.AwsTemporaryAuth{
		RoleArn:     a.RoleArn.ValueString(),
		SessionName: fromStringValueToStringPointer(a.SessionName),
		StsRegion:   fromStringValueToStringPointer(a.StsRegion),
	}
}

func parseStateStorageConfigurationResponse(ctx context.Context, ssc canyoncp.StateStorageConfiguration) (*basetypes.ObjectValue, error) {
	discriminator, _ := ssc.Discriminator()
	stateStorageConfig := RunnerStateStorageConfigurationModel{Type: discriminator}
	switch canyoncp.StateStorageType(discriminator) {
	case canyoncp.StateStorageTypeKubernetes:
		k8sConfig, _ := ssc.AsK8sStorageConfiguration()
		stateStorageConfig.KubernetesConfiguration = RunnerKubernetesStateStorageConfigurationModel{
			Namespace: k8sConfig.Namespace,
		}
	case canyoncp.StateStorageTypeS3:
		s3Config, _ := ssc.AsS3StorageConfiguration()
		stateStorageConfig.S3Configuration = RunnerS3StateStorageConfigurationModel{
			Bucket:     s3Config.Bucket,
			PathPrefix: ref.DerefOr(s3Config.PathPrefix, ""),
		}
	default:
		return nil, fmt.Errorf("unknown state storage type: %s", discriminator)
	}
	objectValue, diags := types.ObjectValueFrom(ctx, RunnerStateStorageAttributeTypes, stateStorageConfig)
	if diags.HasError() {
		return nil, fmt.Errorf("can't parse state storage configuration from model: %v", diags.Errors())
	}
	return &objectValue, nil
}

func createStateStorageConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.StateStorageConfiguration, error) {
	var stateStorageConfiguration = new(canyoncp.StateStorageConfiguration)

	var stateStorageInner RunnerStateStorageConfigurationModel
	if diags := obj.As(ctx, &stateStorageInner, basetypes.ObjectAsOptions{}); diags.HasError() {
		return canyoncp.StateStorageConfiguration{}, fmt.Errorf("failed to parse state storage configuration from model: %v", diags.Errors())
	}

	discriminator, _ := obj.Attributes()["type"].(types.String)
	switch canyoncp.StateStorageType(discriminator.ValueString()) {
	case canyoncp.StateStorageTypeKubernetes:
		_ = stateStorageConfiguration.FromK8sStorageConfiguration(canyoncp.K8sStorageConfiguration{
			Type:      canyoncp.StateStorageType(stateStorageInner.Type),
			Namespace: stateStorageInner.KubernetesConfiguration.Namespace,
		})
	case canyoncp.StateStorageTypeS3:
		_ = stateStorageConfiguration.FromS3StorageConfiguration(canyoncp.S3StorageConfiguration{
			Type:       canyoncp.StateStorageType(stateStorageInner.Type),
			Bucket:     stateStorageInner.S3Configuration.Bucket,
			PathPrefix: ref.RefStringEmptyNil(stateStorageInner.S3Configuration.PathPrefix),
		})
	default:
		return *stateStorageConfiguration, fmt.Errorf("unknown state storage type: %s", discriminator.ValueString())
	}
	return *stateStorageConfiguration, nil
}
