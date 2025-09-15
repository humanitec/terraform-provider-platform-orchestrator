package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
	canyondp "terraform-provider-humanitec-v2/internal/clients/canyon-dp"

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
	ConfigFilePath types.String `tfsdk:"hctl_config_file"`
	ApiUrl         types.String `tfsdk:"api_url"`
	OrgId          types.String `tfsdk:"org_id"`
	AuthToken      types.String `tfsdk:"auth_token"`
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
			"hctl_config_file": schema.StringAttribute{
				MarkdownDescription: "Path to the hctl config file path. Takes precedences over the HUMANITEC_ environment variables.",
				Optional:            true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Humanitec API URL prefix. Takes precedence over the contents of hctl_config_file but overridden by the HUMANITEC_API_PREFIX environment variable.",
				Optional:            true,
			},
			"org_id": schema.StringAttribute{
				MarkdownDescription: "Humanitec Organization ID. Takes precedence over the contents of hctl_config_file but overridden by the HUMANITEC_ORG environment variable.",
				Optional:            true,
			},
			"auth_token": schema.StringAttribute{
				MarkdownDescription: "Humanitec Auth Token. Takes precedence over the contents of hctl_config_file but overridden by the HUMANITEC_AUTH_TOKEN environment variable.",
				Sensitive:           true,
				Optional:            true,
			},
		},
	}
}

type Config struct {
	HctlConfigFile string `yaml:"hctl_config_file" json:"hctl_config_file"`
	ApiUrl         string `yaml:"api_url" json:"api_url"`
	DefaultOrg     string `yaml:"default_org_id" json:"default_org_id"`
	Token          string `yaml:"token" json:"token"`
}

func readConfigFile(path string) (Config, error) {
	var cfg Config
	f, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			// If the file does not exist, return an empty config
			return cfg, nil
		}
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}
	if err := yaml.Unmarshal(f, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}
	return cfg, nil
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return path.Join(homeDir, ".config", "hctl", "config.yaml"), nil
}

func (p *HumanitecProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data HumanitecProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// FIRST - we configure based on hardcoded configuration
	apiUrl := data.ApiUrl.ValueString()
	orgId := data.OrgId.ValueString()
	authToken := data.AuthToken.ValueString()
	// the config file counts as hard coded if set specifically
	if p := data.ConfigFilePath.ValueString(); p != "" {
		if cfg, err := readConfigFile(p); err != nil {
			resp.Diagnostics.AddError(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to read config file '%s': %s", p, err))
		} else {
			if cfg.ApiUrl != "" {
				apiUrl = cfg.ApiUrl
			} else if cfg.DefaultOrg != "" {
				orgId = cfg.DefaultOrg
			} else if cfg.Token != "" {
				authToken = cfg.Token
			}
		}
	}

	// SECOND - we fall back to environment variables
	if v := os.Getenv(HUM_API_URL_ENV_VAR); apiUrl == "" && v != "" {
		apiUrl = v
	}
	if v := os.Getenv(HUM_ORG_ID_ENV_VAR); orgId == "" && v != "" {
		orgId = v
	}
	if v := os.Getenv(HUM_AUTH_TOKEN_ENV_VAR); authToken == "" && v != "" {
		authToken = v
	}

	// THIRD - we fall back to shared implicit config file
	if data.ConfigFilePath.IsNull() {
		if p, err := getConfigFilePath(); err != nil {
			resp.Diagnostics.AddWarning(HUM_PROVIDER_ERR, err.Error())
		} else if cfg, err := readConfigFile(p); err != nil {
			resp.Diagnostics.AddWarning(HUM_PROVIDER_ERR, fmt.Sprintf("Failed to read config file '%s': %s", p, err))
		} else {
			if cfg.ApiUrl != "" {
				apiUrl = cfg.ApiUrl
			} else if cfg.DefaultOrg != "" {
				orgId = cfg.DefaultOrg
			} else if cfg.Token != "" {
				authToken = cfg.Token
			}
		}
	}

	if apiUrl == "" {
		apiUrl = HUM_DEFAULT_API_URL
	}
	if orgId == "" {
		resp.Diagnostics.AddError(
			HUM_INPUT_ERR,
			"While configuring the provider, the Org ID was not found in "+
				"the HUMANITEC_ORG_ID environment variable or provider "+
				"configuration block org_id attribute.",
		)
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
		NewProjectResource,
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
		NewProjectDataSource,
		NewProjectsDataSource,
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
