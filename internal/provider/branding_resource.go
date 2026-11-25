package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/igorovh/terraform-provider-anpi/internal/client"
)

var _ resource.ResourceWithImportState = (*brandingResource)(nil)

type brandingResource struct{ client *client.Client }
type brandingModel struct {
	ID                types.String `tfsdk:"id"`
	SiteName          types.String `tfsdk:"site_name"`
	StatusTitle       types.String `tfsdk:"status_title"`
	StatusDescription types.String `tfsdk:"status_description"`
	DefaultTheme      types.String `tfsdk:"default_theme"`
	ShowPoweredBy     types.Bool   `tfsdk:"show_powered_by"`
	CustomCSS         types.String `tfsdk:"custom_css"`
	Logo              types.String `tfsdk:"logo"`
	Favicon           types.String `tfsdk:"favicon"`
}

func NewBrandingResource() resource.Resource { return &brandingResource{} }
func (r *brandingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branding"
}
func (r *brandingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	image := func(what string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, MarkdownDescription: what + " as base64, e.g. `filebase64(\"logo.svg\")`. PNG, JPEG, WebP, GIF, ICO or SVG, up to 512 KB."}
	}
	resp.Schema = schema.Schema{MarkdownDescription: "Manages the look of the panel and the status page. Requires anpi 0.5.0. Destroying it restores the defaults.", Attributes: map[string]schema.Attribute{
		"id":                 schema.StringAttribute{Computed: true},
		"site_name":          schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("anpi")},
		"status_title":       schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("Service status")},
		"status_description": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
		"default_theme":      schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("system"), Validators: []validator.String{stringvalidator.OneOf("system", "light", "dark")}},
		"show_powered_by":    schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
		"custom_css":         schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""), MarkdownDescription: "Stylesheet loaded after the built-in one, e.g. `file(\"theme.css\")`. Up to 512 KB."},
		"logo":               image("Logo"),
		"favicon":            image("Favicon; without it the logo is used"),
	}}
}
func (r *brandingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}
func (m *brandingModel) toAPI() *client.Branding {
	return &client.Branding{SiteName: m.SiteName.ValueString(), StatusTitle: m.StatusTitle.ValueString(), StatusDescription: m.StatusDescription.ValueString(), DefaultTheme: m.DefaultTheme.ValueString(), ShowPoweredBy: m.ShowPoweredBy.ValueBool(), CustomCSS: m.CustomCSS.ValueString(), Logo: m.Logo.ValueStringPointer(), Favicon: m.Favicon.ValueStringPointer()}
}

// trimmed keeps the configured spelling when anpi only trimmed it and turned CRLF into LF, e.g. a file() from a Windows checkout.
func trimmed(prev types.String, got string) types.String {
	if !prev.IsNull() && !prev.IsUnknown() && strings.TrimSpace(strings.ReplaceAll(prev.ValueString(), "\r\n", "\n")) == got {
		return prev
	}
	return types.StringValue(got)
}

// sameImage keeps configured base64 that anpi returned without its line breaks.
func sameImage(prev types.String, got *string) types.String {
	if got != nil && !prev.IsNull() && !prev.IsUnknown() && strings.Join(strings.Fields(prev.ValueString()), "") == *got {
		return prev
	}
	return types.StringPointerValue(got)
}

func (m *brandingModel) fromAPI(b *client.Branding) {
	m.ID = types.StringValue("branding")
	m.SiteName = trimmed(m.SiteName, b.SiteName)
	m.StatusTitle = trimmed(m.StatusTitle, b.StatusTitle)
	m.StatusDescription = trimmed(m.StatusDescription, b.StatusDescription)
	m.DefaultTheme = types.StringValue(b.DefaultTheme)
	m.ShowPoweredBy = types.BoolValue(b.ShowPoweredBy)
	m.CustomCSS = trimmed(m.CustomCSS, b.CustomCSS)
	m.Logo = sameImage(m.Logo, b.Logo)
	m.Favicon = sameImage(m.Favicon, b.Favicon)
}
func (r *brandingResource) put(ctx context.Context, plan *brandingModel) error {
	out, err := r.client.PutBranding(ctx, plan.toAPI())
	if err != nil {
		return err
	}
	plan.fromAPI(out)
	return nil
}
func (r *brandingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan brandingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.put(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Creating anpi branding failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *brandingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state brandingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetBranding(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Reading anpi branding failed", err.Error())
		return
	}
	state.fromAPI(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *brandingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan brandingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.put(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Updating anpi branding failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *brandingResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.client.DeleteBranding(ctx); err != nil {
		resp.Diagnostics.AddError("Deleting anpi branding failed", err.Error())
	}
}
func (r *brandingResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), "branding")...)
}
