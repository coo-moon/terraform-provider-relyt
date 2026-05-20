package model

import "github.com/hashicorp/terraform-plugin-framework/types"

type EntraIdConfigModel struct {
	DwsuId     types.String `tfsdk:"dwsu_id"`
	DmsHost    types.String `tfsdk:"dms_host"`
	TenantId   types.String `tfsdk:"tenant_id"`
	ClientId   types.String `tfsdk:"client_id"`
	TenantType types.String `tfsdk:"tenant_type"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}
