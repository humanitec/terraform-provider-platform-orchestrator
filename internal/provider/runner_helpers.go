package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
	"terraform-provider-humanitec-v2/internal/ref"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func commonStateStorageConfigurationAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"type": types.StringType,
		"kubernetes_configuration": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"namespace": types.StringType,
			},
		},
		"s3": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"bucket":      types.StringType,
				"path_prefix": types.StringType,
			},
		},
	}
}

func parseStateStorageConfigurationResponse(ctx context.Context, ssc canyoncp.StateStorageConfiguration) (*basetypes.ObjectValue, error) {
	var stateStorageConfig commonRunnerStateStorageModel

	stateStorageConfig.Type, _ = ssc.Discriminator()
	switch canyoncp.StateStorageType(stateStorageConfig.Type) {
	case canyoncp.StateStorageTypeS3:
		typedSsc, _ := ssc.AsS3StorageConfiguration()
		stateStorageConfig.S3 = &commonRunnerS3StateStorageModel{
			Bucket:     typedSsc.Bucket,
			PathPrefix: ref.DerefOr(typedSsc.PathPrefix, ""),
		}
	case canyoncp.StateStorageTypeKubernetes:
		typedSsc, _ := ssc.AsK8sStorageConfiguration()
		stateStorageConfig.KubernetesConfiguration = &commonRunnerKubernetesStateStorageModel{
			Namespace: typedSsc.Namespace,
		}
	default:
		return nil, fmt.Errorf("unsupported state storage type: %s", stateStorageConfig.Type)
	}

	objectValue, diags := types.ObjectValueFrom(ctx, commonStateStorageConfigurationAttributes(), stateStorageConfig)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to build state storage configuration from model parsing API response: %v", diags.Errors())
	}
	return &objectValue, nil
}

func createStateStorageConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.StateStorageConfiguration, error) {
	var stateStorageConfig commonRunnerStateStorageModel
	diags := obj.As(ctx, &stateStorageConfig, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return canyoncp.StateStorageConfiguration{}, fmt.Errorf("failed to parse state storage configuration from model: %v", diags.Errors())
	}

	var stateStorageConfiguration = new(canyoncp.StateStorageConfiguration)
	switch canyoncp.StateStorageType(stateStorageConfig.Type) {
	case canyoncp.StateStorageTypeS3:
		_ = stateStorageConfiguration.FromS3StorageConfiguration(canyoncp.S3StorageConfiguration{
			Type:       canyoncp.StateStorageTypeS3,
			Bucket:     stateStorageConfig.S3.Bucket,
			PathPrefix: ref.RefStringEmptyNil(stateStorageConfig.S3.PathPrefix),
		})
	case canyoncp.StateStorageTypeKubernetes:
		_ = stateStorageConfiguration.FromK8sStorageConfiguration(canyoncp.K8sStorageConfiguration{
			Type:      canyoncp.StateStorageTypeKubernetes,
			Namespace: stateStorageConfig.KubernetesConfiguration.Namespace,
		})
	default:
		return canyoncp.StateStorageConfiguration{}, fmt.Errorf("unsupported state storage type: %s", stateStorageConfig.Type)
	}

	return *stateStorageConfiguration, nil
}
