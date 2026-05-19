package resource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var (
	_ resource.Resource                = &dwsuEntraIdConfig{}
	_ resource.ResourceWithConfigure   = &dwsuEntraIdConfig{}
	_ resource.ResourceWithImportState = &dwsuEntraIdConfig{}
)

func NewDwsuEntraIdConfig() resource.Resource {
	return &dwsuEntraIdConfig{}
}

type dwsuEntraIdConfig struct {
	RelytClientResource
}

func (r *dwsuEntraIdConfig) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dwsu_entraid_config"
}

func (r *dwsuEntraIdConfig) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"dwsu_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The ID of the DWSU to attach Entra ID SSO config to.",
			},
			"tenant_id": schema.StringAttribute{
				Required:    true,
				Description: "Azure AD tenant (directory) GUID.",
			},
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "Azure AD application (client) GUID.",
			},
			"tenant_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("single"),
				Description: "One of 'single' (only users in the configured tenant can log in) or 'multi' (any Azure org user). Default 'single'. Invalid values are rejected by the backend at apply time.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether SSO is active. Set false to retain config but disable login. Default true.",
			},
		},
	}
}

// Stubbed; implemented in subsequent tasks.
func (r *dwsuEntraIdConfig) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
}
func (r *dwsuEntraIdConfig) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}
func (r *dwsuEntraIdConfig) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *dwsuEntraIdConfig) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
func (r *dwsuEntraIdConfig) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("dwsu_id"), req, resp)
}
