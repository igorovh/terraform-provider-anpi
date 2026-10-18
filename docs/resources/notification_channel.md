---
page_title: "anpi_notification_channel Resource"
---

# anpi_notification_channel

A channel that sends alerts. Each kind reads only its own settings; the others stay empty.

## Example

```hcl
resource "anpi_notification_channel" "discord" {
  name        = "Discord #alerts"
  kind        = "discord"
  webhook_url = var.discord_webhook_url
}

resource "anpi_notification_channel" "mail" {
  name          = "On-call mail"
  kind          = "email"
  smtp_host     = "smtp.example.com"
  smtp_port     = "587"
  smtp_username = "anpi@example.com"
  smtp_password = var.smtp_password
  email_from    = "anpi@example.com"
  email_to      = "oncall@example.com, ops@example.com"
}
```

## Argument Reference

- `name` (String, required)
- `kind` (String, required): `discord`, `telegram`, `ntfy`, `email` or `webhook`.
- `active` (Boolean, optional): send alerts through this channel. Default `true`.
- `is_default` (Boolean, optional): pre-select the channel for monitors created in the panel. It does not affect `anpi_monitor`. Default `false`.

Settings per kind (all strings, default empty):

| Kind | Settings |
| --- | --- |
| `discord` | `webhook_url` (required, sensitive), `username` |
| `telegram` | `bot_token` (required, sensitive), `chat_id` (required), `api_base` |
| `ntfy` | `topic` (required), `server` (default ntfy.sh), `token` (sensitive), `priority` |
| `email` | `smtp_host`, `smtp_port`, `email_from`, `email_to` (required; recipients separated by commas), `smtp_security` (`starttls` when empty, `tls` or `none`), `smtp_username`, `smtp_password` (sensitive) |
| `webhook` | `url` (required, sensitive): receives a JSON POST for each event |

## Attribute Reference

- `id` (Number): channel ID.

## Import

```sh
terraform import anpi_notification_channel.discord 2
```
