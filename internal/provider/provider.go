package provider

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"

	cp "terraform-provider-platform-orchestrator/internal/clients/platform-orchestrator-cp"
	dp "terraform-provider-platform-orchestrator/internal/clients/platform-orchestrator-dp"

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
	PO_CLIENT_ERR             = "Platform orchestrator client error"
	PO_API_ERR                = "Platform orchestrator API error"
	PO_PROVIDER_ERR           = "Provider error"
	PO_INPUT_ERR              = "Input error"
	PO_RESOURCE_NOT_FOUND_ERR = "Resource not found error"

	PO_API_URL_ENV_VAR    = "PO_API_URL"
	PO_ORG_ID_ENV_VAR     = "PO_ORG_ID"
	PO_AUTH_TOKEN_ENV_VAR = "PO_AUTH_TOKEN"

	// FIXME: Update the default API URL to match the new domain.
	PO_DEFAULT_API_URL = "[Default api url]"

	DefaultAsyncPollInterval = time.Second * 3
	DefaultAsyncTimeout      = time.Minute * 20
)

// Ensure PlatformOrchestratorProvider satisfies various provider interfaces.
var _ provider.Provider = &PlatformOrchestratorProvider{}
var _ provider.ProviderWithFunctions = &PlatformOrchestratorProvider{}
var _ provider.ProviderWithEphemeralResources = &PlatformOrchestratorProvider{}

// PlatformOrchestratorProvider defines the provider implementation.
type PlatformOrchestratorProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// PlatformOrchestratorProviderModel describes the provider data model.
type PlatformOrchestratorProviderModel struct {
	ConfigFilePath types.String `tfsdk:"hctl_config_file"`
	ApiUrl         types.String `tfsdk:"api_url"`
	OrgId          types.String `tfsdk:"org_id"`
	AuthToken      types.String `tfsdk:"auth_token"`
}

type PlatformOrchestratorProviderData struct {
	OrgId string

	CpClient cp.ClientWithResponsesInterface
	DpClient dp.ClientWithResponsesInterface
}

func (p *PlatformOrchestratorProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "platform-orchestrator"
	resp.Version = p.version
}

