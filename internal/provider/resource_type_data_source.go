package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ResourceTypeDataSource{}

func NewResourceTypeDataSource() datasource.DataSource {
	return &ResourceTypeDataSource{}
}

// ResourceTypeDataSource defines the data source implementation.
type ResourceTypeDataSource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// ResourceTypeDataSourceModel describes the data source data model.
type ResourceTypeDataSourceModel struct {
	Id                    types.String         `tfsdk:"id"`
	Description           types.String         `tfsdk:"description"`
	OutputSchema          jsontypes.Normalized `tfsdk:"output_schema"`
	IsDeveloperAccessible types.Bool           `tfsdk:"is_developer_accessible"`
}

func (d *ResourceTypeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_type"
}

func (d *ResourceTypeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Resource Type resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the Resource Type.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
						"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
					),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Resource Type.",
				Computed:            true,
			},
			"output_schema": schema.StringAttribute{
				MarkdownDescription: "The JSON schema for output parameters.",
				CustomType:          jsontypes.NormalizedType{},
				Computed:            true,
			},
			"is_developer_accessible": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this resource type is for developers to use in the manifest. Resource types with this flag set to false, will not be available as types of resources in a manifest.",
				Computed:            true,
			},
		},
	}
}

func (d *ResourceTypeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*HumanitecProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			HUM_PROVIDER_ERR,
			fmt.Sprintf("Expected *HumanitecProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.cpClient = providerData.CpClient
	d.orgId = providerData.OrgId
}

func (d *ResourceTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourceTypeDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetResourceTypeWithResponse(ctx, d.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read resource type, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == 404 {
		resp.Diagnostics.AddError(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Resource type with ID %s not found in org %s", data.Id.ValueString(), d.orgId))
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read resource type, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	resourceType := httpResp.JSON200

	var description types.String
	if resourceType.Description != nil {
		description = types.StringValue(*resourceType.Description)
	} else {
		description = types.StringNull()
	}

	outputSchemaBytes, err := json.Marshal(resourceType.OutputSchema)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Unable to marshal output schema: %s", err))
		return
	}

	data.Id = types.StringValue(resourceType.Id)
	data.Description = description
	data.OutputSchema = jsontypes.NewNormalizedValue(string(outputSchemaBytes))
	data.IsDeveloperAccessible = types.BoolValue(resourceType.IsDeveloperAccessible)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
