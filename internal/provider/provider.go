package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/igorovh/terraform-provider-anpi/internal/client"
)

var _ provider.Provider = (*anpiProvider)(nil)

type (
	validatorInt64  = validator.Int64
	validatorString = validator.String
)

type anpiProvider struct {
	version string
}

type providerModel struct {
	URL   types.String `tfsdk:"url"`
	Token types.String `tfsdk:"token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &anpiProvider{version: version}
	}
}

func (p *anpiProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "anpi"
	resp.Version = p.version
}

func (p *anpiProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages monitors, groups, notification channels and maintenance windows of an [anpi](https://github.com/igorovh/anpi) uptime monitor through its REST API.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Base URL of the anpi instance, e.g. `https://status.example.com`. Can also be set with `ANPI_URL`.",
			},
			"token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "API token created under **Settings → API tokens** or with `anpi api-token <name>`. Can also be set with `ANPI_TOKEN`.",
			},
		},
	}
}

func (p *anpiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Unknown values come from resources not created yet; Terraform configures the provider again later.
	if cfg.URL.IsUnknown() || cfg.Token.IsUnknown() {
		return
	}
	url := os.Getenv("ANPI_URL")
	if !cfg.URL.IsNull() {
		url = cfg.URL.ValueString()
	}
	token := os.Getenv("ANPI_TOKEN")
	if !cfg.Token.IsNull() {
		token = cfg.Token.ValueString()
	}
	if url == "" {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Missing anpi URL", "Set `url` in the provider block or the ANPI_URL environment variable.")
	}
	if token == "" {
		resp.Diagnostics.AddAttributeError(path.Root("token"), "Missing anpi API token", "Set `token` in the provider block or the ANPI_TOKEN environment variable. Create one under Settings → API tokens.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	c := client.New(url, token, "terraform-provider-anpi/"+p.version)
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *anpiProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMonitorResource,
		NewGroupResource,
		NewNotificationChannelResource,
		NewMaintenanceResource,
		NewSSOResource,
	}
}

func (p *anpiProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// clientFrom reads the client set by Configure; it is nil while the provider is not configured yet.
func clientFrom(data any, diags *diag.Diagnostics) *client.Client {
	if data == nil {
		return nil
	}
	c, ok := data.(*client.Client)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", data))
		return nil
	}
	return c
}

func importID(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected the numeric anpi ID (shown in the panel URL), got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func int64Ptr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := v.ValueInt64()
	return &n
}

func int64Set(ctx context.Context, ids []int64) (types.Set, diag.Diagnostics) {
	if ids == nil {
		ids = []int64{}
	}
	return types.SetValueFrom(ctx, types.Int64Type, ids)
}

func setInt64s(ctx context.Context, s types.Set) ([]int64, diag.Diagnostics) {
	ids := []int64{}
	if s.IsNull() || s.IsUnknown() {
		return ids, nil
	}
	diags := s.ElementsAs(ctx, &ids, false)
	return ids, diags
}
