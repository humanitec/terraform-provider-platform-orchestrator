package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

// AttributeTypeFromResourceSchemaAttr returns the attribute type for the given schema attribute.
func AttributeTypeFromResourceSchemaAttr(a schema.Attribute) (attr.Type, error) {
	switch typed := a.(type) {
	case schema.StringAttribute:
		return types.StringType, nil
	case schema.BoolAttribute:
		return types.BoolType, nil
	case schema.Float64Attribute:
		return types.Float64Type, nil
	case schema.Float32Attribute:
		return types.Float32Type, nil
	case schema.Int64Attribute:
		return types.Int64Type, nil
	case schema.Int32Attribute:
		return types.Int32Type, nil
	case schema.NumberAttribute:
		return types.NumberType, nil
	case schema.MapAttribute:
		return types.MapType{
			ElemType: typed.ElementType,
		}, nil
	case schema.ListAttribute:
		return types.ListType{
			ElemType: typed.ElementType,
		}, nil
	case schema.SingleNestedAttribute:
		attrs, err := AttributeTypesFromResourceSchema(typed.Attributes)
		return types.ObjectType{AttrTypes: attrs}, err
	default:
		// NOTE: add more cases as needed
		return nil, fmt.Errorf("unsupported attribute type for '%v': %T", a, typed)
	}
}

// AttributeTypesFromResourceSchema returns the attribute types map for the given schema.
func AttributeTypesFromResourceSchema(attributes map[string]schema.Attribute) (map[string]attr.Type, error) {
	attrTypes := make(map[string]attr.Type, len(attributes))
	for k, v := range attributes {
		attrType, err := AttributeTypeFromResourceSchemaAttr(v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
		attrTypes[k] = attrType
	}
	return attrTypes, nil
}
