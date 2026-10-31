# anpi_sso

Manages the instance's SSO configuration. Requires anpi 0.4.0 and a token with the SSO permission. This resource controls panel sign-in. Terraform state contains `client_secret`, even though Terraform marks it sensitive.

```hcl
resource "anpi_sso" "this" {
  enabled       = true
  base_url      = "https://status.example.com"
  issuer        = "https://id.example.com/realms/status"
  client_id     = "status"
  client_secret = var.client_secret
  required_role = "status-admin"
  scopes        = "openid profile email"
}
```

`base_url`, `issuer` and `client_id` are required. `enabled` defaults to `true`, `client_secret` and `required_role` to an empty string, and `scopes` to `openid profile email`. When SSO is configured through `ANPI_OIDC_*`, the API cannot change it. Deleting the resource disables SSO.

Import with any ID, for example `terraform import anpi_sso.this sso`.
