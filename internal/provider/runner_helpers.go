package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func commonStateStorageConfigurationAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"type": types.StringType,
		"kubernetes_configuration": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"namespace": types.StringType,
			},
		},
	}
}

func parseStateStorageConfigurationResponse(ctx context.Context, k8sStateStorageConfiguration canyoncp.K8sStorageConfiguration) *basetypes.ObjectValue {
	var stateStorageConfig commonRunnerStateStorageModel

	stateStorageConfig.Type = string(k8sStateStorageConfiguration.Type)
	stateStorageConfig.KubernetesConfiguration = &commonRunnerKubernetesStateStorageModel{
		Namespace: k8sStateStorageConfiguration.Namespace,
	}

	objectValue, diags := types.ObjectValueFrom(ctx, commonStateStorageConfigurationAttributes(), stateStorageConfig)
	if diags.HasError() {
		tflog.Warn(ctx, "can't parse state storage configuration from model", map[string]interface{}{"err": diags.Errors()})
		return nil
	}
	return &objectValue
}

func createStateStorageConfigurationFromObject(ctx context.Context, obj types.Object) (canyoncp.StateStorageConfiguration, error) {
	var stateStorageConfig commonRunnerStateStorageModel
	diags := obj.As(ctx, &stateStorageConfig, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return canyoncp.StateStorageConfiguration{}, fmt.Errorf("failed to parse state storage configuration from model: %v", diags.Errors())
	}

	var stateStorageConfiguration = new(canyoncp.StateStorageConfiguration)
	_ = stateStorageConfiguration.FromK8sStorageConfiguration(canyoncp.K8sStorageConfiguration{
		Type:      canyoncp.StateStorageType(stateStorageConfig.Type),
		Namespace: stateStorageConfig.KubernetesConfiguration.Namespace,
	})
	return *stateStorageConfiguration, nil
}
