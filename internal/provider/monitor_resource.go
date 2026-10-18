package provider

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/igorovh/terraform-provider-anpi/internal/client"
)

var (
	_ resource.Resource                   = (*monitorResource)(nil)
	_ resource.ResourceWithConfigure      = (*monitorResource)(nil)
	_ resource.ResourceWithImportState    = (*monitorResource)(nil)
	_ resource.ResourceWithValidateConfig = (*monitorResource)(nil)
)

var (
	monitorKinds   = []string{"http", "websocket", "tcp", "ping", "dns", "push", "aggregate"}
	dnsRecordTypes = []string{"A", "AAAA", "CNAME", "MX", "NS", "TXT", "SOA", "CAA"}
)

type monitorResource struct {
	client *client.Client
}

type monitorModel struct {
	ID               types.Int64  `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Kind             types.String `tfsdk:"kind"`
	Target           types.String `tfsdk:"target"`
	Port             types.Int64  `tfsdk:"port"`
	Method           types.String `tfsdk:"method"`
	Headers          types.Map    `tfsdk:"headers"`
	Body             types.String `tfsdk:"body"`
	IntervalS        types.Int64  `tfsdk:"interval_s"`
	RetryIntervalS   types.Int64  `tfsdk:"retry_interval_s"`
	TimeoutS         types.Int64  `tfsdk:"timeout_s"`
	FailureThreshold types.Int64  `tfsdk:"failure_threshold"`
	ExpectedStatus   types.String `tfsdk:"expected_status"`
	IPFamily         types.String `tfsdk:"ip_family"`
	FollowRedirects  types.Bool   `tfsdk:"follow_redirects"`
	IgnoreTLS        types.Bool   `tfsdk:"ignore_tls"`
	ContentKind      types.String `tfsdk:"content_kind"`
	ContentValue     types.String `tfsdk:"content_value"`
	ContentExpected  types.String `tfsdk:"content_expected"`
	SSLWarnDays      types.Int64  `tfsdk:"ssl_warn_days"`
	DNSRecordType    types.String `tfsdk:"dns_record_type"`
	DNSServer        types.String `tfsdk:"dns_server"`
	Active           types.Bool   `tfsdk:"active"`
	Public           types.Bool   `tfsdk:"public"`
	PublicName       types.String `tfsdk:"public_name"`
	GroupID          types.Int64  `tfsdk:"group_id"`
	ParentID         types.Int64  `tfsdk:"parent_id"`
	ChannelIDs       types.Set    `tfsdk:"channel_ids"`
	PushToken        types.String `tfsdk:"push_token"`
	PushURL          types.String `tfsdk:"push_url"`
}

func NewMonitorResource() resource.Resource {
	return &monitorResource{}
}

func (r *monitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (r *monitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(desc, def string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(def), MarkdownDescription: desc}
	}
	num := func(desc string, def, lo, hi int64) schema.Int64Attribute {
		return schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(def), MarkdownDescription: desc,
			Validators: []validatorInt64{int64validator.Between(lo, hi)},
		}
	}
	flag := func(desc string, def bool) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(def), MarkdownDescription: desc}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "A monitor: an HTTP(S), WebSocket, TCP, ping, DNS, push (cron heartbeat) or aggregate check.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true, MarkdownDescription: "Numeric monitor ID, as in the panel URL `/admin/monitors/<id>`.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{Required: true, MarkdownDescription: "Name shown in the panel and alerts (up to 100 characters).",
				Validators: []validatorString{stringvalidator.LengthBetween(1, 100)}},
			"kind": schema.StringAttribute{Required: true, MarkdownDescription: "One of `http`, `websocket`, `tcp`, `ping`, `dns`, `push`, `aggregate`.",
				Validators: []validatorString{stringvalidator.OneOf(monitorKinds...)}},
			"target": str("URL for `http` (`https://…`) and `websocket` (`wss://…`); host for `tcp` and `ping`; domain for `dns`. Not used by `push` and `aggregate`.", ""),
			"port": schema.Int64Attribute{Optional: true, MarkdownDescription: "Port, required for `tcp` and only used by it.",
				Validators: []validatorInt64{int64validator.Between(1, 65535)}},
			"method": str("HTTP method for `http` monitors.", "GET"),
			"headers": schema.MapAttribute{
				ElementType: types.StringType, Optional: true, Computed: true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				MarkdownDescription: "Request headers for `http` and `websocket` monitors.",
			},
			"body":              str("Request body for `http` monitors.", ""),
			"interval_s":        num("Seconds between checks (5 to 604800).", 60, 5, 7*86400),
			"retry_interval_s":  num("Seconds between checks while a monitor is failing (5 to 604800).", 60, 5, 7*86400),
			"timeout_s":         num("Seconds before a check times out (1 to 300).", 30, 1, 300),
			"failure_threshold": num("Failed checks in a row before the monitor is marked down (1 to 100).", 3, 1, 100),
			"expected_status":   str("Accepted HTTP status codes, e.g. `200-299` or `200,301`.", "200-299"),
			"ip_family": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("auto"),
				MarkdownDescription: "`auto`, `v4` or `v6`.", Validators: []validatorString{stringvalidator.OneOf("auto", "v4", "v6")}},
			"follow_redirects": flag("Follow HTTP redirects.", true),
			"ignore_tls":       flag("Accept invalid TLS certificates.", false),
			"content_kind": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("none"),
				MarkdownDescription: "Response rule for `http` and `websocket`: `none`, `contains`, `not_contains`, `regex` or `json_path`.",
				Validators:          []validatorString{stringvalidator.OneOf("none", "contains", "not_contains", "regex", "json_path")}},
			"content_value":    str("Keyword, regex or JSONPath used by `content_kind`.", ""),
			"content_expected": str("Expected value for a `json_path` rule, or the expected answer of a `dns` monitor.", ""),
			"ssl_warn_days":    num("Days before certificate expiry to send a warning (0 to 365).", 14, 0, 365),
			"dns_record_type": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("A"),
				MarkdownDescription: "Record type for `dns` monitors.", Validators: []validatorString{stringvalidator.OneOf(dnsRecordTypes...)}},
			"dns_server":  str("DNS server IP (optionally `ip:port`) for `dns` monitors; empty uses the system resolver.", ""),
			"active":      flag("Whether checks run. `false` pauses the monitor.", true),
			"public":      flag("Show the monitor on the public status page.", false),
			"public_name": str("Name shown on the status page instead of `name`.", ""),
			"group_id": schema.Int64Attribute{Optional: true,
				MarkdownDescription: "ID of an `anpi_group`. Sub-monitors usually use the group of their parent."},
			"parent_id": schema.Int64Attribute{Optional: true,
				MarkdownDescription: "ID of the parent monitor; only one level of nesting is allowed."},
			"channel_ids": schema.SetAttribute{
				ElementType: types.Int64Type, Optional: true, Computed: true,
				Default:             setdefault.StaticValue(types.SetValueMust(types.Int64Type, []attr.Value{})),
				MarkdownDescription: "IDs of `anpi_notification_channel`s that alert for this monitor. Defaults to none; channels marked as default in the panel are not added automatically.",
			},
			"push_token": schema.StringAttribute{Computed: true, Sensitive: true,
				MarkdownDescription: "Token of a `push` monitor.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"push_url": schema.StringAttribute{Computed: true, Sensitive: true,
				MarkdownDescription: "URL a `push` monitor expects to be called, e.g. from cron.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *monitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

// ValidateConfig rejects settings the server would silently drop for the chosen kind, which would show up as drift.
func (r *monitorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var m monitorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() || m.Kind.IsUnknown() || m.Kind.IsNull() {
		return
	}
	kind := m.Kind.ValueString()
	set := func(v types.String, def string) bool { return !v.IsNull() && !v.IsUnknown() && v.ValueString() != def }
	bad := func(attr, msg string) {
		resp.Diagnostics.AddAttributeError(path.Root(attr), "Setting not used by this monitor kind", msg)
	}
	switch {
	case kind == "tcp" && m.Port.IsNull():
		bad("port", "`tcp` monitors need a `port`.")
	case kind != "tcp" && !m.Port.IsNull():
		bad("port", fmt.Sprintf("`port` is only used by `tcp` monitors, not `%s`; for other kinds put the port in the URL.", kind))
	}
	needsTarget := !slices.Contains([]string{"push", "aggregate"}, kind)
	if needsTarget && m.Target.IsNull() {
		bad("target", fmt.Sprintf("`%s` monitors need a `target`.", kind))
	}
	if !needsTarget && set(m.Target, "") {
		bad("target", fmt.Sprintf("`%s` monitors have no target.", kind))
	}
	if kind != "http" && set(m.Method, "GET") {
		bad("method", "`method` is only used by `http` monitors.")
	}
	usesURL := kind == "http" || kind == "websocket"
	if !usesURL && set(m.ContentKind, "none") {
		bad("content_kind", "Response rules are only used by `http` and `websocket` monitors.")
	}
	if !usesURL && kind != "dns" && set(m.ContentExpected, "") {
		bad("content_expected", "`content_expected` is only used by `http`, `websocket` and `dns` monitors.")
	}
}

func headersText(ctx context.Context, m types.Map) (string, diag.Diagnostics) {
	values := map[string]string{}
	if m.IsNull() || m.IsUnknown() {
		return "", nil
	}
	diags := m.ElementsAs(ctx, &values, false)
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+": "+values[k])
	}
	return strings.Join(lines, "\n"), diags
}

