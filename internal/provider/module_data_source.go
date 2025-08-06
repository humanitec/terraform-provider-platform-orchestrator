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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ModuleDataSource{}

func NewModuleDataSource() datasource.DataSource {
	return &ModuleDataSource{}
}

// ModuleDataSource defines the data source implementation.
type ModuleDataSource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

// ModuleDataSourceModel describes the data source data model.
type ModuleDataSourceModel struct {
	Id               types.String         `tfsdk:"id"`
	Description      types.String         `tfsdk:"description"`
	ResourceType     types.String         `tfsdk:"resource_type"`
	ModuleSource     types.String         `tfsdk:"module_source"`
	ModuleSourceCode types.String         `tfsdk:"module_source_code"`
	ModuleInputs     jsontypes.Normalized `tfsdk:"module_inputs"`
	ProviderMapping  basetypes.MapValue   `tfsdk:"provider_mapping"`
	Coprovisioned    types.List           `tfsdk:"coprovisioned"`
	Dependencies     basetypes.MapValue   `tfsdk:"dependencies"`
}

func (d *ModuleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_module"
}

func (d *ModuleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Module data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique identifier for a module",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z](?:-?[a-z0-9]+)+$"), ""),
				},
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "An optional text description for this module",
			},
			"resource_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The resource type that this module provisions",
			},
			"module_source": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The source of the OpenTofu module backing this module",
			},
			"module_source_code": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The source code of the OpenTofu module backing this module",
			},
			"module_inputs": schema.StringAttribute{
				MarkdownDescription: "The JSON encoded string which represents the inputs to the module. These may contain expressions referencing the modules context.",
				Computed:            true,
				CustomType:          jsontypes.NormalizedType{},
			},
			"provider_mapping": schema.MapAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "A mapping of module providers to use when provisioning using this module.",
			},
			"coprovisioned": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "A set of resources to provision after or in parallel with the resource of the current module.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"class": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "A resource class requested by the resource graph. 'default' is the default value.",
						},
						"copy_dependents_from_current": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "If true, all resources that depend on the current resource will also depend on (be provisioned after) this coprovisioned resource.\n",
						},
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "A specific resource id requested by the resource graph",
						},
						"is_dependent_on_current": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "If true, this coprovisioned resource will have a dependency on the current resource so that the current\nresource must be successfully provisioned before the coprovisioned one is.\n",
						},
						"params": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "A JSON encoded string representing the parameters to pass for provisioning.",
							CustomType:          jsontypes.NormalizedType{},
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The resource type to provision",
						},
					},
				},
			},
			"dependencies": schema.MapNestedAttribute{
				Computed:            true,
				MarkdownDescription: "A mapping of alias to resource dependencies that must be provisioned with this module",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"class": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "A resource class requested by the resource graph. 'default' is the default value.",
						},
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "A specific resource id requested by the resource graph",
						},
						"params": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "A JSON encoded string representing the parameters to pass for provisioning.",
							CustomType:          jsontypes.NormalizedType{},
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The resource type to provision",
						},
					},
				},
			},
		},
	}
}

func (d *ModuleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ModuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ModuleDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetModuleWithResponse(ctx, d.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read module, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read module, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	module := httpResp.JSON200

	// Convert the module to the data source model using the existing helper function
	moduleModel, err := toModuleResourceModel(ctx, *module)
	if err != nil {
		resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to convert module response to model: %s", err))
		return
	}

	data.Id = moduleModel.Id
	data.Description = moduleModel.Description
	data.ResourceType = moduleModel.ResourceType
	data.ModuleSource = moduleModel.ModuleSource
	data.ModuleSourceCode = moduleModel.ModuleSourceCode
	data.ModuleInputs = moduleModel.ModuleInputs
	data.ProviderMapping = moduleModel.ProviderMapping
	data.Coprovisioned = moduleModel.Coprovisioned
	data.Dependencies = moduleModel.Dependencies

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}