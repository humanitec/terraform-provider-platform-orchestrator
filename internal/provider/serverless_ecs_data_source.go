package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewServerlessEcsRunnerDataSource() datasource.DataSource {
	return &commonRunnerDataSource{
		SubType: "serverless_ecs_runner",
		SchemaDef: schema.Schema{
			// This description is used by the documentation generator and the language server.
			MarkdownDescription: "Kubernetes GKE Runner data source",

			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "The unique identifier for the Runner.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-z](?:-?[a-z0-9]+)+$`),
							"must start with a lowercase letter, can contain lowercase letters, numbers, and hyphens and can not be empty.",
						),
					},
				},
				"description": schema.StringAttribute{
					MarkdownDescription: "The description of the Runner.",
					Computed:            true,
				},
				"runner_configuration": schema.SingleNestedAttribute{
					MarkdownDescription: "The configuration of the AWS ECS Runner.",
					Computed:            true,
					Attributes: map[string]schema.Attribute{
						"auth": schema.SingleNestedAttribute{
							MarkdownDescription: "Configuration to obtain temporary AWS security credentials by assuming an IAM role.",
							Required:            true,
							Attributes: map[string]schema.Attribute{
								"role_arn": schema.StringAttribute{
									MarkdownDescription: "The ARN of the role to assume.",
									Computed:            true,
								},
								"session_name": schema.StringAttribute{
									MarkdownDescription: "Session name to be used when assuming the role. If not provided, a default session name will be \"{org_id}-{runner_id}\".",
									Computed:            true,
								},
								"sts_region": schema.StringAttribute{
									MarkdownDescription: "The AWS region identifier for the Security Token Service (STS) endpoint. If not provided, the cluster region will be used.",
									Computed:            true,
								},
							},
						},
						"job": schema.SingleNestedAttribute{
							MarkdownDescription: "The job configuration for the AWS ECS Runner.",
							Required:            true,
							Attributes: map[string]schema.Attribute{
								"region": schema.StringAttribute{
									MarkdownDescription: "The AWS Region.",
									Computed:            true,
								},
								"cluster": schema.StringAttribute{
									MarkdownDescription: "The ECS Cluster name.",
									Computed:            true,
								},
								"subnets": schema.ListAttribute{
									ElementType:         types.StringType,
									MarkdownDescription: "The list of subnets to use for the Runner. At least one subnet must be provided.",
									Computed:            true,
								},
								"execution_role_arn": schema.StringAttribute{
									MarkdownDescription: "The ARN of the IAM role to use for launching the ECS Task.",
									Computed:            true,
								},
								"security_groups": schema.ListAttribute{
									ElementType:         types.StringType,
									MarkdownDescription: "The list of subnets to use for the Runner.",
									Computed:            true,
								},
								"is_public_ip_enabled": schema.BoolAttribute{
									MarkdownDescription: "Whether to provision a public IP for the ECS Task.",
									Computed:            true,
								},
								"task_role_arn": schema.StringAttribute{
									MarkdownDescription: "The ARN of the IAM role to use for running the ECS Task.",
									Computed:            true,
								},
								"image": schema.StringAttribute{
									MarkdownDescription: "The container image to use for the ECS Task. If not provided, a default humanitec-runner image will be used.",
									Computed:            true,
								},
								"environment": schema.MapAttribute{
									MarkdownDescription: "The plain-text environment variables to set for the ECS Task.",
									ElementType:         types.StringType,
									Computed:            true,
								},
								"secrets": schema.MapAttribute{
									MarkdownDescription: "The secrets to set for the Runner. The values must be Secrets Manager ARNs or Parameter Store ARNs.",
									ElementType:         types.StringType,
									Computed:            true,
								},
							},
						},
					},
				},
				"state_storage_configuration": commonRunnerStateStorageDataSourceSchema,
			},
		},
		ReadApiResponseIntoModel: convertEcsRunnerApiIntoModel,
	}
}
