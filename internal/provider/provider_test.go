package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testProviders = map[string]func() (tfprotov6.ProviderServer, error){
	"anpi": providerserver.NewProtocol6WithError(New("test")()),
}

// Acceptance tests need TF_ACC=1 and a disposable anpi in ANPI_URL with a token in ANPI_TOKEN.
func preCheck(t *testing.T) {
	if os.Getenv("ANPI_URL") == "" || os.Getenv("ANPI_TOKEN") == "" {
		t.Fatal("ANPI_URL and ANPI_TOKEN must point at a disposable anpi instance")
	}
}

func TestHeadersRoundTrip(t *testing.T) {
	ctx := context.Background()
	m := types.MapValueMust(types.StringType, map[string]attr.Value{
		"X-Token":       types.StringValue("abc"),
		"Authorization": types.StringValue("Bearer x:y"),
	})
	text, diags := headersText(ctx, m)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if text != "Authorization: Bearer x:y\nX-Token: abc" {
		t.Fatalf("headers are sorted lines, got %q", text)
	}
	if !headersMap(text + "\n").Equal(m) {
		t.Fatal("parsing the stored text gives the same map back")
	}
}

func TestSameInstantKeepsTheConfiguredOffset(t *testing.T) {
	cfg := types.StringValue("2030-05-02T00:00:00+02:00")
	if got := sameInstant(cfg, "2030-05-01T22:00:00Z"); !got.Equal(cfg) {
		t.Fatalf("same instant should keep the configured spelling, got %s", got)
	}
	if got := sameInstant(cfg, "2030-05-01T23:00:00Z"); got.ValueString() != "2030-05-01T23:00:00Z" {
		t.Fatalf("a different instant is drift, got %s", got)
	}
}

const fullConfig = `
resource "anpi_group" "product" {
  name = "Product %[1]s"
}

resource "anpi_notification_channel" "hook" {
  name = "hook %[1]s"
  kind = "webhook"
  url  = "https://hooks.example.com/anpi"
}

resource "anpi_monitor" "api" {
  name            = "API %[1]s"
  kind            = "http"
  target          = "https://example.com/health"
  interval_s      = 300
  active          = false
  public          = true
  group_id        = anpi_group.product.id
  channel_ids     = [anpi_notification_channel.hook.id]
  headers         = { "X-Probe" = "anpi" }
  content_kind    = "contains"
  content_value   = "ok"
}

resource "anpi_monitor" "backup" {
  name      = "Backup %[1]s"
  kind      = "push"
  parent_id = anpi_monitor.api.id
  group_id  = anpi_group.product.id
}

resource "anpi_monitor" "db" {
  name   = "DB %[1]s"
  kind   = "tcp"
  target = "db.internal"
  port   = 5432
  active = false
}

resource "anpi_maintenance" "upgrade" {
  title       = "Upgrade %[1]s"
  starts_at   = "2030-05-02T00:00:00+02:00"
  ends_at     = "2030-05-02T01:00:00+02:00"
  monitor_ids = [anpi_monitor.api.id, anpi_monitor.db.id]
}
`

func TestAccFullConfiguration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(fullConfig, "v1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anpi_monitor.api", "retry_interval_s", "60"),
					resource.TestCheckResourceAttr("anpi_monitor.api", "headers.X-Probe", "anpi"),
					resource.TestCheckResourceAttr("anpi_monitor.api", "channel_ids.#", "1"),
					resource.TestCheckResourceAttrPair("anpi_monitor.api", "group_id", "anpi_group.product", "id"),
					resource.TestCheckResourceAttrSet("anpi_monitor.backup", "push_token"),
					resource.TestMatchResourceAttr("anpi_monitor.backup", "push_url", regexp.MustCompile(`/api/push/.+`)),
					resource.TestCheckNoResourceAttr("anpi_monitor.api", "port"),
					resource.TestCheckResourceAttr("anpi_maintenance.upgrade", "starts_at", "2030-05-02T00:00:00+02:00"),
					resource.TestCheckResourceAttr("anpi_maintenance.upgrade", "monitor_ids.#", "2"),
				),
			},
			{ResourceName: "anpi_monitor.api", ImportState: true, ImportStateVerify: true},
			{ResourceName: "anpi_monitor.backup", ImportState: true, ImportStateVerify: true},
			{ResourceName: "anpi_group.product", ImportState: true, ImportStateVerify: true},
			{ResourceName: "anpi_notification_channel.hook", ImportState: true, ImportStateVerify: true},
			{ResourceName: "anpi_maintenance.upgrade", ImportState: true, ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"starts_at", "ends_at"}},
			{
				Config: fmt.Sprintf(fullConfig, "v2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anpi_monitor.api", "name", "API v2"),
					resource.TestCheckResourceAttr("anpi_group.product", "name", "Product v2"),
					resource.TestCheckResourceAttr("anpi_maintenance.upgrade", "title", "Upgrade v2"),
				),
			},
		},
	})
}

func TestAccSettingsOfOtherKindsAreRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testProviders,
		Steps: []resource.TestStep{
			{
				Config: `resource "anpi_monitor" "x" {
  name = "x"
  kind = "http"
  target = "https://example.com"
  port = 443
}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("only used by `tcp`"),
			},
			{
				Config: `resource "anpi_monitor" "x" {
  name = "x"
  kind = "tcp"
  target = "db"
}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("need a `port`"),
			},
			{
				Config: `resource "anpi_maintenance" "x" {
  title = "x"
  starts_at = "2030-01-01T10:00:00Z"
  ends_at = "2030-01-01T09:00:00Z"
  all_monitors = true
}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("must be after"),
			},
			{
				Config: `resource "anpi_monitor" "x" {
  name = "x"
  kind = "http"
  target = "not a url"
}`,
				ExpectError: regexp.MustCompile("URL must be a full address"),
			},
		},
	})
}
