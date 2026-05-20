package resource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"terraform-provider-relyt/internal/provider/client"
	"terraform-provider-relyt/internal/provider/common"
	tfModel "terraform-provider-relyt/internal/provider/model"
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
			"dms_host": schema.StringAttribute{
				Optional:    true,
				Description: "Explicit DMS API host (e.g. 'https://<dwsu-domain>'). When set, the provider skips the control-plane DwsuModel lookup that would otherwise be used to resolve the host from dwsu_id. Useful for air-gapped / single-tenant / DMS-only deployments where the control-plane service is not reachable. Leave unset (default) for normal multi-tenant environments where the control plane is available.",
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

func (r *dwsuEntraIdConfig) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.put(ctx, req.Plan, &resp.Diagnostics, &resp.State)
}

func (r *dwsuEntraIdConfig) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.put(ctx, req.Plan, &resp.Diagnostics, &resp.State)
}

func (r *dwsuEntraIdConfig) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tfModel.EntraIdConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dmsHost := r.resolveDmsHost(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := common.CommonRetry(ctx, func() (*client.EntraIdConfig, error) {
		return r.client.GetEntraIdConfig(ctx, dmsHost, state.DwsuId.ValueString())
	})
	if err != nil {
		resp.Diagnostics.AddError("Error reading entraid-config",
			"GET /api/entraid-config failed for dwsu_id="+state.DwsuId.ValueString()+": "+err.Error())
		return
	}

	// Drift: backend has no config (externally deleted or never created).
	// Remove from state → next plan will show '+ create'.
	if cfg == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.TenantId = types.StringValue(cfg.TenantId)
	state.ClientId = types.StringValue(cfg.ClientId)
	state.TenantType = types.StringValue(cfg.TenantType)
	state.Enabled = types.BoolValue(cfg.Enabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *dwsuEntraIdConfig) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tfModel.EntraIdConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dmsHost := r.resolveDmsHost(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := common.CommonRetry(ctx, func() (*string, error) {
		if e := r.client.DeleteEntraIdConfig(ctx, dmsHost, state.DwsuId.ValueString()); e != nil {
			return nil, e
		}
		s := "ok"
		return &s, nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting entraid-config",
			"DELETE /api/entraid-config failed for dwsu_id="+state.DwsuId.ValueString()+": "+err.Error())
	}
	// No explicit state clear — framework removes the resource automatically on success.
}
func (r *dwsuEntraIdConfig) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("dwsu_id"), req, resp)
}

// resolveDmsHost returns the host to send /api/entraid-config requests to.
// If the user supplied dms_host explicitly, use it (skips control-plane lookup
// — useful for air-gapped / DMS-only deployments). Otherwise resolve via the
// control plane: GetDwsu(id) → Endpoints[type==openapi].URI.
func (r *dwsuEntraIdConfig) resolveDmsHost(ctx context.Context, m *tfModel.EntraIdConfigModel, diags *diag.Diagnostics) string {
	if !m.DmsHost.IsNull() && !m.DmsHost.IsUnknown() && m.DmsHost.ValueString() != "" {
		return m.DmsHost.ValueString()
	}
	return common.RouteDwsuOpenApiHost(ctx, m.DwsuId.ValueString(), r.client, diags)
}

// put applies the plan to the backend via PUT /api/entraid-config and writes
// the same plan into TF state on success. Shared by Create and Update because
// the backend endpoint is upsert-semantics.
func (r *dwsuEntraIdConfig) put(ctx context.Context, plan tfsdk.Plan, diags *diag.Diagnostics, state *tfsdk.State) {
	var m tfModel.EntraIdConfigModel
	diags.Append(plan.Get(ctx, &m)...)
	if diags.HasError() {
		return
	}

	dmsHost := r.resolveDmsHost(ctx, &m, diags)
	if diags.HasError() {
		return
	}

	_, err := common.CommonRetry(ctx, func() (*client.EntraIdConfig, error) {
		return r.client.PutEntraIdConfig(ctx, dmsHost, m.DwsuId.ValueString(), client.EntraIdConfig{
			TenantId:   m.TenantId.ValueString(),
			ClientId:   m.ClientId.ValueString(),
			TenantType: m.TenantType.ValueString(),
			Enabled:    m.Enabled.ValueBool(),
		})
	})
	if err != nil {
		diags.AddError("Error writing entraid-config",
			"PUT /api/entraid-config failed for dwsu_id="+m.DwsuId.ValueString()+": "+err.Error())
		return
	}
	diags.Append(state.Set(ctx, &m)...)
}
