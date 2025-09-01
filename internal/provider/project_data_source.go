package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProjectDataSource{}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

type ProjectDataSource struct {
	cpClient canyoncp.ClientWithResponsesInterface
	orgId    string
}

func (d *ProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func projectDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The unique identifier for the Project within the Organization.",
			Required:            true,
		},
		"display_name": schema.StringAttribute{
			MarkdownDescription: "The display name of the Project.",
			Computed:            true,
		},
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the Project.",
			Computed:            true,
		},
		"created_at": schema.StringAttribute{
			MarkdownDescription: "The Created At timestamp of the Project in RFC3339 format.",
			Computed:            true,
		},
		"updated_at": schema.StringAttribute{
			MarkdownDescription: "The Updated At timestamp of the Project in RFC3339 format.",
			Computed:            true,
		},
		"status": schema.StringAttribute{
			MarkdownDescription: "The status of the Project.",
			Computed:            true,
		},
	}
}

func (d *ProjectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Project data source",
		Attributes:          projectDataSourceAttributes(),
	}
}

func (d *ProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.cpClient.GetProjectWithResponse(ctx, d.orgId, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to read project, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read project, unexpected status code: %d, body: %s", httpResp.StatusCode(), httpResp.Body))
		return
	}

	p := httpResp.JSON200
	data.Id = types.StringValue(p.Id)
	data.DisplayName = types.StringValue(p.DisplayName)
	data.Uuid = types.StringValue(p.Uuid.String())
	data.CreatedAt = types.StringValue(p.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(p.UpdatedAt.Format(time.RFC3339))
	data.Status = types.StringValue(string(p.Status))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
