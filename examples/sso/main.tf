resource "anpi_sso" "this" {
  enabled       = true
  base_url      = "https://status.example.com"
  issuer        = "https://id.example.com/realms/status"
  client_id     = "status"
  client_secret = var.client_secret
  required_role = "status-admin"
}

variable "client_secret" {
  type      = string
  sensitive = true
}
