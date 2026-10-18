package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/igorovh/terraform-provider-anpi/internal/client"
)

var (
	_ resource.Resource                = (*channelResource)(nil)
	_ resource.ResourceWithConfigure   = (*channelResource)(nil)
	_ resource.ResourceWithImportState = (*channelResource)(nil)
)

type channelResource struct {
	client *client.Client
}

type channelModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Kind         types.String `tfsdk:"kind"`
	Active       types.Bool   `tfsdk:"active"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
	WebhookURL   types.String `tfsdk:"webhook_url"`
	Username     types.String `tfsdk:"username"`
	BotToken     types.String `tfsdk:"bot_token"`
	ChatID       types.String `tfsdk:"chat_id"`
	APIBase      types.String `tfsdk:"api_base"`
	Server       types.String `tfsdk:"server"`
	Topic        types.String `tfsdk:"topic"`
	Token        types.String `tfsdk:"token"`
	Priority     types.String `tfsdk:"priority"`
	SMTPHost     types.String `tfsdk:"smtp_host"`
	SMTPPort     types.String `tfsdk:"smtp_port"`
	SMTPSecurity types.String `tfsdk:"smtp_security"`
	SMTPUsername types.String `tfsdk:"smtp_username"`
	SMTPPassword types.String `tfsdk:"smtp_password"`
	EmailFrom    types.String `tfsdk:"email_from"`
	EmailTo      types.String `tfsdk:"email_to"`
	URL          types.String `tfsdk:"url"`
}

func NewNotificationChannelResource() resource.Resource {
	return &channelResource{}
}

func (r *channelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_channel"
}

func (r *channelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	opt := func(desc string, sensitive bool) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Sensitive: sensitive, Default: stringdefault.StaticString(""), MarkdownDescription: desc}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "A notification channel that sends alerts to Discord, Telegram, ntfy, e-mail or a webhook. Each kind reads only its own settings.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{Computed: true, MarkdownDescription: "Numeric channel ID.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"name": schema.StringAttribute{Required: true, MarkdownDescription: "Channel name.",
				Validators: []validatorString{stringvalidator.LengthAtLeast(1)}},
			"kind": schema.StringAttribute{Required: true, MarkdownDescription: "One of `discord`, `telegram`, `ntfy`, `email`, `webhook`.",
				Validators: []validatorString{stringvalidator.OneOf("discord", "telegram", "ntfy", "email", "webhook")}},
			"active": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				MarkdownDescription: "Whether alerts are sent through this channel."},
			"is_default": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				MarkdownDescription: "Pre-select this channel for monitors created in the panel. It does not affect `anpi_monitor`, whose `channel_ids` are explicit."},
			"webhook_url":   opt("`discord`: webhook URL.", true),
			"username":      opt("`discord`: name the messages are posted as.", false),
			"bot_token":     opt("`telegram`: bot token.", true),
			"chat_id":       opt("`telegram`: chat ID.", false),
			"api_base":      opt("`telegram`: API base URL; empty uses api.telegram.org.", false),
			"server":        opt("`ntfy`: server URL; empty uses ntfy.sh.", false),
			"topic":         opt("`ntfy`: topic.", false),
			"token":         opt("`ntfy`: access token.", true),
			"priority":      opt("`ntfy`: message priority.", false),
			"smtp_host":     opt("`email`: SMTP host.", false),
			"smtp_port":     opt("`email`: SMTP port, e.g. `587`.", false),
			"smtp_security": opt("`email`: `starttls` (used when empty), `tls` or `none`.", false),
			"smtp_username": opt("`email`: SMTP user name.", false),
			"smtp_password": opt("`email`: SMTP password.", true),
			"email_from":    opt("`email`: sender address.", false),
			"email_to":      opt("`email`: recipient addresses, separated by commas.", false),
			"url":           opt("`webhook`: URL that receives a JSON POST for each event.", true),
		},
	}
}

func (r *channelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (m *channelModel) toAPI() *client.Channel {
	return &client.Channel{
		Name:      m.Name.ValueString(),
		Kind:      m.Kind.ValueString(),
		Active:    m.Active.ValueBool(),
		IsDefault: m.IsDefault.ValueBool(),
		Config: client.ChannelConfig{
			WebhookURL:   m.WebhookURL.ValueString(),
			Username:     m.Username.ValueString(),
			BotToken:     m.BotToken.ValueString(),
			ChatID:       m.ChatID.ValueString(),
			APIBase:      m.APIBase.ValueString(),
			Server:       m.Server.ValueString(),
			Topic:        m.Topic.ValueString(),
			Token:        m.Token.ValueString(),
			Priority:     m.Priority.ValueString(),
			SMTPHost:     m.SMTPHost.ValueString(),
			SMTPPort:     m.SMTPPort.ValueString(),
			SMTPSecurity: m.SMTPSecurity.ValueString(),
			SMTPUsername: m.SMTPUsername.ValueString(),
			SMTPPassword: m.SMTPPassword.ValueString(),
			EmailFrom:    m.EmailFrom.ValueString(),
			EmailTo:      m.EmailTo.ValueString(),
			URL:          m.URL.ValueString(),
		},
	}
}

func (m *channelModel) fromAPI(c *client.Channel) {
	m.ID = types.Int64Value(c.ID)
	m.Name = types.StringValue(c.Name)
	m.Kind = types.StringValue(c.Kind)
	m.Active = types.BoolValue(c.Active)
	m.IsDefault = types.BoolValue(c.IsDefault)
	cfg := c.Config
	m.WebhookURL = types.StringValue(cfg.WebhookURL)
	m.Username = types.StringValue(cfg.Username)
	m.BotToken = types.StringValue(cfg.BotToken)
	m.ChatID = types.StringValue(cfg.ChatID)
	m.APIBase = types.StringValue(cfg.APIBase)
	m.Server = types.StringValue(cfg.Server)
	m.Topic = types.StringValue(cfg.Topic)
	m.Token = types.StringValue(cfg.Token)
	m.Priority = types.StringValue(cfg.Priority)
	m.SMTPHost = types.StringValue(cfg.SMTPHost)
	m.SMTPPort = types.StringValue(cfg.SMTPPort)
	m.SMTPSecurity = types.StringValue(cfg.SMTPSecurity)
	m.SMTPUsername = types.StringValue(cfg.SMTPUsername)
	m.SMTPPassword = types.StringValue(cfg.SMTPPassword)
	m.EmailFrom = types.StringValue(cfg.EmailFrom)
	m.EmailTo = types.StringValue(cfg.EmailTo)
	m.URL = types.StringValue(cfg.URL)
}

func (r *channelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan channelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.CreateChannel(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Creating anpi notification channel failed", err.Error())
		return
	}
	plan.fromAPI(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state channelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetChannel(ctx, state.ID.ValueInt64())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading anpi notification channel failed", err.Error())
		return
	}
	state.fromAPI(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *channelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state channelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpdateChannel(ctx, state.ID.ValueInt64(), plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Updating anpi notification channel failed", err.Error())
		return
	}
	plan.fromAPI(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state channelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteChannel(ctx, state.ID.ValueInt64()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting anpi notification channel failed", err.Error())
	}
}

func (r *channelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID(ctx, req, resp)
}
