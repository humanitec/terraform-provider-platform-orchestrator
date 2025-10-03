package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
)

func parseStateStorageConfigurationResponse(ctx context.Context, ssc canyoncp.StateStorageConfiguration) (*basetypes.ObjectValue, error) {
	var stateStorageConfig commonRunnerStateStorageModel

	stateStorageConfig.Type, _ = ssc.Discriminator()
	switch canyoncp.StateStorageType(stateStorageConfig.Type) {
	case canyoncp.StateStorageTypeS3:
		typedSsc, _ := ssc.AsS3StorageConfiguration()
		stateStorageConfig.S3Configuration = &commonRunnerS3StateStorageModel{
			Bucket:     typedSsc.Bucket,
			PathPrefix: typedSsc.PathPrefix,
		}
	case canyoncp.StateStorageTypeKubernetes:
		typedSsc, _ := ssc.AsK8sStorageConfiguration()
		stateStorageConfig.KubernetesConfiguration = &commonRunnerKubernetesStateStorageModel{
			Namespace: typedSsc.Namespace,
		}
	default:
		return nil, fmt.Errorf("unsupported state storage type: %s", stateStorageConfig.Type)
	}

	attrs, err := AttributeTypesFromResourceSchema(commonRunnerStateStorageResourceSchema.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to build attributes: %v", err)
	}

	objectValue, diags := types.ObjectValueFrom(ctx, attrs, stateStorageConfig)
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
		if stateStorageConfig.S3Configuration == nil {
			return canyoncp.StateStorageConfiguration{}, fmt.Errorf("s3 configuration in object is not set")
		}
		_ = stateStorageConfiguration.FromS3StorageConfiguration(canyoncp.S3StorageConfiguration{
			Type:       canyoncp.StateStorageTypeS3,
			Bucket:     stateStorageConfig.S3Configuration.Bucket,
			PathPrefix: stateStorageConfig.S3Configuration.PathPrefix,
		})
	case canyoncp.StateStorageTypeKubernetes:
		if stateStorageConfig.KubernetesConfiguration == nil {
			return canyoncp.StateStorageConfiguration{}, fmt.Errorf("k8s configuration in object is not set")
		}
		_ = stateStorageConfiguration.FromK8sStorageConfiguration(canyoncp.K8sStorageConfiguration{
			Type:      canyoncp.StateStorageTypeKubernetes,
			Namespace: stateStorageConfig.KubernetesConfiguration.Namespace,
		})
	default:
		return canyoncp.StateStorageConfiguration{}, fmt.Errorf("unsupported state storage type: %s", stateStorageConfig.Type)
	}

	return *stateStorageConfiguration, nil
}
