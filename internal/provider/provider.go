package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
	canyondp "terraform-provider-humanitec-v2/internal/clients/canyon-dp"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/justinrixx/retryhttp"
)

const (
	HUM_CLIENT_ERR             = "Humanitec client error"
	HUM_API_ERR                = "Humanitec API error"
	HUM_PROVIDER_ERR           = "Provider error"
	HUM_INPUT_ERR              = "Input error"
	HUM_RESOURCE_NOT_FOUND_ERR = "Resource not found error"

	HUM_API_URL_ENV_VAR    = "HUMANITEC_API_URL"
	HUM_ORG_ID_ENV_VAR     = "HUMANITEC_ORG_ID"
	HUM_AUTH_TOKEN_ENV_VAR = "HUMANITEC_AUTH_TOKEN"

	HUM_DEFAULT_API_URL = "https://api.humanitec.dev"
)

// isBase64Encoded checks if a string is base64 encoded.
func isBase64Encoded(s string) bool {
	// Try to decode the string as base64
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

// Ensure HumanitecProvider satisfies various provider interfaces.
var _ provider.Provider = &HumanitecProvider{}
var _ provider.ProviderWithFunctions = &HumanitecProvider{}
var _ provider.ProviderWithEphemeralResources = &HumanitecProvider{}

// HumanitecProvider defines the provider implementation.
type HumanitecProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// HumanitecProvider describes the provider data model.
type HumanitecProviderModel struct {
	ApiUrl    types.String `tfsdk:"api_url"`
	OrgId     types.String `tfsdk:"org_id"`
	AuthToken types.String `tfsdk:"auth_token"`
}

type HumanitecProviderData struct {
	OrgId string

	CpClient canyoncp.ClientWithResponsesInterface
	DpClient canyondp.ClientWithResponsesInterface
}

func (p *HumanitecProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "platform-orchestrator"
	resp.Version = p.version
}

func (p *HumanitecProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Humanitec API URL prefix",
				Optional:            true,
			},
			"org_id": schema.StringAttribute{
				MarkdownDescription: "Humanitec Organization ID",
				Optional:            true,
			},
			"auth_token": schema.StringAttribute{
				MarkdownDescription: "Humanitec Auth Token",
				Sensitive:           true,
				Optional:            true,
			},
		},
	}
}

func (p *HumanitecProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data HumanitecProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiUrl := data.ApiUrl.ValueString()
	if v := os.Getenv(HUM_API_URL_ENV_VAR); apiUrl == "" && v != "" {
		apiUrl = v
	}
	if apiUrl == "" {
		apiUrl = HUM_DEFAULT_API_URL
	}

	orgId := data.OrgId.ValueString()
	if v := os.Getenv(HUM_ORG_ID_ENV_VAR); orgId == "" && v != "" {
		orgId = v
	}
	if orgId == "" {
		resp.Diagnostics.AddError(
			HUM_INPUT_ERR,
			"While configuring the provider, the Org ID was not found in "+
				"the HUMANITEC_ORG_ID environment variable or provider "+
				"configuration block org_id attribute.",
		)
	}

	authToken := data.AuthToken.ValueString()
	if v := os.Getenv(HUM_AUTH_TOKEN_ENV_VAR); authToken == "" && v != "" {
		authToken = v
	}

	u, err := url.Parse(apiUrl)
	if err != nil {
		resp.Diagnostics.AddError(HUM_INPUT_ERR, fmt.Sprintf("Unable to parse API URL: %s", err))
		return
	}

	extraHeaders := make(http.Header)
	if authToken != "" {
		// For now we support temporary Basic authentication with OrgId as a username and the token as a password
		if isBase64Encoded(authToken) {
			extraHeaders.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(orgId+":"+authToken)))
		} else {
			extraHeaders.Set("Authorization", "Bearer "+authToken)
		}
	} else if u.Hostname() == "localhost" {
		// For the local version, our auth is to just set the 'From' header directly.
		extraHeaders.Set("From", uuid.Nil.String())
	} else {
		resp.Diagnostics.AddError(
			HUM_INPUT_ERR,
			"While configuring the provider, the Auth token was not found in "+
				"the HUMANITEC_AUTH_TOKEN environment variable or provider "+
				"configuration block auth_token attribute.",
		)
	}

	// If there are some diagnostics, we should not continue creating the client, as it will fail anyway.
	if resp.Diagnostics.HasError() {
		return
	}

	extraHeadersEditor := func(ctx context.Context, req *http.Request) error {
		maps.Copy(req.Header, extraHeaders)
		return nil
	}

	client := &http.Client{
		Transport: retryhttp.New(),
		Timeout:   30 * time.Second,
	}

	cpc, err := canyoncp.NewClientWithResponses(apiUrl, canyoncp.WithRequestEditorFn(extraHeadersEditor), canyoncp.WithHTTPClient(client))
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to create Canyon CP client: %s", err.Error()))
		return
	}

	dpc, err := canyondp.NewClientWithResponses(apiUrl, canyondp.WithRequestEditorFn(extraHeadersEditor), canyondp.WithHTTPClient(client))
	if err != nil {
		resp.Diagnostics.AddError(HUM_CLIENT_ERR, fmt.Sprintf("Unable to create Canyon DP client: %s", err.Error()))
		return
	}

	respData := &HumanitecProviderData{
		OrgId:    orgId,
		CpClient: cpc,
		DpClient: dpc,
	}

	resp.DataSourceData = respData
	resp.ResourceData = respData
}

func (p *HumanitecProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewEnvironmentTypeResource,
		NewKubernetesRunnerResource,
		NewKubernetesGkeRunnerResource,
		NewKubernetesAgentRunnerResource,
		NewProviderResource,
		NewResourceTypeResource,
		NewModuleResource,
		NewModuleRuleResource,
		NewRunnerRuleResource,
		NewEnvironmentResource,
	}
}

func (p *HumanitecProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *HumanitecProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewEnvironmentTypeDataSource,
		NewKubernetesRunnerDataSource,
		NewKubernetesGkeRunnerDataSource,
		NewKubernetesAgentRunnerDataSource,
		NewProviderDataSource,
		NewResourceTypeDataSource,
		NewModuleDataSource,
		NewModuleRuleDataSource,
		NewRunnerRuleDataSource,
		NewEnvironmentDataSource,
	}
}

func (p *HumanitecProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HumanitecProvider{
			version: version,
		}
	}
}
