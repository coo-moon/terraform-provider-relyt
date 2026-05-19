// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"math"
	"os"
	"strconv"
	"terraform-provider-relyt/internal/provider/client"
	relytDS "terraform-provider-relyt/internal/provider/datasource"
	"terraform-provider-relyt/internal/provider/model"
	relytRS "terraform-provider-relyt/internal/provider/resource"
)

// Ensure RelytProvider satisfies various provider interfaces.
var _ provider.Provider = &RelytProvider{}
var _ provider.ProviderWithFunctions = &RelytProvider{}
var _ provider.ProviderWithMetaSchema = &RelytProvider{}

// RelytProvider defines the provider implementation.
type RelytProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type RelytProviderEnv struct {
	EnvKey         string
	PropertyName   string
	SummarySuggest string
	detailSuggest  string
}

var (
	apiHostEnv = RelytProviderEnv{
		EnvKey:         "RELYT_API_HOST",
		PropertyName:   "api_host",
		SummarySuggest: "Unknown Relyt API Host",
		detailSuggest: "The provider cannot create the Relyt API client as there is an unknown configuration value for the Relyt API apiHost. " +
			"Either target apply the source of the value first, set the value statically in the configuration, or use the RELYT_API_HOST environment variable.",
	}
	authKeyEnv = RelytProviderEnv{
		EnvKey:         "RELYT_AUTH_KEY",
		PropertyName:   "auth_key",
		SummarySuggest: "Unknown Relyt Auth EnvKey",
		detailSuggest: "The provider cannot create the Relyt API client as there is an unknown configuration value for the Relyt Auth EnvKey. " +
			"Either target apply the source of the value first, set the value statically in the configuration, or use the RELYT_AUTH_KEY environment variable.",
	}
)

func (p *RelytProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "relyt"
	resp.Version = p.version
}

func (p *RelytProvider) MetaSchema(ctx context.Context, request provider.MetaSchemaRequest, response *provider.MetaSchemaResponse) {
	//response.Schema = metaschema.Schema{
	//	Attributes: map[string]metaschema.Attribute{
	//		"data_access_config": schema.SingleNestedAttribute{
	//			Optional:    true,
	//			Description: "data_access_configs",
	//			Attributes: map[string]schema.Attribute{
	//				"access_key":     schema.StringAttribute{Required: true, Description: "access Key"},
	//				"secret_key":     schema.StringAttribute{Required: true, Description: "secret Key"},
	//				"endpoint":       schema.StringAttribute{Required: true, Description: "data access endpoint"},
	//				"client_timeout": schema.Int32Attribute{Optional: true, Description: "client timeout seconds! default 10s"},
	//			},
	//		},
	//	},
	//}
}

// 定义provider能接受的参数，类型，是否可选等
func (p *RelytProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	tflog.Info(ctx, "===== provider scheme get ")
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			//"endpoint": schema.StringAttribute{
			//	MarkdownDescription: "Example provider attribute",
			//	Optional:            true,
			//},
			"api_host": schema.StringAttribute{
				Optional:    true,
				Description: "target api address",
			},
			"auth_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: " Your Console Auth Key! Can be set through env 'RELYT_AUTH_KEY'",
			},
			"role": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "your role",
			},
			"resource_check_timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "Timeout second used in wait for create and delete dwsu or dps! Defaults 1800",
			},
			"resource_check_interval": schema.Int64Attribute{
				Optional:    true,
				Description: "Interval second used in wait for cycle check! Defaults 5",
			},
			"client_timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "http client timeout seconds! Defaults 10",
			},
			"data_access_config": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "data_access_configs",
				Attributes: map[string]schema.Attribute{
					"access_key":     schema.StringAttribute{Required: true, Description: "The access key for Open API operations."},
					"secret_key":     schema.StringAttribute{Required: true, Description: "The secret key for Open API operations."},
					"endpoint":       schema.StringAttribute{Required: true, Description: "The VPC endpoint for the private link, which must be in the http://<dns_name>:8180 format. Replace <dns_name> with the DNS name of the VPC endpoint you have obtained from Amazon VPC."},
					"client_timeout": schema.Int32Attribute{Optional: true, Description: "The data access client timeout seconds!"},
				},
			},
		},
	}
}

