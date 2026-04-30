package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	cp "terraform-provider-platform-orchestrator/internal/clients/platform-orchestrator-cp"
)

func parseStateStorageConfigurationResponse[T any](ctx context.Context, ssc cp.StateStorageConfiguration, schemaAttrs map[string]schema.Attribute, buildModel func(cp.StateStorageConfiguration) (T, error)) (*basetypes.ObjectValue, error) {
	model, err := buildModel(ssc)
	if err != nil {
		return nil, err
	}

	attrs, err := AttributeTypesFromResourceSchema(schemaAttrs)
	if err != nil {
		return nil, fmt.Errorf("failed to build attributes: %v", err)
	}

	objectValue, diags := types.ObjectValueFrom(ctx, attrs, model)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to build state storage configuration from model parsing API response: %v", diags.Errors())
	}
	return &objectValue, nil
}

func buildCommonStateStorageModel(ssc cp.StateStorageConfiguration) (commonRunnerStateStorageModel, error) {
	var model commonRunnerStateStorageModel
	model.Type, _ = ssc.Discriminator()
	switch cp.StateStorageType(model.Type) {
	case cp.StateStorageTypeS3:
		typedSsc, _ := ssc.AsS3StorageConfiguration()
		model.S3Configuration = &commonRunnerS3StateStorageModel{
			Bucket:     typedSsc.Bucket,
			PathPrefix: typedSsc.PathPrefix,
		}
	case cp.StateStorageTypeKubernetes:
		typedSsc, _ := ssc.AsK8sStorageConfiguration()
		model.KubernetesConfiguration = &commonRunnerKubernetesStateStorageModel{
			Namespace: typedSsc.Namespace,
		}
	case cp.StateStorageTypeGcs:
		typedSsc, _ := ssc.AsGCSStorageConfiguration()
		model.GCSConfiguration = &commonRunnerGCSStateStorageModel{
			Bucket:     typedSsc.Bucket,
			PathPrefix: typedSsc.PathPrefix,
		}
	case cp.StateStorageTypeAzurerm:
		typedSsc, _ := ssc.AsAzureRMStorageConfiguration()
		model.AzureRMConfiguration = &commonRunnerAzureRMStateStorageModel{
			ResourceGroupName:  typedSsc.ResourceGroupName,
			StorageAccountName: typedSsc.StorageAccountName,
			ContainerName:      typedSsc.ContainerName,
			PathPrefix:         typedSsc.PathPrefix,
		}
	default:
		return model, fmt.Errorf("unsupported state storage type: %s", model.Type)
	}
	return model, nil
}

func createStateStorageConfigurationFromObject(ctx context.Context, obj types.Object) (cp.StateStorageConfiguration, error) {
	storageTypeAttr, ok := obj.Attributes()["type"].(types.String)
	if !ok {
		return cp.StateStorageConfiguration{}, fmt.Errorf("type attribute is not set or has unexpected type")
	}
	storageType := storageTypeAttr.ValueString()

	var stateStorageConfiguration = new(cp.StateStorageConfiguration)
	switch cp.StateStorageType(storageType) {
	case cp.StateStorageTypeS3:
		s3Obj, ok := obj.Attributes()["s3_configuration"].(types.Object)
		if !ok || s3Obj.IsNull() {
			return cp.StateStorageConfiguration{}, fmt.Errorf("s3 configuration in object is not set")
		}
		var s3Config commonRunnerS3StateStorageModel
		diags := s3Obj.As(ctx, &s3Config, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return cp.StateStorageConfiguration{}, fmt.Errorf("failed to parse s3 configuration: %v", diags.Errors())
		}
		_ = stateStorageConfiguration.FromS3StorageConfiguration(cp.S3StorageConfiguration{
			Type:       cp.StateStorageTypeS3,
			Bucket:     s3Config.Bucket,
			PathPrefix: s3Config.PathPrefix,
		})

	case cp.StateStorageTypeKubernetes:
		k8sObj, ok := obj.Attributes()["kubernetes_configuration"].(types.Object)
		if !ok || k8sObj.IsNull() {
			return cp.StateStorageConfiguration{}, fmt.Errorf("k8s configuration in object is not set")
		}
		var k8sConfig commonRunnerKubernetesStateStorageModel
		diags := k8sObj.As(ctx, &k8sConfig, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return cp.StateStorageConfiguration{}, fmt.Errorf("failed to parse kubernetes configuration: %v", diags.Errors())
		}
		_ = stateStorageConfiguration.FromK8sStorageConfiguration(cp.K8sStorageConfiguration{
			Type:      cp.StateStorageTypeKubernetes,
			Namespace: k8sConfig.Namespace,
		})

	case cp.StateStorageTypeGcs:
		gcsObj, ok := obj.Attributes()["gcs_configuration"].(types.Object)
		if !ok || gcsObj.IsNull() {
			return cp.StateStorageConfiguration{}, fmt.Errorf("gcs configuration in object is not set")
		}
		var gcsConfig commonRunnerGCSStateStorageModel
		diags := gcsObj.As(ctx, &gcsConfig, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return cp.StateStorageConfiguration{}, fmt.Errorf("failed to parse gcs configuration: %v", diags.Errors())
		}
		_ = stateStorageConfiguration.FromGCSStorageConfiguration(cp.GCSStorageConfiguration{
			Type:       cp.StateStorageTypeGcs,
			Bucket:     gcsConfig.Bucket,
			PathPrefix: gcsConfig.PathPrefix,
		})

	case cp.StateStorageTypeAzurerm:
		azurermObj, ok := obj.Attributes()["azurerm_configuration"].(types.Object)
		if !ok || azurermObj.IsNull() {
			return cp.StateStorageConfiguration{}, fmt.Errorf("azurerm configuration in object is not set")
		}
		var azurermConfig commonRunnerAzureRMStateStorageModel
		diags := azurermObj.As(ctx, &azurermConfig, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return cp.StateStorageConfiguration{}, fmt.Errorf("failed to parse azurerm configuration: %v", diags.Errors())
		}
		_ = stateStorageConfiguration.FromAzureRMStorageConfiguration(cp.AzureRMStorageConfiguration{
			Type:               cp.StateStorageTypeAzurerm,
			ResourceGroupName:  azurermConfig.ResourceGroupName,
			StorageAccountName: azurermConfig.StorageAccountName,
			ContainerName:      azurermConfig.ContainerName,
			PathPrefix:         azurermConfig.PathPrefix,
		})

	default:
		return cp.StateStorageConfiguration{}, fmt.Errorf("unsupported state storage type: %s", storageType)
	}

	return *stateStorageConfiguration, nil
}
