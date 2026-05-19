# Reference an existing DWSU
data "relyt_dwsus" "all" {}

resource "relyt_dwsu_entraid_config" "sso" {
  dwsu_id   = data.relyt_dwsus.all.dwsu_list[0].id
  tenant_id = "00000000-0000-0000-0000-000000000000" # Azure portal -> Directory (tenant) ID
  client_id = "00000000-0000-0000-0000-000000000000" # Azure portal -> Application (client) ID

  # The two below are optional and default to "single" and true.
  tenant_type = "single" # "single" | "multi"
  enabled     = true
}
