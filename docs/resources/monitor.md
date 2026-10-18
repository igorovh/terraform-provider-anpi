---
page_title: "anpi_monitor Resource"
---

# anpi_monitor

A monitor. Saving it starts or restarts its checks, just like saving the panel form.

## Example

```hcl
resource "anpi_monitor" "api" {
  name        = "API"
  kind        = "http"
  target      = "https://api.example.com/health"
  interval_s  = 30
  public      = true
  group_id    = anpi_group.product.id
  channel_ids = [anpi_notification_channel.discord.id]

  headers = {
    Authorization = "Bearer ${var.health_token}"
  }

  content_kind     = "json_path"
  content_value    = "$.status"
  content_expected = "ok"
}

resource "anpi_monitor" "db" {
  name      = "Database"
  kind      = "tcp"
  target    = "db.internal"
  port      = 5432
  parent_id = anpi_monitor.api.id
  group_id  = anpi_group.product.id
}

resource "anpi_monitor" "backup" {
  name       = "Nightly backup"
  kind       = "push"
  interval_s = 86400
}

output "backup_push_url" {
  value     = anpi_monitor.backup.push_url
  sensitive = true
}
```

## Argument Reference

Required:

- `name` (String): name in the panel and alerts, up to 100 characters.
- `kind` (String): `http`, `websocket`, `tcp`, `ping`, `dns`, `push` or `aggregate`.

Optional:

- `target` (String): URL for `http` (`https://…`) and `websocket` (`wss://…`), host for `tcp` and `ping`, domain for `dns`. Required for those kinds; not allowed for `push` and `aggregate`.
- `port` (Number): port of a `tcp` monitor. Required for `tcp`, not allowed for other kinds.
- `method` (String): HTTP method, `http` only. Default `GET`.
- `headers` (Map of String): request headers for `http` and `websocket`.
- `body` (String): request body for `http`.
- `interval_s` (Number): seconds between checks, 5 to 604800. Default `60`. For `push`, how often the push is expected.
- `retry_interval_s` (Number): seconds between checks while failing, 5 to 604800. Default `60`.
- `timeout_s` (Number): 1 to 300. Default `30`.
- `failure_threshold` (Number): failures in a row before the monitor is down, 1 to 100. Default `3`.
- `expected_status` (String): accepted status codes, e.g. `200-299` or `200,301`. Default `200-299`.
- `ip_family` (String): `auto`, `v4` or `v6`. Default `auto`.
- `follow_redirects` (Boolean): default `true`.
- `ignore_tls` (Boolean): accept invalid certificates. Default `false`.
- `content_kind` (String): response rule for `http` and `websocket`: `none`, `contains`, `not_contains`, `regex` or `json_path`. Default `none`.
- `content_value` (String): keyword, regex or JSONPath for the rule.
- `content_expected` (String): expected value of a `json_path` rule, or the expected answer of a `dns` monitor.
- `ssl_warn_days` (Number): days before certificate expiry to warn, 0 to 365. Default `14`.
- `dns_record_type` (String): `A`, `AAAA`, `CNAME`, `MX`, `NS`, `TXT`, `SOA` or `CAA`. Default `A`.
- `dns_server` (String): resolver IP, optionally `ip:port`. Empty uses the system resolver.
- `active` (Boolean): `false` pauses the monitor. Default `true`.
- `public` (Boolean): show on the status page. Default `false`.
- `public_name` (String): name on the status page instead of `name`.
- `group_id` (Number): ID of an `anpi_group`.
- `parent_id` (Number): ID of the parent monitor. Only one level of nesting is allowed, and a monitor with sub-monitors cannot get a parent.
- `channel_ids` (Set of Number): IDs of `anpi_notification_channel`s that alert for this monitor. Default: none.

## Attribute Reference

- `id` (Number): monitor ID, as in `/admin/monitors/<id>`.
- `push_token` (String, sensitive): token of a `push` monitor.
- `push_url` (String, sensitive): URL a `push` monitor expects, `<url>/api/push/<token>`. Add `?status=up&msg=OK` or `?status=down`.

## Import

```sh
terraform import anpi_monitor.api 12
```
