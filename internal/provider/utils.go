package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// fromStringValueToStringPointer converts a StringValue to a pointer to a string.
func fromStringValueToStringPointer(str basetypes.StringValue) *string {
	if str.IsNull() || str.IsUnknown() {
		return nil
	}

	return str.ValueStringPointer()
}

// toStringValueOrNil returns a StringValue that is null if the input string pointer is nil, otherwise it returns a StringValue with the value of the string pointer.
func toStringValueOrNil(str *string) basetypes.StringValue {
	if str == nil {
		return types.StringNull()
	}
	return types.StringValue(*str)
}
