package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/igorovh/terraform-provider-anpi/internal/client"
)

var (
	_ resource.Resource                   = (*maintenanceResource)(nil)
	_ resource.ResourceWithConfigure      = (*maintenanceResource)(nil)
	_ resource.ResourceWithImportState    = (*maintenanceResource)(nil)
	_ resource.ResourceWithValidateConfig = (*maintenanceResource)(nil)
)

type maintenanceResource struct {
	client *client.Client
}

type maintenanceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Title       types.String `tfsdk:"title"`
	StartsAt    types.String `tfsdk:"starts_at"`
	EndsAt      types.String `tfsdk:"ends_at"`
	AllMonitors types.Bool   `tfsdk:"all_monitors"`
	MonitorIDs  types.Set    `tfsdk:"monitor_ids"`
}

func NewMaintenanceResource() resource.Resource {
	return &maintenanceResource{}
}

func (r *maintenanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance"
}

type rfc3339Validator struct{}

func (rfc3339Validator) Description(context.Context) string {
	return "must be an RFC 3339 time such as 2026-05-01T22:00:00Z"
}

func (v rfc3339Validator) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }

func (v rfc3339Validator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := time.Parse(time.RFC3339, req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid time", fmt.Sprintf("%q %s.", req.ConfigValue.ValueString(), v.Description(ctx)))
	}
}

func (r *maintenanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A maintenance window. During it, checks of the selected monitors keep running but outages raise no alerts and show as maintenance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{Computed: true, MarkdownDescription: "Numeric maintenance window ID.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"title": schema.StringAttribute{Required: true, MarkdownDescription: "Title shown in the panel.",
				Validators: []validatorString{stringvalidator.LengthAtLeast(1)}},
			"starts_at": schema.StringAttribute{Required: true, MarkdownDescription: "Start as an RFC 3339 time, e.g. `2026-05-01T22:00:00Z` or `2026-05-02T00:00:00+02:00`.",
				Validators: []validatorString{rfc3339Validator{}}},
			"ends_at": schema.StringAttribute{Required: true, MarkdownDescription: "End as an RFC 3339 time; must be after `starts_at`.",
				Validators: []validatorString{rfc3339Validator{}}},
			"all_monitors": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				MarkdownDescription: "Cover every monitor, including ones created later."},
			"monitor_ids": schema.SetAttribute{ElementType: types.Int64Type, Optional: true, Computed: true,
				Default:             setdefault.StaticValue(types.SetValueMust(types.Int64Type, []attr.Value{})),
				MarkdownDescription: "IDs of the covered monitors; required unless `all_monitors` is true."},
		},
	}
}

func (r *maintenanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *maintenanceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var m maintenanceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() || m.StartsAt.IsUnknown() || m.EndsAt.IsUnknown() {
		return
	}
	start, err1 := time.Parse(time.RFC3339, m.StartsAt.ValueString())
	end, err2 := time.Parse(time.RFC3339, m.EndsAt.ValueString())
	if err1 == nil && err2 == nil && !end.After(start) {
		resp.Diagnostics.AddAttributeError(path.Root("ends_at"), "Invalid maintenance window", "`ends_at` must be after `starts_at`.")
	}
	if !m.AllMonitors.ValueBool() && !m.AllMonitors.IsUnknown() && m.MonitorIDs.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("monitor_ids"), "No monitors selected", "Set `all_monitors = true` or list monitors in `monitor_ids`.")
	}
}

// sameInstant keeps the configured spelling of a time when the server returns it in another offset.
func sameInstant(configured types.String, fromServer string) types.String {
	a, err1 := time.Parse(time.RFC3339, configured.ValueString())
	b, err2 := time.Parse(time.RFC3339, fromServer)
	if !configured.IsNull() && !configured.IsUnknown() && err1 == nil && err2 == nil && a.Equal(b) {
		return configured
	}
	return types.StringValue(fromServer)
}

func (m *maintenanceModel) toAPI(ctx context.Context) (*client.Maintenance, diag.Diagnostics) {
	ids, diags := setInt64s(ctx, m.MonitorIDs)
	return &client.Maintenance{
		Title:       m.Title.ValueString(),
		StartsAt:    m.StartsAt.ValueString(),
		EndsAt:      m.EndsAt.ValueString(),
		AllMonitors: m.AllMonitors.ValueBool(),
		MonitorIDs:  ids,
	}, diags
}

func (m *maintenanceModel) fromAPI(ctx context.Context, w *client.Maintenance) diag.Diagnostics {
	m.ID = types.Int64Value(w.ID)
	m.Title = types.StringValue(w.Title)
	m.StartsAt = sameInstant(m.StartsAt, w.StartsAt)
	m.EndsAt = sameInstant(m.EndsAt, w.EndsAt)
	m.AllMonitors = types.BoolValue(w.AllMonitors)
	var diags diag.Diagnostics
	m.MonitorIDs, diags = int64Set(ctx, w.MonitorIDs)
	return diags
}

func (r *maintenanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan maintenanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	body, diags := plan.toAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.CreateMaintenance(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Creating anpi maintenance window failed", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.fromAPI(ctx, out)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *maintenanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state maintenanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetMaintenance(ctx, state.ID.ValueInt64())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading anpi maintenance window failed", err.Error())
		return
	}
	resp.Diagnostics.Append(state.fromAPI(ctx, out)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *maintenanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state maintenanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	body, diags := plan.toAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpdateMaintenance(ctx, state.ID.ValueInt64(), body)
	if err != nil {
		resp.Diagnostics.AddError("Updating anpi maintenance window failed", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.fromAPI(ctx, out)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *maintenanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state maintenanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteMaintenance(ctx, state.ID.ValueInt64()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting anpi maintenance window failed", err.Error())
	}
}

func (r *maintenanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID(ctx, req, resp)
}
