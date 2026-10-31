terraform {
  required_providers {
    anpi = {
      source  = "igorovh/anpi"
      version = "~> 0.2"
    }
  }
}

# url and token can also come from ANPI_URL and ANPI_TOKEN.
provider "anpi" {
  url   = var.anpi_url
  token = var.anpi_token
}

variable "anpi_url" {
  type = string
}

variable "anpi_token" {
  type      = string
  sensitive = true
}

variable "discord_webhook_url" {
  type      = string
  sensitive = true
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
  interval_s  = 30
  public      = true
  public_name = "Public API"
  group_id    = anpi_group.product.id
  channel_ids = [anpi_notification_channel.discord.id]

  headers = {
    "X-Health-Check" = "anpi"
  }

  content_kind     = "json_path"
  content_value    = "$.status"
  content_expected = "ok"
}

# A sub-monitor: it rolls up into the API row on the dashboard and the status page.
resource "anpi_monitor" "api_db" {
  name        = "Database"
  kind        = "tcp"
  target      = "db.internal"
  port        = 5432
  parent_id   = anpi_monitor.api.id
  group_id    = anpi_group.product.id
  channel_ids = [anpi_notification_channel.discord.id]
}

resource "anpi_monitor" "nightly_backup" {
  name              = "Nightly backup"
  kind              = "push"
  interval_s        = 86400
  failure_threshold = 1
  channel_ids       = [anpi_notification_channel.discord.id]
}

resource "anpi_maintenance" "db_upgrade" {
  title       = "Database upgrade"
  starts_at   = "2030-05-02T00:00:00+02:00"
  ends_at     = "2030-05-02T01:00:00+02:00"
  monitor_ids = [anpi_monitor.api.id, anpi_monitor.api_db.id]
}

output "backup_push_url" {
  description = "Call this from the backup job when it finishes, e.g. curl -fsS \"$URL?status=up\"."
  value       = anpi_monitor.nightly_backup.push_url
  sensitive   = true
}
