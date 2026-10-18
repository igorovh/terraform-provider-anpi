// Package client talks to the anpi REST API (/api/v1).
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL   string
	token     string
	userAgent string
	http      *http.Client
}

func New(baseURL, token, userAgent string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		token:     token,
		userAgent: userAgent,
		http:      &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) BaseURL() string { return c.baseURL }

// APIError is a non-2xx answer; anpi puts the reason in {"error": "..."}.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("anpi API returned HTTP %d: %s", e.Status, e.Message)
}

func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/api/v1"+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("calling anpi at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var e struct {
			Error string `json:"error"`
		}
		msg := strings.TrimSpace(string(data))
		if json.Unmarshal(data, &e) == nil && e.Error != "" {
			msg = e.Error
		}
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return &APIError{Status: resp.StatusCode, Message: msg}
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unexpected answer from anpi (is %s an anpi server?): %w", c.baseURL, err)
	}
	return nil
}

type Info struct {
	Version string `json:"version"`
	Token   string `json:"token"`
}

func (c *Client) Info(ctx context.Context) (*Info, error) {
	var out Info
	return &out, c.do(ctx, http.MethodGet, "/info", nil, &out)
}

// Monitor is both the request and the response; read-only fields are ignored by the server.
type Monitor struct {
	ID               int64   `json:"id,omitempty"`
	Name             string  `json:"name"`
	Kind             string  `json:"kind"`
	Target           string  `json:"target"`
	Port             *int64  `json:"port"`
	Method           string  `json:"method"`
	Headers          string  `json:"headers"`
	Body             string  `json:"body"`
	IntervalS        int64   `json:"interval_s"`
	RetryIntervalS   int64   `json:"retry_interval_s"`
	TimeoutS         int64   `json:"timeout_s"`
	FailureThreshold int64   `json:"failure_threshold"`
	ExpectedStatus   string  `json:"expected_status"`
	IPFamily         string  `json:"ip_family"`
	FollowRedirects  bool    `json:"follow_redirects"`
	IgnoreTLS        bool    `json:"ignore_tls"`
	ContentKind      string  `json:"content_kind"`
	ContentValue     string  `json:"content_value"`
	ContentExpected  string  `json:"content_expected"`
	SSLWarnDays      int64   `json:"ssl_warn_days"`
	DNSRecordType    string  `json:"dns_record_type"`
	DNSServer        string  `json:"dns_server"`
	Active           bool    `json:"active"`
	Public           bool    `json:"public"`
	PublicName       string  `json:"public_name"`
	GroupID          *int64  `json:"group_id"`
	ParentID         *int64  `json:"parent_id"`
	ChannelIDs       []int64 `json:"channel_ids"`
	PushToken        *string `json:"push_token,omitempty"`
}

func (c *Client) GetMonitor(ctx context.Context, id int64) (*Monitor, error) {
	var out Monitor
	return &out, c.do(ctx, http.MethodGet, fmt.Sprintf("/monitors/%d", id), nil, &out)
}

func (c *Client) CreateMonitor(ctx context.Context, m *Monitor) (*Monitor, error) {
	var out Monitor
	return &out, c.do(ctx, http.MethodPost, "/monitors", m, &out)
}

func (c *Client) UpdateMonitor(ctx context.Context, id int64, m *Monitor) (*Monitor, error) {
	var out Monitor
	return &out, c.do(ctx, http.MethodPut, fmt.Sprintf("/monitors/%d", id), m, &out)
}

func (c *Client) DeleteMonitor(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/monitors/%d", id), nil, nil)
}

type Group struct {
	ID        int64  `json:"id,omitempty"`
	Name      string `json:"name"`
	SortOrder *int64 `json:"sort_order,omitempty"`
}

func (c *Client) GetGroup(ctx context.Context, id int64) (*Group, error) {
	var out Group
	return &out, c.do(ctx, http.MethodGet, fmt.Sprintf("/groups/%d", id), nil, &out)
}

func (c *Client) CreateGroup(ctx context.Context, g *Group) (*Group, error) {
	var out Group
	return &out, c.do(ctx, http.MethodPost, "/groups", g, &out)
}

func (c *Client) UpdateGroup(ctx context.Context, id int64, g *Group) (*Group, error) {
	var out Group
	return &out, c.do(ctx, http.MethodPut, fmt.Sprintf("/groups/%d", id), g, &out)
}

func (c *Client) DeleteGroup(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/groups/%d", id), nil, nil)
}

// ChannelConfig holds the settings of every channel kind; each kind reads only its own fields.
type ChannelConfig struct {
	WebhookURL   string `json:"webhook_url"`
	Username     string `json:"username"`
	BotToken     string `json:"bot_token"`
	ChatID       string `json:"chat_id"`
	APIBase      string `json:"api_base"`
	Server       string `json:"server"`
	Topic        string `json:"topic"`
	Token        string `json:"token"`
	Priority     string `json:"priority"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     string `json:"smtp_port"`
	SMTPSecurity string `json:"smtp_security"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	EmailFrom    string `json:"email_from"`
	EmailTo      string `json:"email_to"`
	URL          string `json:"url"`
}

type Channel struct {
	ID        int64         `json:"id,omitempty"`
	Name      string        `json:"name"`
	Kind      string        `json:"kind"`
	Active    bool          `json:"active"`
	IsDefault bool          `json:"is_default"`
	Config    ChannelConfig `json:"config"`
}

func (c *Client) GetChannel(ctx context.Context, id int64) (*Channel, error) {
	var out Channel
	return &out, c.do(ctx, http.MethodGet, fmt.Sprintf("/notification-channels/%d", id), nil, &out)
}

func (c *Client) CreateChannel(ctx context.Context, ch *Channel) (*Channel, error) {
	var out Channel
	return &out, c.do(ctx, http.MethodPost, "/notification-channels", ch, &out)
}

func (c *Client) UpdateChannel(ctx context.Context, id int64, ch *Channel) (*Channel, error) {
	var out Channel
	return &out, c.do(ctx, http.MethodPut, fmt.Sprintf("/notification-channels/%d", id), ch, &out)
}

func (c *Client) DeleteChannel(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/notification-channels/%d", id), nil, nil)
}

type Maintenance struct {
	ID          int64   `json:"id,omitempty"`
	Title       string  `json:"title"`
	StartsAt    string  `json:"starts_at"`
	EndsAt      string  `json:"ends_at"`
	AllMonitors bool    `json:"all_monitors"`
	MonitorIDs  []int64 `json:"monitor_ids"`
}

func (c *Client) GetMaintenance(ctx context.Context, id int64) (*Maintenance, error) {
	var out Maintenance
	return &out, c.do(ctx, http.MethodGet, fmt.Sprintf("/maintenances/%d", id), nil, &out)
}

func (c *Client) CreateMaintenance(ctx context.Context, m *Maintenance) (*Maintenance, error) {
	var out Maintenance
	return &out, c.do(ctx, http.MethodPost, "/maintenances", m, &out)
}

func (c *Client) UpdateMaintenance(ctx context.Context, id int64, m *Maintenance) (*Maintenance, error) {
	var out Maintenance
	return &out, c.do(ctx, http.MethodPut, fmt.Sprintf("/maintenances/%d", id), m, &out)
}

func (c *Client) DeleteMaintenance(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/maintenances/%d", id), nil, nil)
}
