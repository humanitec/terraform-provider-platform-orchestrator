package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
)

func parseStateStorageConfigurationResponse[T any](ctx context.Context, ssc canyoncp.StateStorageConfiguration, schemaAttrs map[string]schema.Attribute, buildModel func(canyoncp.StateStorageConfiguration) (T, error)) (*basetypes.ObjectValue, error) {
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

func buildCommonStateStorageModel(ssc canyoncp.StateStorageConfiguration) (commonRunnerStateStorageModel, error) {
	var model commonRunnerStateStorageModel
	model.Type, _ = ssc.Discriminator()
	switch canyoncp.StateStorageType(model.Type) {
	case canyoncp.StateStorageTypeS3:
		typedSsc, _ := ssc.AsS3StorageConfiguration()
		model.S3Configuration = &commonRunnerS3StateStorageModel{
			Bucket:     typedSsc.Bucket,
			PathPrefix: typedSsc.PathPrefix,
		}
	case canyoncp.StateStorageTypeKubernetes:
		typedSsc, _ := ssc.AsK8sStorageConfiguration()
		model.KubernetesConfiguration = &commonRunnerKubernetesStateStorageModel{
			Namespace: typedSsc.Namespace,
		}
	case canyoncp.StateStorageTypeGcs:
		typedSsc, _ := ssc.AsGCSStorageConfiguration()
		model.GCSConfiguration = &commonRunnerGCSStateStorageModel{
			Bucket:     typedSsc.Bucket,
			PathPrefix: typedSsc.PathPrefix,
		}
	case canyoncp.StateStorageTypeAzurerm:
		typedSsc, _ := ssc.AsAzureRMStorageConfiguration()
		model.AzureRMConfiguration = &commonRunnerAzureRMStateStorageModel{
			ResourceGroupName:  typedSsc.ResourceGroupName,
			StorageAccountName: typedSsc.StorageAccountName,
			ContainerName:      typedSsc.ContainerName,
			LookupBlobEndpoint: typedSsc.LookupBlobEndpoint,
			PathPrefix:         typedSsc.PathPrefix,
		}
	default:
		return model, fmt.Errorf("unsupported state storage type: %s", model.Type)
	}
	return model, nil
}

func createStateStorageConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.StateStorageConfiguration, error) {
	storageTypeAttr, ok := obj.Attributes()["type"].(types.String)
	if !ok {
		return canyoncp.StateStorageConfiguration{}, fmt.Errorf("type attribute is not set or has unexpected type")
	}
	storageType := storageTypeAttr.ValueString()

	var stateStorageConfiguration = new(canyoncp.StateStorageConfiguration)
	switch canyoncp.StateStorageType(storageType) {
	case canyoncp.StateStorageTypeS3:
		s3Obj, ok := obj.Attributes()["s3_configuration"].(types.Object)
		if !ok || s3Obj.IsNull() {
			return canyoncp.StateStorageConfiguration{}, fmt.Errorf("s3 configuration in object is not set")
		}
		var s3Config commonRunnerS3StateStorageModel
		diags := s3Obj.As(ctx, &s3Config, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return canyoncp.StateStorageConfiguration{}, fmt.Errorf("failed to parse s3 configuration: %v", diags.Errors())
		}
		_ = stateStorageConfiguration.FromS3StorageConfiguration(canyoncp.S3StorageConfiguration{
			Type:       canyoncp.StateStorageTypeS3,
			Bucket:     s3Config.Bucket,
			PathPrefix: s3Config.PathPrefix,
		})

	case canyoncp.StateStorageTypeKubernetes:
		k8sObj, ok := obj.Attributes()["kubernetes_configuration"].(types.Object)
		if !ok || k8sObj.IsNull() {
			return canyoncp.StateStorageConfiguration{}, fmt.Errorf("k8s configuration in object is not set")
		}
		var k8sConfig commonRunnerKubernetesStateStorageModel
		diags := k8sObj.As(ctx, &k8sConfig, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return canyoncp.StateStorageConfiguration{}, fmt.Errorf("failed to parse kubernetes configuration: %v", diags.Errors())
		}
		_ = stateStorageConfiguration.FromK8sStorageConfiguration(canyoncp.K8sStorageConfiguration{
			Type:      canyoncp.StateStorageTypeKubernetes,
			Namespace: k8sConfig.Namespace,
		})

	case canyoncp.StateStorageTypeGcs:
		gcsObj, ok := obj.Attributes()["gcs_configuration"].(types.Object)
		if !ok || gcsObj.IsNull() {
			return canyoncp.StateStorageConfiguration{}, fmt.Errorf("gcs configuration in object is not set")
		}
		var gcsConfig commonRunnerGCSStateStorageModel
		diags := gcsObj.As(ctx, &gcsConfig, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return canyoncp.StateStorageConfiguration{}, fmt.Errorf("failed to parse gcs configuration: %v", diags.Errors())
		}
		_ = stateStorageConfiguration.FromGCSStorageConfiguration(canyoncp.GCSStorageConfiguration{
			Type:       canyoncp.StateStorageTypeGcs,
			Bucket:     gcsConfig.Bucket,
			PathPrefix: gcsConfig.PathPrefix,
		})

	case canyoncp.StateStorageTypeAzurerm:
		azurermObj, ok := obj.Attributes()["azurerm_configuration"].(types.Object)
		if !ok || azurermObj.IsNull() {
			return canyoncp.StateStorageConfiguration{}, fmt.Errorf("azurerm configuration in object is not set")
		}
		var azurermConfig commonRunnerAzureRMStateStorageModel
		diags := azurermObj.As(ctx, &azurermConfig, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return canyoncp.StateStorageConfiguration{}, fmt.Errorf("failed to parse azurerm configuration: %v", diags.Errors())
		}
		_ = stateStorageConfiguration.FromAzureRMStorageConfiguration(canyoncp.AzureRMStorageConfiguration{
			Type:               canyoncp.StateStorageTypeAzurerm,
			ResourceGroupName:  azurermConfig.ResourceGroupName,
			StorageAccountName: azurermConfig.StorageAccountName,
			ContainerName:      azurermConfig.ContainerName,
			LookupBlobEndpoint: azurermConfig.LookupBlobEndpoint,
			PathPrefix:         azurermConfig.PathPrefix,
		})

	default:
		return canyoncp.StateStorageConfiguration{}, fmt.Errorf("unsupported state storage type: %s", storageType)
	}

	return *stateStorageConfiguration, nil
}
