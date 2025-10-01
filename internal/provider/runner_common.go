package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
)

type RunnerResourceModel struct {
	Id                        types.String `tfsdk:"id"`
	Description               types.String `tfsdk:"description"`
	RunnerConfiguration       types.Object `tfsdk:"runner_configuration"`
	StateStorageConfiguration types.Object `tfsdk:"state_storage_configuration"`
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
