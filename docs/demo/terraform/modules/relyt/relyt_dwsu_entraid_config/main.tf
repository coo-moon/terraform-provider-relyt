resource "relyt_dwsu_entraid_config" "this" {
  dwsu_id     = var.dwsu_id
  tenant_id   = var.tenant_id
  client_id   = var.client_id
  tenant_type = var.tenant_type
  enabled     = var.enabled
}
