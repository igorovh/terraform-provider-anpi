package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/igorovh/terraform-provider-anpi/internal/client"
)

var _ resource.ResourceWithImportState = (*ssoResource)(nil)

type ssoResource struct{ client *client.Client }
type ssoModel struct {
	ID           types.String `tfsdk:"id"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	BaseURL      types.String `tfsdk:"base_url"`
	Issuer       types.String `tfsdk:"issuer"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	RequiredRole types.String `tfsdk:"required_role"`
	Scopes       types.String `tfsdk:"scopes"`
}

func NewSSOResource() resource.Resource { return &ssoResource{} }
func (r *ssoResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso"
}
func (r *ssoResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages the instance's single sign-on settings. Requires anpi 0.4.0 and an SSO-enabled token.", Attributes: map[string]schema.Attribute{
		"id":            schema.StringAttribute{Computed: true},
		"enabled":       schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
		"base_url":      schema.StringAttribute{Required: true},
		"issuer":        schema.StringAttribute{Required: true},
		"client_id":     schema.StringAttribute{Required: true},
		"client_secret": schema.StringAttribute{Optional: true, Computed: true, Sensitive: true, Default: stringdefault.StaticString("")},
		"required_role": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
		"scopes":        schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("openid profile email")},
	}}
}
func (r *ssoResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}
func (m *ssoModel) toAPI() *client.SSO {
	return &client.SSO{Enabled: m.Enabled.ValueBool(), BaseURL: m.BaseURL.ValueString(), Issuer: m.Issuer.ValueString(), ClientID: m.ClientID.ValueString(), ClientSecret: m.ClientSecret.ValueString(), RequiredRole: m.RequiredRole.ValueString(), Scopes: m.Scopes.ValueString()}
}
func (m *ssoModel) fromAPI(s *client.SSO) {
	m.ID = types.StringValue("sso")
	m.Enabled = types.BoolValue(s.Enabled)
	m.BaseURL = types.StringValue(s.BaseURL)
	m.Issuer = types.StringValue(s.Issuer)
	m.ClientID = types.StringValue(s.ClientID)
	m.ClientSecret = types.StringValue(s.ClientSecret)
	m.RequiredRole = types.StringValue(s.RequiredRole)
	m.Scopes = types.StringValue(s.Scopes)
}
func (r *ssoResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ssoModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.PutSSO(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Creating anpi SSO failed", err.Error())
		return
	}
	plan.fromAPI(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *ssoResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ssoModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetSSO(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Reading anpi SSO failed", err.Error())
		return
	}
	state.fromAPI(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *ssoResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ssoModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.PutSSO(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Updating anpi SSO failed", err.Error())
		return
	}
	plan.fromAPI(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *ssoResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.client.DeleteSSO(ctx); err != nil {
		resp.Diagnostics.AddError("Deleting anpi SSO failed", err.Error())
	}
}
func (r *ssoResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), "sso")...)
}
