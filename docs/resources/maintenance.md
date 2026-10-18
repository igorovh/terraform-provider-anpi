---
page_title: "anpi_maintenance Resource"
---

# anpi_maintenance

A maintenance window. While it is active, the covered monitors show as maintenance and outages raise no alerts.

## Example

```hcl
resource "anpi_maintenance" "db_upgrade" {
  title       = "Database upgrade"
  starts_at   = "2030-05-02T00:00:00+02:00"
  ends_at     = "2030-05-02T01:00:00+02:00"
  monitor_ids = [anpi_monitor.api.id, anpi_monitor.db.id]
}
```

## Argument Reference

- `title` (String, required)
- `starts_at` (String, required): RFC 3339 time with an offset, e.g. `2030-05-01T22:00:00Z`. anpi stores it in UTC; the provider keeps your spelling as long as it is the same moment.
- `ends_at` (String, required): RFC 3339 time after `starts_at`.
- `all_monitors` (Boolean, optional): cover every monitor, including ones created later. Default `false`.
- `monitor_ids` (Set of Number, optional): covered monitors. Required unless `all_monitors` is true.

Times are stored to the millisecond; fractions below that are dropped.

## Attribute Reference

- `id` (Number): maintenance window ID.

## Import

```sh
terraform import anpi_maintenance.db_upgrade 4
```

After an import, `starts_at` and `ends_at` are in UTC (`…Z`). If your configuration writes the same moment with another offset, the first plan shows an in-place update; applying it changes nothing in anpi.
