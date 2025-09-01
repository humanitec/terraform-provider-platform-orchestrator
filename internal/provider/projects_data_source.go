package provider

import (
	"context"
	"fmt"
	"net/http"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = &ProjectsDataSource{}

func NewProjectsDataSource() datasource.DataSource {
	return &ProjectsDataSource{}
}

type ProjectsDataSource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

type ProjectsDataSourceModel struct {
	Projects []ProjectModel `tfsdk:"projects"`
}

func (d *ProjectsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *ProjectsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Project data source",

		Attributes: map[string]schema.Attribute{
			"projects": schema.ListNestedAttribute{
				MarkdownDescription: "The list of projects.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: projectDataSourceAttributes(),
				},
			},
		},
	}
}

func (d *ProjectsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var items []ProjectModel
	var pageCursor *string
	for {
		httpResp, err := d.cpClient.ListProjectsWithResponse(ctx, d.orgId, &canyoncp.ListProjectsParams{
			Page: pageCursor,
		})
		if err != nil {
			resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read project, got error: %s", err))
			return
		}
		if httpResp.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read project, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
			return
		}
		for _, item := range httpResp.JSON200.Items {
			items = append(items, toProjectModel(item))
		}
		if httpResp.JSON200.NextPageToken == nil {
			break
		}
		pageCursor = httpResp.JSON200.NextPageToken
	}

	data.Projects = items

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