func headersMap(raw string) types.Map {
	values := map[string]attr.Value{}
	for _, line := range strings.Split(raw, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), ":"); ok {
			values[strings.TrimSpace(k)] = types.StringValue(strings.TrimSpace(v))
		}
	}
	return types.MapValueMust(types.StringType, values)
}

func (m *monitorModel) toAPI(ctx context.Context) (*client.Monitor, diag.Diagnostics) {
	headers, diags := headersText(ctx, m.Headers)
	channels, d := setInt64s(ctx, m.ChannelIDs)
	diags.Append(d...)
	return &client.Monitor{
		Name:             m.Name.ValueString(),
		Kind:             m.Kind.ValueString(),
		Target:           m.Target.ValueString(),
		Port:             int64Ptr(m.Port),
		Method:           m.Method.ValueString(),
		Headers:          headers,
		Body:             m.Body.ValueString(),
		IntervalS:        m.IntervalS.ValueInt64(),
		RetryIntervalS:   m.RetryIntervalS.ValueInt64(),
		TimeoutS:         m.TimeoutS.ValueInt64(),
		FailureThreshold: m.FailureThreshold.ValueInt64(),
		ExpectedStatus:   m.ExpectedStatus.ValueString(),
		IPFamily:         m.IPFamily.ValueString(),
		FollowRedirects:  m.FollowRedirects.ValueBool(),
		IgnoreTLS:        m.IgnoreTLS.ValueBool(),
		ContentKind:      m.ContentKind.ValueString(),
		ContentValue:     m.ContentValue.ValueString(),
		ContentExpected:  m.ContentExpected.ValueString(),
		SSLWarnDays:      m.SSLWarnDays.ValueInt64(),
		DNSRecordType:    m.DNSRecordType.ValueString(),
		DNSServer:        m.DNSServer.ValueString(),
		Active:           m.Active.ValueBool(),
		Public:           m.Public.ValueBool(),
		PublicName:       m.PublicName.ValueString(),
		GroupID:          int64Ptr(m.GroupID),
		ParentID:         int64Ptr(m.ParentID),
		ChannelIDs:       channels,
	}, diags
}

