package provider

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	canyoncp "terraform-provider-humanitec-v2/internal/clients/canyon-cp"
	"terraform-provider-humanitec-v2/internal/ref"
)

var ecsRunnerConfigurationResourceSchema = schema.SingleNestedAttribute{
	MarkdownDescription: "The configuration of the AWS ECS Runner.",
	Required:            true,
	Attributes: map[string]schema.Attribute{
		"auth": schema.SingleNestedAttribute{
			MarkdownDescription: "Configuration to obtain temporary AWS security credentials by assuming an IAM role.",
			Required:            true,
			Attributes: map[string]schema.Attribute{
				"role_arn": schema.StringAttribute{
					MarkdownDescription: "The ARN of the role to assume.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^arn:aws:iam::[0-9]{12}:role/[a-zA-Z_0-9+=,.@\-/]+$`),
							"must be a valid IAM Role ARN",
						),
					},
				},
				"session_name": schema.StringAttribute{
					MarkdownDescription: "Session name to be used when assuming the role. If not provided, a default session name will be \"{org_id}-{runner_id}\".",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.LengthBetween(3, 64),
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-zA-Z0-9+=,.@\-_/]+$`),
							"must contain only valid characters (letters, digits, and +=,.@-_/)",
						),
					},
				},
				"sts_region": schema.StringAttribute{
					MarkdownDescription: "The AWS region identifier for the Security Token Service (STS) endpoint. If not provided, the cluster region will be used.",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-z]{2}-[a-z]+-\d$`),
							"must be a valid AWS region",
						),
					},
				},
			},
		},
		"job": schema.SingleNestedAttribute{
			MarkdownDescription: "The job configuration for the AWS ECS Runner.",
			Required:            true,
			Attributes: map[string]schema.Attribute{
				"region": schema.StringAttribute{
					MarkdownDescription: "The AWS Region.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-z]{2}-[a-z]+-\d$`),
							"must be a valid AWS region",
						),
					},
				},
				"cluster": schema.StringAttribute{
					MarkdownDescription: "The ECS Cluster name.",
					Required:            true,
					Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
				},
				"subnets": schema.ListAttribute{
					ElementType:         types.StringType,
					MarkdownDescription: "The list of subnets to use for the Runner. At least one subnet must be provided.",
					Required:            true,
					Validators: []validator.List{
						listvalidator.SizeBetween(1, 16),
					},
				},
				"execution_role_arn": schema.StringAttribute{
					MarkdownDescription: "The ARN of the IAM role to use for launching the ECS Task.",
					Required:            true,
					Validators:          []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^arn:aws:iam::\d{12}:role/[a-zA-Z_0-9+=,.@\-_/]+$`), "must be a valid IAM role ARN")},
				},
				"security_groups": schema.ListAttribute{
					ElementType:         types.StringType,
					MarkdownDescription: "The list of subnets to use for the Runner.",
					Optional:            true,
					Computed:            true,
					Validators: []validator.List{
						listvalidator.SizeAtMost(5),
						listvalidator.ValueStringsAre(stringvalidator.LengthBetween(1, 255)),
					},
				},
				"is_public_ip_enabled": schema.BoolAttribute{
					MarkdownDescription: "Whether to provision a public IP for the ECS Task.",
					Optional:            true,
					Computed:            true,
				},
				"task_role_arn": schema.StringAttribute{
					MarkdownDescription: "The ARN of the IAM role to use for running the ECS Task.",
					Optional:            true,
					Validators:          []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^arn:aws:iam::\d{12}:role/[a-zA-Z_0-9+=,.@\-_/]+$`), "must be a valid IAM role ARN")},
				},
				"image": schema.StringAttribute{
					MarkdownDescription: "The container image to use for the ECS Task. If not provided, a default canyon-runner image will be used.",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.LengthBetween(1, 255),
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-zA-Z0-9._:/-]+(?:@[a-z0-9]+:[a-fA-F0-9]+)?$`),
							"image must be a valid container image",
						),
					},
				},
				"environment": schema.MapAttribute{
					MarkdownDescription: "The plain-text environment variables to set for the ECS Task.",
					ElementType:         types.StringType,
					Optional:            true,
					Computed:            true,
					Validators: []validator.Map{
						mapvalidator.ValueStringsAre(stringvalidator.LengthAtMost(1024)),
					},
				},
				"secrets": schema.MapAttribute{
					MarkdownDescription: "The secrets to set for the Runner. The values must be Secrets Manager ARNs or Parameter Store ARNs.",
					ElementType:         types.StringType,
					Optional:            true,
					Computed:            true,
					Validators: []validator.Map{
						mapvalidator.ValueStringsAre(stringvalidator.RegexMatches(regexp.MustCompile(`^arn:aws:((secretsmanager:[^:]+:[^:]+:secret:[^:]+-[a-zA-Z0-9]{6})|(ssm:[^:]+:[^:]+:parameter/.+))$`), "must be a valid AWS Secret or Parameter ARN")),
					},
				},
			},
		},
	},
}

var ecsRunnerResourceSchema = schema.Schema{
	// This description is used by the documentation generator and the language server.
	MarkdownDescription: "AWS ECS Task Runner resource",
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The unique identifier for the Runner.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.RegexMatches(
					regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
					"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
				),
				stringvalidator.LengthAtMost(100),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "The description of the Runner.",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.LengthAtMost(200),
			},
		},
		"runner_configuration":        ecsRunnerConfigurationResourceSchema,
		"state_storage_configuration": commonRunnerStateStorageResourceSchema,
	},
}

func NewServerlessEcsRunnerResource() resource.Resource {
	return &commonRunnerResource{
		SubType:                          "serverless_ecs_runner",
		SchemaDef:                        ecsRunnerResourceSchema,
		ReadApiResponseIntoModel:         convertEcsRunnerApiIntoModel,
		ConvertRunnerConfigIntoCreateApi: convertEcsRunnerModelIntoRunnerConfigCreate,
		ConvertRunnerConfigIntoUpdateApi: convertEcsRunnerModelIntoRunnerConfigUpdate,
	}
}

type ServerlessEcsRunnerConfiguration struct {
	Auth KubernetesEksRunnerClusterAuth `tfsdk:"auth"`
	Job  ServerlessEcsRunnerJob         `tfsdk:"job"`
}

type ServerlessEcsRunnerJob struct {
	Region            types.String `tfsdk:"region"`
	Cluster           types.String `tfsdk:"cluster"`
	Subnets           types.List   `tfsdk:"subnets"`
	ExecutionRole     types.String `tfsdk:"execution_role_arn"`
	SecurityGroups    types.List   `tfsdk:"security_groups"`
	IsPublicIpEnabled types.Bool   `tfsdk:"is_public_ip_enabled"`
	TaskRole          types.String `tfsdk:"task_role_arn"`
	Image             types.String `tfsdk:"image"`
	Environment       types.Map    `tfsdk:"environment"`
	Secrets           types.Map    `tfsdk:"secrets"`
}

func convertEcsRunnerApiIntoModel(item canyoncp.Runner, _ commonRunnerModel) (commonRunnerModel, error) {
	typedSsc, _ := item.RunnerConfiguration.AsServerlessEcsRunnerConfiguration()

	runnerConfigurationModel, err := convertEcsRunnerApiConfigIntoObject(context.Background(), typedSsc)
	if err != nil {
		return commonRunnerModel{}, err
	}

	stateStorageConfigurationModel, err := parseStateStorageConfigurationResponse(context.Background(), item.StateStorageConfiguration)
	if err != nil {
		return commonRunnerModel{}, err
	}

	return commonRunnerModel{
		Id:                        types.StringValue(item.Id),
		Description:               types.StringPointerValue(item.Description),
		StateStorageConfiguration: *stateStorageConfigurationModel,
		RunnerConfiguration:       runnerConfigurationModel,
	}, nil
}

func convertEcsRunnerApiConfigIntoObject(ctx context.Context, typedSsc canyoncp.ServerlessEcsRunnerConfiguration) (basetypes.ObjectValue, error) {
	runnerConfig := ServerlessEcsRunnerConfiguration{
		Auth: KubernetesEksRunnerClusterAuth{
			RoleArn:     types.StringValue(typedSsc.Auth.RoleArn),
			SessionName: types.StringPointerValue(typedSsc.Auth.SessionName),
			StsRegion:   types.StringPointerValue(typedSsc.Auth.StsRegion),
		},
		Job: convertEcsRunnerApiJobIntoModel(typedSsc.Job),
	}

	attrs, err := AttributeTypesFromResourceSchema(ecsRunnerConfigurationResourceSchema.Attributes)
	if err != nil {
		return basetypes.ObjectValue{}, fmt.Errorf("failed to build schema: %v", err)
	}

	objectValue, diags := types.ObjectValueFrom(ctx, attrs, runnerConfig)
	if diags.HasError() {
		return basetypes.ObjectValue{}, fmt.Errorf("failed to build runner configuration from model parsing API response: %v", diags.Errors())
	}
	return objectValue, nil
}

func convertEcsRunnerApiJobIntoModel(j canyoncp.ServerlessEcsRunnerJob) ServerlessEcsRunnerJob {
	return ServerlessEcsRunnerJob{
		Region:  types.StringValue(j.Region),
		Cluster: types.StringValue(j.Cluster),
		Subnets: types.ListValueMust(types.StringType, slices.Collect(func(yield func(attr.Value) bool) {
			for _, subnet := range j.Subnets {
				yield(types.StringValue(subnet))
			}
		})),
		SecurityGroups: types.ListValueMust(types.StringType, slices.Collect(func(yield func(attr.Value) bool) {
			for _, sg := range j.SecurityGroups {
				yield(types.StringValue(sg))
			}
		})),
		ExecutionRole:     types.StringValue(j.ExecutionRoleArn),
		IsPublicIpEnabled: types.BoolValue(j.IsPublicIpEnabled),
		TaskRole:          types.StringPointerValue(j.TaskRoleArn),
		Image:             types.StringPointerValue(j.Image),

		Environment: types.MapValueMust(
			types.StringType,
			maps.Collect(func(yield func(string, attr.Value) bool) {
				for k, v := range j.Environment {
					yield(k, types.StringValue(v))
				}
			}),
		),
		Secrets: types.MapValueMust(
			types.StringType,
			maps.Collect(func(yield func(string, attr.Value) bool) {
				for k, v := range j.Secrets {
					yield(k, types.StringValue(v))
				}
			}),
		),
	}
}

func convertEcsRunnerModelIntoRunnerConfigCreate(ctx context.Context, obj types.Object) (canyoncp.RunnerConfiguration, error) {
	var runnerConfig ServerlessEcsRunnerConfiguration
	if diags := obj.As(ctx, &runnerConfig, basetypes.ObjectAsOptions{}); diags.HasError() {
		return canyoncp.RunnerConfiguration{}, fmt.Errorf("failed to parse runner configuration from model: %v", diags.Errors())
	}

	var runnerConfiguration = new(canyoncp.RunnerConfiguration)
	_ = runnerConfiguration.FromServerlessEcsRunnerConfiguration(canyoncp.ServerlessEcsRunnerConfiguration{
		Auth: canyoncp.AwsTemporaryAuth{
			RoleArn:     runnerConfig.Auth.RoleArn.ValueString(),
			SessionName: fromStringValueToStringPointer(runnerConfig.Auth.SessionName),
			StsRegion:   fromStringValueToStringPointer(runnerConfig.Auth.StsRegion),
		},
		Job: convertEcsRunnerJobModelIntoApi(runnerConfig.Job),
	})
	return *runnerConfiguration, nil
}

func convertEcsRunnerModelIntoRunnerConfigUpdate(ctx context.Context, obj types.Object) (canyoncp.RunnerConfigurationUpdate, error) {
	var runnerConfig ServerlessEcsRunnerConfiguration
	if diags := obj.As(ctx, &runnerConfig, basetypes.ObjectAsOptions{}); diags.HasError() {
		return canyoncp.RunnerConfigurationUpdate{}, fmt.Errorf("failed to parse runner configuration from model: %v", diags.Errors())
	}

	var updateRunnerConfiguration = new(canyoncp.RunnerConfigurationUpdate)
	_ = updateRunnerConfiguration.FromServerlessEcsRunnerConfigurationUpdateBody(canyoncp.ServerlessEcsRunnerConfigurationUpdateBody{
		Auth: &canyoncp.AwsTemporaryAuth{
			RoleArn:     runnerConfig.Auth.RoleArn.ValueString(),
			SessionName: fromStringValueToStringPointer(runnerConfig.Auth.SessionName),
			StsRegion:   fromStringValueToStringPointer(runnerConfig.Auth.StsRegion),
		},
		Job: ref.Ref(convertEcsRunnerJobModelIntoApi(runnerConfig.Job)),
	})
	return *updateRunnerConfiguration, nil
}

func convertEcsRunnerJobModelIntoApi(j ServerlessEcsRunnerJob) canyoncp.ServerlessEcsRunnerJob {
	// helper function to convert known types
	stringify := func(a attr.Value) string {
		if s, ok := a.(types.String); ok {
			return s.ValueString()
		}
		return ""
	}
	result := canyoncp.ServerlessEcsRunnerJob{
		Region:  j.Region.ValueString(),
		Cluster: j.Cluster.ValueString(),
		Subnets: slices.Collect(func(yield func(string) bool) {
			for _, value := range j.Subnets.Elements() {
				yield(stringify(value))
			}
		}),
		ExecutionRoleArn: j.ExecutionRole.ValueString(),
		SecurityGroups: slices.Collect(func(yield func(string) bool) {
			for _, value := range j.SecurityGroups.Elements() {
				yield(stringify(value))
			}
		}),
		IsPublicIpEnabled: j.IsPublicIpEnabled.ValueBool(),
		TaskRoleArn:       fromStringValueToStringPointer(j.TaskRole),
		Image:             fromStringValueToStringPointer(j.Image),
		Environment: maps.Collect(func(yield func(string, string) bool) {
			for k, v := range j.Environment.Elements() {
				yield(k, stringify(v))
			}
		}),
		Secrets: maps.Collect(func(yield func(string, string) bool) {
			for k, v := range j.Secrets.Elements() {
				yield(k, stringify(v))
			}
		}),
	}

	return result
}
