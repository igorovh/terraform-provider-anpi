# anpi_branding

Manages the look of the panel and the status page: names, logo, favicon, default theme, footer credit and a custom stylesheet. Requires anpi 0.5.0. Any API token can use it.

```hcl
resource "anpi_branding" "this" {
  site_name          = "ACME"
  status_title       = "ACME status"
  status_description = "Live status of ACME services."
  default_theme      = "dark"
  show_powered_by    = false
  custom_css         = file("${path.module}/theme.css")
  logo               = filebase64("${path.module}/logo.svg")
  favicon            = filebase64("${path.module}/favicon.png")
}
```

| Argument | Default | |
| --- | --- | --- |
| `site_name` | `anpi` | up to 40 characters; header, footer and browser tab |
| `status_title` | `Service status` | up to 100 characters |
| `status_description` | `""` | up to 1000 characters |
| `default_theme` | `system` | `system`, `light` or `dark`; a visitor's own choice still wins |
| `show_powered_by` | `true` | the "powered by anpi" footer credit |
| `custom_css` | `""` | up to 512 KB, loaded after the built-in stylesheet on every page |
| `logo` | none | base64 image, use `filebase64()`; PNG, JPEG, WebP, GIF, ICO or SVG up to 512 KB |
| `favicon` | none | same as `logo`; without it the logo is used |

The resource replaces the whole branding, so settings left out return to their defaults, and removing `logo` or `favicon` deletes the image. Destroying the resource restores all defaults. anpi trims surrounding whitespace, such as the trailing newline `file()` adds; the provider ignores that difference.

Custom CSS can override the colour variables (`--bg`, `--panel`, `--ink`, `--muted`, `--line`, `--up`, `--down`, `--pending`, `--maint`, …) for each theme:

```css
:root[data-theme="dark"]  { --bg: #0b1020; --panel: #111833; --up: #4ade80; }
:root[data-theme="light"] { --bg: #f7f8fc; --panel: #ffffff; --up: #15803d; }
```

It can also restyle any element. The page's Content Security Policy only allows images and fonts from the anpi site or `data:` URLs, so embed fonts as `data:` URLs. Images are stored in Terraform state as base64.

Import with any ID, for example `terraform import anpi_branding.this branding`.
