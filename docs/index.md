---
page_title: "anpi Provider"
---

# anpi Provider

Manages an [anpi](https://github.com/igorovh/anpi) uptime monitor through its REST API: monitors, groups, notification channels and maintenance windows.

## Example

```hcl
provider "anpi" {
  url   = "https://status.example.com"
  token = var.anpi_token
}
```

## Authentication

Create an API token in anpi under **Settings → API tokens**, or with `anpi api-token <name>` on the server. Tokens belong to the instance and give full access to the configuration. Revoke a token in the same place.

## Argument Reference

- `url` (String, optional): base URL of anpi, e.g. `https://status.example.com`. Defaults to the `ANPI_URL` environment variable.
- `token` (String, optional, sensitive): API token. Defaults to the `ANPI_TOKEN` environment variable.

Both must be set in the block or in the environment.
