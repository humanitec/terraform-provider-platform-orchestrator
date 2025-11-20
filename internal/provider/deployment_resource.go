package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"

	canyondp "terraform-provider-humanitec-v2/internal/clients/canyon-dp"
	"terraform-provider-humanitec-v2/internal/ref"
)

var _ resource.Resource = &DeploymentResource{}
var _ resource.ResourceWithConfigure = &DeploymentResource{}

func NewDeploymentResource() resource.Resource {
	return &DeploymentResource{}
}

type DeploymentResource struct {
	dpClient canyondp.ClientWithResponsesInterface
	orgId    string
}

type DeploymentResourceModel struct {
	ProjectId     types.String   `tfsdk:"project_id"`
	EnvId         types.String   `tfsdk:"env_id"`
	Manifest      types.String   `tfsdk:"manifest"`
	Mode          types.String   `tfsdk:"mode"`
	Id            types.String   `tfsdk:"id"`
	CreatedAt     types.String   `tfsdk:"created_at"`
	CompletedAt   types.String   `tfsdk:"completed_at"`
	Status        types.String   `tfsdk:"status"`
	StatusMessage types.String   `tfsdk:"status_message"`
	RunnerId      types.String   `tfsdk:"runner_id"`
	WaitFor       types.Bool     `tfsdk:"wait_for"`
	Outputs       types.String   `tfsdk:"outputs"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func (d *DeploymentResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_deployment"
}

func (d *DeploymentResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Deployment resource",

		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The project ID to deploy.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
						"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"env_id": schema.StringAttribute{
				MarkdownDescription: "The environment ID to deploy.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
						"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"manifest": schema.StringAttribute{
				MarkdownDescription: "The YAML/JSON encoded manifest to deploy.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mode": schema.StringAttribute{
				MarkdownDescription: "The mode of the deployment. 'deploy' (the default) or 'plan_only'.",
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(string(canyondp.Deploy)),
				Validators: []validator.String{
					stringvalidator.OneOf(string(canyondp.Deploy), string(canyondp.PlanOnly)),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Deployment.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The date and time when the deployment was created.",
				Computed:            true,
			},
			"completed_at": schema.StringAttribute{
				MarkdownDescription: "The date and time when the deployment was completed.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the deployment (succeeded, failed).",
				Computed:            true,
			},
			"status_message": schema.StringAttribute{
				MarkdownDescription: "An optional message associated with the status.",
				Computed:            true,
			},
			"runner_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the runner used in this deployment.",
				Computed:            true,
			},
			"wait_for": schema.BoolAttribute{
				MarkdownDescription: "Whether to wait for the deployment to complete. Defaults to true. If false, the output will be empty.",
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(true),
			},
			"outputs": schema.StringAttribute{
				MarkdownDescription: "The JSON encoded outputs of the deployment.",
				Computed:            true,
				Sensitive:           true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Delete: true}),
		},
	}
}

func (d *DeploymentResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	providerData, ok := request.ProviderData.(*HumanitecProviderData)
	if !ok {
		response.Diagnostics.AddError(
			HUM_PROVIDER_ERR,
			fmt.Sprintf("Expected *HumanitecProviderData, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)
		return
	}

	d.dpClient = providerData.DpClient
	d.orgId = providerData.OrgId
}

func (d *DeploymentResource) doDeployment(ctx context.Context, data *DeploymentResourceModel, diags *diag.Diagnostics) (outputsKey *age.X25519Identity) {
	if data.Mode.IsNull() {
		data.Mode = types.StringValue(string(canyondp.Deploy))
	}
	if data.WaitFor.IsNull() {
		data.WaitFor = types.BoolValue(true)
	}

	var manifest canyondp.DeploymentManifest
	if err := yaml.Unmarshal([]byte(data.Manifest.ValueString()), &manifest); err != nil {
		diags.AddError(HUM_API_ERR, fmt.Sprintf("Unable to parse manifest, got error: %s", err))
		return
	}

	outputsKey, _ = age.GenerateX25519Identity()
	if r, err := d.dpClient.CreateDeploymentWithResponse(
		ctx, d.orgId, &canyondp.CreateDeploymentParams{IdempotencyKey: ref.Ref(uuid.NewString())},
		canyondp.DeploymentCreateBody{
			ProjectId:                 data.ProjectId.ValueString(),
			EnvId:                     data.EnvId.ValueString(),
			Manifest:                  manifest,
			Mode:                      canyondp.DeploymentCreateBodyMode(data.Mode.ValueString()),
			EncryptedOutputsRecipient: ref.Ref(outputsKey.Recipient().String()),
		},
	); err != nil {
		diags.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to create deployment, got error: %s", err))
		return
	} else if r.StatusCode() != http.StatusCreated {
		diags.AddError(HUM_API_ERR, fmt.Sprintf("Unable to create deployment, unexpected status code: %d, body: %s", r.StatusCode(), r.Body))
		return
	} else {
		data.Id = types.StringValue(r.JSON201.Id.String())
		data.CreatedAt = types.StringValue(r.JSON201.CreatedAt.Format(time.RFC3339))
		data.CompletedAt = types.StringNull()
		data.Outputs = types.StringNull()
		data.Status = types.StringValue(r.JSON201.Status)
		data.StatusMessage = types.StringValue(r.JSON201.StatusMessage)
		data.RunnerId = types.StringValue(r.JSON201.RunnerId)
	}
	return outputsKey
}

func (d *DeploymentResource) waitForDeployment(ctx context.Context, data *DeploymentResourceModel, diags *diag.Diagnostics, outputsKey *age.X25519Identity) {
	deleteTimeout, dd := data.Timeouts.Create(ctx, DefaultAsyncTimeout)
	if dd.HasError() {
		diags.Append(dd...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	deploymentUuid, err := uuid.Parse(data.Id.ValueString())
	if err != nil {
		diags.AddError(HUM_API_ERR, fmt.Sprintf("Unable to parse deployment ID, got error: %s", err))
		return
	}

	for {
		if r, err := d.dpClient.WaitForDeploymentCompleteWithResponse(ctx, d.orgId, deploymentUuid, &canyondp.WaitForDeploymentCompleteParams{}); err != nil {
			if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
				continue
			}
			diags.AddError(HUM_API_ERR, fmt.Sprintf("Unable to wait for deployment to complete, got error: %s", err))
			return
		} else if r.StatusCode() == http.StatusRequestTimeout {
			if err := ctx.Err(); err != nil {
				diags.AddError(HUM_API_ERR, fmt.Sprintf("Unable to wait for deployment to complete, got error: %s", err))
				return
			}
			continue
		} else if r.StatusCode() != http.StatusOK {
			diags.AddError(HUM_API_ERR, fmt.Sprintf("Unable to wait for deployment to complete, unexpected status code: %d, body: %s", r.StatusCode(), r.Body))
			return
		} else {
			data.Status = types.StringValue(r.JSON200.Status)
			data.StatusMessage = types.StringValue(r.JSON200.StatusMessage)
			data.CompletedAt = types.StringValue(r.JSON200.CompletedAt.Format(time.RFC3339))
			if data.Status.ValueString() == "succeeded" {
				if r, err := d.dpClient.GetDeploymentEncryptedOutputsWithResponse(ctx, d.orgId, deploymentUuid); err != nil {
					diags.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read deployment outputs, got error: %s", err))
				} else if r.StatusCode() != http.StatusOK {
					diags.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read deployment outputs, unexpected status code: %d, body: %s", r.StatusCode(), r.Body))
				} else {
					if decrypted, err := age.Decrypt(base64.NewDecoder(base64.StdEncoding, strings.NewReader(r.JSON200.Raw)), outputsKey); err != nil {
						diags.AddError(HUM_API_ERR, fmt.Sprintf("Unable to decrypt deployment outputs, got error: %s", err))
					} else if raw, err := io.ReadAll(decrypted); err != nil {
						diags.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read deployment outputs, got error: %s", err))
					} else {
						data.Outputs = types.StringValue(string(raw))
					}
				}
			} else {
				diags.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Deployment failed: %s", data.StatusMessage))
			}
			return
		}
	}
}

func (d *DeploymentResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data DeploymentResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	outputsKey := d.doDeployment(ctx, &data, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if data.WaitFor.ValueBool() {
		d.waitForDeployment(ctx, &data, &response.Diagnostics, outputsKey)
		response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	}
}

func (d *DeploymentResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data DeploymentResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	deploymentUuid, err := uuid.Parse(data.Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to parse deployment ID, got error: %s", err))
		return
	}

	if r, err := d.dpClient.GetDeploymentWithResponse(ctx, d.orgId, deploymentUuid); err != nil {
		response.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Unable to read deployment, got error: %s", err))
		return
	} else if r.StatusCode() == http.StatusNotFound {
		response.Diagnostics.AddWarning(HUM_RESOURCE_NOT_FOUND_ERR, fmt.Sprintf("Deployment with ID %s not found, assuming it has been deleted.", data.Id.ValueString()))
		response.State.RemoveResource(ctx)
		return
	} else if r.StatusCode() != http.StatusOK {
		response.Diagnostics.AddError(HUM_API_ERR, fmt.Sprintf("Unable to read deployment, unexpected status code: %d, body: %s", r.StatusCode(), r.Body))
		return
	} else {
		// Just refresh the status/status_message/completed_at fields.
		data.Status = types.StringValue(r.JSON200.Status)
		data.StatusMessage = types.StringValue(r.JSON200.StatusMessage)
		data.CompletedAt = types.StringNull()
		if r.JSON200.CompletedAt != nil {
			data.CompletedAt = types.StringValue(r.JSON200.CompletedAt.Format(time.RFC3339))
		}
	}
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (d *DeploymentResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data DeploymentResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	outputsKey := d.doDeployment(ctx, &data, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if data.WaitFor.ValueBool() {
		d.waitForDeployment(ctx, &data, &response.Diagnostics, outputsKey)
		response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	}
}

func (d *DeploymentResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
}
