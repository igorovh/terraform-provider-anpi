resource "anpi_branding" "this" {
  site_name       = "ACME"
  status_title    = "ACME status"
  default_theme   = "dark"
  show_powered_by = false
  custom_css      = file("${path.module}/theme.css")
  logo            = filebase64("${path.module}/logo.svg")
}
