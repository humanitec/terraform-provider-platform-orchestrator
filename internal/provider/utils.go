package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func fromStringValueToStringPointer(str basetypes.StringValue) *string {
	if str.IsNull() || str.IsUnknown() {
		return nil
	}

	return str.ValueStringPointer()
}