func (m *monitorModel) fromAPI(ctx context.Context, a *client.Monitor, baseURL string) diag.Diagnostics {
	m.ID = types.Int64Value(a.ID)
	m.Name = types.StringValue(a.Name)
	m.Kind = types.StringValue(a.Kind)
	m.Target = types.StringValue(a.Target)
	m.Port = types.Int64PointerValue(a.Port)
	m.Method = types.StringValue(a.Method)
	m.Headers = headersMap(a.Headers)
	m.Body = types.StringValue(a.Body)
	m.IntervalS = types.Int64Value(a.IntervalS)
	m.RetryIntervalS = types.Int64Value(a.RetryIntervalS)
	m.TimeoutS = types.Int64Value(a.TimeoutS)
	m.FailureThreshold = types.Int64Value(a.FailureThreshold)
	m.ExpectedStatus = types.StringValue(a.ExpectedStatus)
	m.IPFamily = types.StringValue(a.IPFamily)
	m.FollowRedirects = types.BoolValue(a.FollowRedirects)
	m.IgnoreTLS = types.BoolValue(a.IgnoreTLS)
	m.ContentKind = types.StringValue(a.ContentKind)
	m.ContentValue = types.StringValue(a.ContentValue)
	m.ContentExpected = types.StringValue(a.ContentExpected)
	m.SSLWarnDays = types.Int64Value(a.SSLWarnDays)
	m.DNSRecordType = types.StringValue(a.DNSRecordType)
	m.DNSServer = types.StringValue(a.DNSServer)
	m.Active = types.BoolValue(a.Active)
	m.Public = types.BoolValue(a.Public)
	m.PublicName = types.StringValue(a.PublicName)
	m.GroupID = types.Int64PointerValue(a.GroupID)
	m.ParentID = types.Int64PointerValue(a.ParentID)
	var diags diag.Diagnostics
	m.ChannelIDs, diags = int64Set(ctx, a.ChannelIDs)
	if a.PushToken != nil && a.Kind == "push" {
		m.PushToken = types.StringValue(*a.PushToken)
		m.PushURL = types.StringValue(baseURL + "/api/push/" + *a.PushToken)
	} else {
		m.PushToken = types.StringNull()
		m.PushURL = types.StringNull()
	}
	return diags
}

func (r *monitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	body, diags := plan.toAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.CreateMonitor(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Creating anpi monitor failed", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.fromAPI(ctx, out, r.client.BaseURL())...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetMonitor(ctx, state.ID.ValueInt64())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading anpi monitor failed", err.Error())
		return
	}
	resp.Diagnostics.Append(state.fromAPI(ctx, out, r.client.BaseURL())...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *monitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state monitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	body, diags := plan.toAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpdateMonitor(ctx, state.ID.ValueInt64(), body)
	if err != nil {
		resp.Diagnostics.AddError("Updating anpi monitor failed", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.fromAPI(ctx, out, r.client.BaseURL())...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteMonitor(ctx, state.ID.ValueInt64()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting anpi monitor failed", err.Error())
	}
}

func (r *monitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID(ctx, req, resp)
}
