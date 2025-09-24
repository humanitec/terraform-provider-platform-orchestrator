package provider

import (
	"context"
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
var _ datasource.DataSource = &ProviderDataSource{}

func NewProviderDataSource() datasource.DataSource {
	return &ProviderDataSource{}
}

// ProviderDataSource defines the data source implementation.
type ProviderDataSource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// ProviderDataSourceModel describes the data source data model.
type ProviderDataSourceModel struct {
	Id                types.String         `tfsdk:"id"`
	Description       types.String         `tfsdk:"description"`
	ProviderType      types.String         `tfsdk:"provider_type"`
	Source            types.String         `tfsdk:"source"`
	VersionConstraint types.String         `tfsdk:"version_constraint"`
	Configuration     jsontypes.Normalized `tfsdk:"configuration"`
}

func (d *ProviderDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_provider"
}

func (d *ProviderDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Provider data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Provider ID",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
						"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
					),
				},
			},
			"provider_type": schema.StringAttribute{
				MarkdownDescription: "Provider type",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z][a-z0-9_-]+$`),
						"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
					),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Provider description",
				Computed:            true,
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "The source of the provider",
				Computed:            true,
			},
			"version_constraint": schema.StringAttribute{
				MarkdownDescription: "The version constraint for the provider",
				Computed:            true,
			},
			"configuration": schema.StringAttribute{
				MarkdownDescription: "JSON encoded configuration of the provider",
				Computed:            true,
				CustomType:          jsontypes.NormalizedType{},
			},
		},
	}
}

func (d *ProviderDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProviderDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetModuleProviderWithResponse(ctx, d.orgId, data.ProviderType.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read provider, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddError(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Provider with ID %s not found in org %s", data.Id.ValueString(), d.orgId))
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read provider, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	provider := httpResp.JSON200

	// Convert the provider to the data source model using the existing helper function
	convertedData := toProviderResourceModel(*provider)

	data.Id = convertedData.Id
	data.Description = convertedData.Description
	data.ProviderType = convertedData.ProviderType
	data.Source = convertedData.Source
	data.VersionConstraint = convertedData.VersionConstraint
	data.Configuration = convertedData.Configuration

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