func (p *PlatformOrchestratorProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"hctl_config_file": schema.StringAttribute{
				MarkdownDescription: "Path to the hctl config file path. Takes precedences over the PO_ environment variables.",
				Optional:            true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Platform Orchestrator API URL prefix. Takes precedence over the contents of hctl_config_file but overridden by the PO_API_URL environment variable.",
				Optional:            true,
			},
			"org_id": schema.StringAttribute{
				MarkdownDescription: "Platform Orchestrator Organization ID. Takes precedence over the contents of hctl_config_file but overridden by the PO_ORG_ID environment variable.",
				Optional:            true,
			},
			"auth_token": schema.StringAttribute{
				MarkdownDescription: "Platform Orchestrator Auth Token. Takes precedence over the contents of hctl_config_file but overridden by the PO_AUTH_TOKEN environment variable.",
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
	homeDirPath, _ := os.UserHomeDir()
	homeDirPath = filepath.Join(homeDirPath, ".config", "hctl", "config.yaml")
	cfgDirPath, _ := os.UserConfigDir()
	cfgDirPath = filepath.Join(cfgDirPath, "hctl", "config.yaml")
	if _, err := os.Stat(cfgDirPath); err == nil {
		return cfgDirPath, nil
	} else if _, err = os.Stat(homeDirPath); err == nil {
		return homeDirPath, nil
	}
	return "", fmt.Errorf("failed to find hctl config file path: neither %s nor %s exists", cfgDirPath, homeDirPath)
}

func loadClientConfig(ctx context.Context, data PlatformOrchestratorProviderModel, diagnostics *diag.Diagnostics) (string, string, string) {
	apiUrl := data.ApiUrl.ValueString()
	orgId := data.OrgId.ValueString()
	authToken := data.AuthToken.ValueString()

	// the config file counts as hard coded if set specifically
	if p := data.ConfigFilePath.ValueString(); p != "" {
		if cfg, err := readConfigFile(p); err != nil {
			diagnostics.AddError(PO_PROVIDER_ERR, fmt.Sprintf("Failed to read config file '%s': %s", p, err))
		} else {
			if apiUrl == "" && cfg.ApiUrl != "" {
				tflog.Debug(ctx, "using platform-orchestrator api url from explicit hctl config file", map[string]interface{}{"path": p})
				apiUrl = cfg.ApiUrl
			}
			if orgId == "" && cfg.DefaultOrg != "" {
				tflog.Debug(ctx, "using platform-orchestrator org id from explicit hctl config file", map[string]interface{}{"path": p})
				orgId = cfg.DefaultOrg
			}
			if authToken == "" && cfg.Token != "" {
				tflog.Debug(ctx, "using platform-orchestrator auth token from explicit hctl config file", map[string]interface{}{"path": p})
				authToken = cfg.Token
			}
		}
	}

	// SECOND - we fall back to environment variables
	if v := os.Getenv(PO_API_URL_ENV_VAR); apiUrl == "" && v != "" {
		tflog.Debug(ctx, "using platform-orchestrator api url from environment variable")
		apiUrl = v
	}
	if v := os.Getenv(PO_ORG_ID_ENV_VAR); orgId == "" && v != "" {
		tflog.Debug(ctx, "using platform-orchestrator org id from environment variable")
		orgId = v
	}
	if v := os.Getenv(PO_AUTH_TOKEN_ENV_VAR); authToken == "" && v != "" {
		tflog.Debug(ctx, "using platform-orchestrator auth token from environment variable")
		authToken = v
	}

	// THIRD - we fall back to shared implicit config file
	if data.ConfigFilePath.IsNull() {
		if p, err := getConfigFilePath(); err != nil {
			tflog.Debug(ctx, "skipping implicit hctl config file load: "+err.Error())
		} else if cfg, err := readConfigFile(p); err != nil {
			diagnostics.AddError(PO_PROVIDER_ERR, fmt.Sprintf("Failed to read config file '%s': %s", p, err))
		} else {
			if apiUrl == "" && cfg.ApiUrl != "" {
				tflog.Debug(ctx, "using platform-orchestrator api url from implicit hctl config file", map[string]interface{}{"path": p})
				apiUrl = cfg.ApiUrl
			}
			if orgId == "" && cfg.DefaultOrg != "" {
				tflog.Debug(ctx, "using platform-orchestrator org id from implicit hctl config file", map[string]interface{}{"path": p})
				orgId = cfg.DefaultOrg
			}
			if authToken == "" && cfg.Token != "" {
				tflog.Debug(ctx, "using platform-orchestrator auth token from implicit hctl config file", map[string]interface{}{"path": p})
				authToken = cfg.Token
			}
		}
	}

	if apiUrl == "" {
		tflog.Debug(ctx, "using default platform-orchestrator api url")
		apiUrl = PO_DEFAULT_API_URL
	}

	return apiUrl, orgId, authToken
}

func (p *PlatformOrchestratorProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data PlatformOrchestratorProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiUrl, orgId, authToken := loadClientConfig(ctx, data, &resp.Diagnostics)

	if orgId == "" {
		resp.Diagnostics.AddError(
			PO_INPUT_ERR,
			"While configuring the provider, the Org ID was not found in "+
				"the PO_ORG_ID environment variable or provider "+
				"configuration block org_id attribute.",
		)
	}

	u, err := url.Parse(apiUrl)
	if err != nil {
		resp.Diagnostics.AddError(PO_INPUT_ERR, fmt.Sprintf("Unable to parse API URL: %s", err))
		return
	}

	extraHeaders := make(http.Header)
	if authToken != "" {
		extraHeaders.Set("Authorization", "Bearer "+authToken)
	} else if u.Hostname() == "localhost" {
		// For the local version, our auth is to just set the 'From' header directly.
		extraHeaders.Set("From", uuid.Nil.String())
	} else {
		resp.Diagnostics.AddError(
			PO_INPUT_ERR,
			"While configuring the provider, the Auth token was not found in "+
				"the PO_AUTH_TOKEN environment variable or provider "+
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

	cpc, err := cp.NewClientWithResponses(apiUrl, cp.WithRequestEditorFn(extraHeadersEditor), cp.WithHTTPClient(client))
	if err != nil {
		resp.Diagnostics.AddError(PO_CLIENT_ERR, fmt.Sprintf("Unable to create Platform Orchestrator CP client: %s", err.Error()))
		return
	}

	dpc, err := dp.NewClientWithResponses(apiUrl, dp.WithRequestEditorFn(extraHeadersEditor), dp.WithHTTPClient(client))
	if err != nil {
		resp.Diagnostics.AddError(PO_CLIENT_ERR, fmt.Sprintf("Unable to create Platform Orchestrator DP client: %s", err.Error()))
		return
	}

	respData := &PlatformOrchestratorProviderData{
		OrgId:    orgId,
		CpClient: cpc,
		DpClient: dpc,
	}

	resp.DataSourceData = respData
	resp.ResourceData = respData
}

func (p *PlatformOrchestratorProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewEnvironmentTypeResource,
		NewKubernetesRunnerResource,
		NewKubernetesEksRunnerResource,
		NewKubernetesGkeRunnerResource,
		NewKubernetesAgentRunnerResource,
		NewServerlessEcsRunnerResource,
		NewProviderResource,
		NewResourceTypeResource,
		NewModuleResource,
		NewModuleRuleResource,
		NewRunnerRuleResource,
		NewEnvironmentResource,
		NewDeploymentResource,
	}
}

func (p *PlatformOrchestratorProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *PlatformOrchestratorProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectDataSource,
		NewProjectsDataSource,
		NewEnvironmentTypeDataSource,
		NewKubernetesRunnerDataSource,
		NewKubernetesEksRunnerDataSource,
		NewKubernetesGkeRunnerDataSource,
		NewKubernetesAgentRunnerDataSource,
		NewServerlessEcsRunnerDataSource,
		NewProviderDataSource,
		NewResourceTypeDataSource,
		NewModuleDataSource,
		NewModuleRuleDataSource,
		NewRunnerRuleDataSource,
		NewEnvironmentDataSource,
	}
}

func (p *PlatformOrchestratorProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PlatformOrchestratorProvider{
			version: version,
		}
	}
}
