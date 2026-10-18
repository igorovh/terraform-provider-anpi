# AGENTS.md

## Repo Shape
- Go Terraform provider (terraform-plugin-framework). Entrypoint `main.go`, address `registry.terraform.io/igorovh/anpi`, type `anpi`; the GitHub repo must be named `terraform-provider-anpi` for the Registry.
- `internal/client` is the HTTP client for the anpi REST API (`/api/v1`, bearer token); `internal/provider` holds the provider and the four resources: `anpi_monitor`, `anpi_group`, `anpi_notification_channel`, `anpi_maintenance`. No data sources.
- The API lives in the anpi repo (`src/web/api.rs`, documented in `docs/api.md`). Keep field names and defaults in sync with it.
- Docs are handwritten in `README.md` and `docs/`; `make docs` only prints a reminder.

## Commands
- Build: `make build`. Unit tests: `make test`. Vet: `make lint`. Format: `make fmt`.
- Acceptance tests (`make testacc`) create and delete real objects: run them only against a throwaway anpi with `ANPI_URL` and `ANPI_TOKEN` set (see README).

## Behavior To Preserve
- Resource IDs are anpi's numeric IDs (`types.Int64`); import takes the number from the panel URL.
- Every optional attribute has a static default equal to the server default, so plans stay clean and PUT sends the full object.
- `anpi_monitor.ValidateConfig` rejects settings the server drops for a kind (`port` off `tcp`, `method` off `http`, response rules off `http`/`websocket`); without it the server normalises them and Terraform reports an inconsistent result.
- `headers` is a map; it is sent as sorted `Name: value` lines because anpi stores headers as text.
- `channel_ids` defaults to an empty set; the provider always sends it, so panel "default" channels are never added implicitly.
- Maintenance times are RFC 3339; anpi returns UTC, and `sameInstant` keeps the configured spelling when the moment is unchanged.
- 404 on read removes the resource from state; 404 on delete counts as success.

## Releases
- `.goreleaser.yml` and `.github/workflows/release.yml` build signed zips on `v*` tags (secrets `GPG_PRIVATE_KEY`, `PASSPHRASE`). Keep `version` in `main.go` aligned with examples.