// 读取配置文件
func (p *RelytProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {

	tflog.Info(ctx, "@@@@@@@@@@@@@@@@@@@@@@  run into check info ")

	var data model.RelytProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if data.ApiHost.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root(apiHostEnv.PropertyName),
			apiHostEnv.SummarySuggest,
			apiHostEnv.detailSuggest,
		)
	}

	if data.AuthKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root(authKeyEnv.PropertyName),
			authKeyEnv.SummarySuggest,
			authKeyEnv.detailSuggest,
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	apiHost := os.Getenv(apiHostEnv.EnvKey)
	if !data.ApiHost.IsNull() {
		apiHost = data.ApiHost.ValueString()
	}
	authKey := os.Getenv(authKeyEnv.EnvKey)
	if !data.AuthKey.IsNull() {
		authKey = data.AuthKey.ValueString()
	}
	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if authKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root(authKeyEnv.PropertyName),
			"Missing Relyt AUTH KEY",
			"The provider cannot create the Relyt API client as there is a missing or empty value for the Relyt AUTH KEY. "+
				"Set the apiHost value in the configuration or use the "+authKeyEnv.EnvKey+" environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if apiHost == "" {
		//apiHost的默认值
		apiHost = "https://api.data.cloud"
	}
	resourceWaitTimeout := int64(1800)
	checkInterval := int32(5)
	clientTimeout := int32(10)
	if !data.ResourceCheckTimeout.IsNull() {
		tflog.Info(ctx, "resource check wait isn't null! set value:"+strconv.FormatInt(data.ResourceCheckTimeout.ValueInt64(), 10))
		resourceWaitTimeout = data.ResourceCheckTimeout.ValueInt64()
		//if resourceWaitTimeout < 500 {
		//	resp.Diagnostics.AddAttributeError(path.Root("resource_check_timeout"), "invalid value", "shouldn't less than 500")
		//}
	}
	if !data.ResourceCheckInterval.IsNull() {
		tflog.Info(ctx, "resource check wait isn't null! set value:"+strconv.FormatInt(data.ResourceCheckTimeout.ValueInt64(), 10))
		if data.ResourceCheckInterval.ValueInt64() < 5 || data.ResourceCheckInterval.ValueInt64() >= math.MaxInt32 {
			resp.Diagnostics.AddAttributeError(path.Root("resource_check_interval"), "invalid value", "should be grater than 5")
		}
		checkInterval = int32(data.ResourceCheckInterval.ValueInt64())
	}
	if !data.ClientTimeout.IsNull() {
		tflog.Info(ctx, "client timeout isn't null! set value:"+strconv.FormatInt(data.ClientTimeout.ValueInt64(), 10))
		if data.ClientTimeout.ValueInt64() <= 1 || data.ClientTimeout.ValueInt64() >= math.MaxInt32 {
			resp.Diagnostics.AddAttributeError(path.Root("client_timeout"), "invalid value", "should be grater than 1")
		}
		clientTimeout = int32(data.ClientTimeout.ValueInt64())
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Example client configuration for data sources and resources

	// Create a new Relyt client using the configuration values
	tflog.Info(ctx, fmt.Sprintf(" host: %s auth: %s, role: %s  check timeout: %d interval: %d",
		apiHost, authKey, data.Role.ValueString(), resourceWaitTimeout, checkInterval))
	roleId := data.Role.ValueString()
	clientConfig := client.RelytClientConfig{
		ApiHost:       apiHost,
		AuthKey:       authKey,
		Role:          roleId,
		CheckTimeOut:  resourceWaitTimeout,
		CheckInterval: checkInterval,
		ClientTimeout: clientTimeout,
	}
	if data.DataAccessConfig != nil {
		clientConfig.RelytDatabaseClientConfig = &client.RelytDatabaseClientConfig{
			DmsHost:       data.DataAccessConfig.Endpoint.ValueString(),
			AccessKey:     data.DataAccessConfig.AccessKey.ValueString(),
			SecretKey:     data.DataAccessConfig.SecretKey.ValueString(),
			ClientTimeout: 60,
		}
		//if clientConfig.RelytDatabaseClientConfig.AccessKey == "" {
		//	resp.Diagnostics.AddError("data_access_config error", "access_key can't be empty string")
		//}
		//if clientConfig.RelytDatabaseClientConfig.SecretKey == "" {
		//	resp.Diagnostics.AddError("data_access_config error", "secret_key can't be empty string")
		//}
		//if clientConfig.RelytDatabaseClientConfig.DmsHost == "" {
		//	resp.Diagnostics.AddError("data_access_config error", "endpoint can't be empty string")
		//}

		if !data.DataAccessConfig.ClientTimeout.IsNull() {
			dataAccessClientTimeOut := data.DataAccessConfig.ClientTimeout.ValueInt32()
			if dataAccessClientTimeOut <= 0 {
				resp.Diagnostics.AddError("wrong data_access_config config!", " client_timeout must greater than 0")
				return
			}
			clientConfig.RelytDatabaseClientConfig.ClientTimeout = dataAccessClientTimeOut
		}
		if resp.Diagnostics.HasError() {
			return
		}
	}
	relytClient, err := client.NewRelytClient(clientConfig)
	//relytClient.RelytClientConfig.RegionApi = data.RegionApi.ValueString()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Relyt API Client",
			"An unexpected error occurred when creating the Relyt API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Relyt Client Error: "+err.Error(),
		)
		return
	}
	resp.DataSourceData = &relytClient
	resp.ResourceData = &relytClient
}

func (p *RelytProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		relytRS.NewdwUserResource,
		relytRS.NewDpsResource,
		relytRS.NewDwsuResource,
		relytRS.NewPrivateLinkResource,
		relytRS.NewDwsuIntegrationInfoResource,
		relytRS.NewDwsuDatabaseResource,
		relytRS.NewDwsuExternalSchemaResource,
		relytRS.NewdwsuUserPolicy,
		relytRS.NewDwsuEntraIdConfig,
		//relytRS.NewTestResource,
	}
}

func (p *RelytProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	tflog.Info(ctx, "===== provider datasource get ")
	return []func() datasource.DataSource{
		//relytDS.NewServiceAccountDataSource,
		relytDS.NewBoto3DataSource,
		relytDS.NewDwsuDatabasesDataSource,
		relytDS.NewDwsuDatabaseDetailDataSource,
		relytDS.NewDwsuSchemasDataSource,
		relytDS.NewDwsuSchemaDetailDataSource,
		relytDS.NewDwsuListDataSource,
		relytDS.NewCloudRegionListDataSource,
	}
}

func (p *RelytProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		//NewExampleFunction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RelytProvider{
			version: version,
		}
	}
}
