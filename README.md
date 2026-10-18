# terraform-provider-anpi

Terraform provider for [anpi](https://github.com/igorovh/anpi), a small self-hosted uptime monitor. It manages monitors, groups, notification channels and maintenance windows through the anpi REST API, so the whole monitoring setup can live in Git.

Needs anpi with the REST API (`/api/v1`), the first release after 0.2.2.

## Quick start

1. In anpi, open **Settings → API tokens** and create a token, or run `anpi api-token terraform` on the server. The token is shown once.
2. Configure the provider:

```hcl
terraform {
  required_providers {
    anpi = {
      source  = "igorovh/anpi"
      version = "~> 0.1"
    }
  }
}

provider "anpi" {
  url = "https://status.example.com" # or ANPI_URL
  # token comes from ANPI_TOKEN
}

resource "anpi_group" "product" {
  name = "Product"
}

resource "anpi_notification_channel" "discord" {
  name        = "Discord #alerts"
  kind        = "discord"
  webhook_url = var.discord_webhook_url
}

resource "anpi_monitor" "api" {
  name        = "API"
  kind        = "http"
  target      = "https://api.example.com/health"
  public      = true
  group_id    = anpi_group.product.id
  channel_ids = [anpi_notification_channel.discord.id]
}
```

3. Run `terraform init` and `terraform apply`.

See [`examples/basic`](examples/basic/main.tf) for sub-monitors, push monitors, response rules and maintenance windows.

## Resources

| Resource | What it manages |
| --- | --- |
| [`anpi_monitor`](docs/resources/monitor.md) | HTTP(S), WebSocket, TCP, ping, DNS, push and aggregate monitors |
| [`anpi_group`](docs/resources/group.md) | Sections of the dashboard and the status page |
| [`anpi_notification_channel`](docs/resources/notification_channel.md) | Discord, Telegram, ntfy, e-mail and webhook alerts |
| [`anpi_maintenance`](docs/resources/maintenance.md) | Maintenance windows |

Existing objects can be imported by the numeric ID from the panel URL, e.g. `terraform import anpi_monitor.api 12`.

## Good to know

- Tokens belong to the instance, not to a user. Revoke them under **Settings → API tokens**.
- Changes made in the panel show up as drift in the next `terraform plan`.
- `anpi_monitor.channel_ids` is explicit: channels marked as default in the panel are not added to monitors created by Terraform.
- Each monitor kind uses only some settings. The provider rejects settings the server would ignore, for example `port` on an `http` monitor, so the plan stays clean.
- State contains secrets: channel webhook URLs, tokens and SMTP passwords, and push tokens. Keep the state private.
- Group order and sort order between monitors are not managed; drag monitors in the panel if you need a specific order.

## Development

```sh
make build     # bin/terraform-provider-anpi
make test      # unit tests
make lint
```

Acceptance tests create and delete real objects, so run them against a throwaway anpi:

```sh
export ANPI_DATA_DIR=$(mktemp -d) ANPI_BIND=127.0.0.1:3917
anpi api-token acc > token && anpi &      # from the anpi repository: cargo run -- api-token acc
ANPI_URL=http://127.0.0.1:3917 ANPI_TOKEN=$(cat token) make testacc
```

To try a local build in Terraform, point `dev_overrides` in `~/.terraformrc` at this repository's `bin` directory:

```hcl
provider_installation {
  dev_overrides {
    "igorovh/anpi" = "/path/to/terraform-anpi/bin"
  }
  direct {}
}
```

Releases are built by GoReleaser when a `v*` tag is pushed; the workflow needs the `GPG_PRIVATE_KEY` and `PASSPHRASE` secrets for the Terraform Registry signature.

## License

[MIT](LICENSE)
