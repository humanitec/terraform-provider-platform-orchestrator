package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &EnvironmentTypeDataSource{}

func NewEnvironmentTypeDataSource() datasource.DataSource {
	return &EnvironmentTypeDataSource{}
}

// EnvironmentTypeDataSource defines the data source implementation.
type EnvironmentTypeDataSource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// EnvironmentTypeDataSourceModel describes the data source data model.
type EnvironmentTypeDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	Uuid        types.String `tfsdk:"uuid"`
}

func (d *EnvironmentTypeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_type"
}

func (d *EnvironmentTypeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Environment Type data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Environment Type ID",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
						"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
					),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Environment Type display name",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Environment Type UUID",
				Computed:            true,
			},
		},
	}
}

func (d *EnvironmentTypeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *EnvironmentTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentTypeDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetEnvironmentTypeWithResponse(ctx, d.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read environment type, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddError(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Environment type with ID %s not found in org %s", data.Id.ValueString(), d.orgId))
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read environment type, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	environmentType := httpResp.JSON200

	data.Id = types.StringValue(environmentType.Id)
	data.DisplayName = types.StringValue(environmentType.DisplayName)
	data.Uuid = types.StringValue(environmentType.Uuid.String())

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
