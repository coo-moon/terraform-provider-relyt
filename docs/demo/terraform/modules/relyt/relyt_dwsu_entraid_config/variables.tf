variable "dwsu_id" {
  type        = string
  description = "The ID of the DWSU to attach SSO config to."
}

variable "tenant_id" {
  type        = string
  description = "Azure AD tenant (directory) GUID."
}

variable "client_id" {
  type        = string
  description = "Azure AD application (client) GUID."
}

variable "tenant_type" {
  type        = string
  default     = "single"
  description = "'single' or 'multi'. Default 'single'."
}

variable "enabled" {
  type        = bool
  default     = true
  description = "Whether SSO is active. Default true."
}
