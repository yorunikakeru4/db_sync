package seedinput

import (
	"net/url"

	opsgeniealert "github.com/opsgenie/opsgenie-go-sdk-v2/alert"

	"github.com/uptrace/bun"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/validerr"
)

// NotifChannel is the interface implemented by all notification channel types.
type NotifChannel interface {
	Base() *BaseNotifChannel
	Validate() error
}

var (
	_ NotifChannel = (*WebhookNotifChannel)(nil)
	_ NotifChannel = (*MattermostNotifChannel)(nil)
	_ NotifChannel = (*OpsgenieNotifChannel)(nil)
	_ NotifChannel = (*PagerdutyNotifChannel)(nil)
	_ NotifChannel = (*ServicenowNotifChannel)(nil)
	_ NotifChannel = (*SlackNotifChannel)(nil)
	_ NotifChannel = (*GoogleChatNotifChannel)(nil)
	_ NotifChannel = (*TeamsNotifChannel)(nil)
	_ NotifChannel = (*TelegramNotifChannel)(nil)
)

// BaseNotifChannel represents the base notification channel fixture input.
type BaseNotifChannel struct {
	bun.BaseModel `bun:"notif_channels,alias:c"`

	Key string `yaml:"key" json:"key" bun:"-"`

	ProjectKey string `yaml:"project_key" json:"projectKey" bun:"-"`
	ProjectID  uint32 `yaml:"-" json:"-" bun:",nullzero"`

	Name   *string                    `yaml:"name" json:"name" bun:",nullzero"`
	Status *models.NotifChannelStatus `yaml:"status" json:"status" bun:",nullzero"`
	Type   models.NotifChannelType    `yaml:"-" json:"-"`

	// MatchAll enables all monitors matching.
	// If false, MonitorKeys must be provided.
	MatchAll *bool `yaml:"match_all" json:"matchAll"`
	// MonitorKeys must be nil if MatchAll = true.
	MonitorKeys []string `yaml:"monitor_keys" json:"monitorRefs" bun:"-"`

	Priorities []models.AlertPriority `yaml:"priorities" json:"priorities" bun:",array,nullzero"`
	Condition  *string                `yaml:"condition" json:"condition" bun:",nullzero"`
}

// FixtureKey returns the fixture key for this notification channel.
func (c *BaseNotifChannel) FixtureKey() string { return c.Key }

// Validate validates the base notification channel input.
func (c *BaseNotifChannel) Validate() error {
	if c.Name != nil && *c.Name == "" {
		return validerr.Empty("name")
	}
	if c.Status != nil && *c.Status == "" {
		return validerr.Empty("status")
	}
	if len(c.Priorities) == 0 {
		return validerr.AtLeastOne("priorities")
	}
	return nil
}

// ValidateWebhookURL validates a webhook URL.
func (c *BaseNotifChannel) ValidateWebhookURL(webhookURL string) error {
	if webhookURL == "" {
		return validerr.Empty("webhook_url")
	}

	u, err := url.Parse(webhookURL)
	if err != nil {
		return validerr.Invalid("webhook_url", err.Error())
	}
	switch u.Scheme {
	case "http", "https":
	default:
		return validerr.UnsupportedValue("webhook_url.scheme", u.Scheme)
	}
	return nil
}

// SlackNotifChannel represents a Slack notification channel fixture input.
type SlackNotifChannel struct {
	*BaseNotifChannel `yaml:",inline" bun:",inherit"`

	Params SlackParams `yaml:"params" json:"params"`
}

// SlackParams holds Slack notification channel parameters.
type SlackParams struct {
	// AuthMethod determines which authentication method to use:
	// - SlackAuthWebhook: uses WebhookURL for sending messages
	// - SlackAuthToken: uses Token and Channel for sending messages via Slack API
	AuthMethod string `yaml:"auth_method" json:"authMethod"`

	// WebhookURL is used when AuthMethod is SlackAuthWebhook
	WebhookURL string `yaml:"webhook_url" json:"webhookUrl"`

	// Token and Channel are used when AuthMethod is SlackAuthToken
	Token   string `yaml:"token" json:"token"`
	Channel string `yaml:"channel" json:"channel"`
}

// Base returns the base notification channel.
func (c *SlackNotifChannel) Base() *BaseNotifChannel {
	return c.BaseNotifChannel
}

// Validate validates the Slack notification channel input.
func (c *SlackNotifChannel) Validate() error {
	if err := c.BaseNotifChannel.Validate(); err != nil {
		return err
	}

	if c.Params.AuthMethod == "" {
		c.Params.AuthMethod = "webhook"
	}

	switch c.Params.AuthMethod {
	case "webhook":
		if err := c.ValidateWebhookURL(c.Params.WebhookURL); err != nil {
			return err
		}
	case "token":
		if c.Params.Token == "" {
			return validerr.Empty("token")
		}
		if c.Params.Channel == "" {
			return validerr.Empty("channel")
		}
	default:
		return validerr.UnsupportedValue("auth_method", c.Params.AuthMethod)
	}

	return nil
}

// GoogleChatNotifChannel represents a Google Chat notification channel fixture input.
type GoogleChatNotifChannel struct {
	*BaseNotifChannel `yaml:",inline" bun:",inherit"`

	Params GoogleChatParams `yaml:"params" json:"params"`
}

// GoogleChatParams holds Google Chat notification channel parameters.
type GoogleChatParams struct {
	WebhookURL string `yaml:"webhook_url" json:"webhookUrl"`
}

// Base returns the base notification channel.
func (c *GoogleChatNotifChannel) Base() *BaseNotifChannel {
	return c.BaseNotifChannel
}

// Validate validates the Google Chat notification channel input.
func (c *GoogleChatNotifChannel) Validate() error {
	if err := c.BaseNotifChannel.Validate(); err != nil {
		return err
	}
	if err := c.ValidateWebhookURL(c.Params.WebhookURL); err != nil {
		return err
	}
	return nil
}

// MattermostNotifChannel represents a Mattermost notification channel fixture input.
type MattermostNotifChannel struct {
	*BaseNotifChannel `yaml:",inline" bun:",inherit"`

	Params MattermostParams `yaml:"params" json:"params"`
}

// MattermostParams holds Mattermost notification channel parameters.
type MattermostParams struct {
	WebhookURL string `yaml:"webhook_url" json:"webhookUrl"`
}

// Base returns the base notification channel.
func (c *MattermostNotifChannel) Base() *BaseNotifChannel {
	return c.BaseNotifChannel
}

// Validate validates the Mattermost notification channel input.
func (c *MattermostNotifChannel) Validate() error {
	if err := c.BaseNotifChannel.Validate(); err != nil {
		return err
	}
	if err := c.ValidateWebhookURL(c.Params.WebhookURL); err != nil {
		return err
	}
	return nil
}

// PagerdutyNotifChannel represents a PagerDuty notification channel fixture input.
type PagerdutyNotifChannel struct {
	*BaseNotifChannel `yaml:",inline" bun:",inherit"`

	Params PagerdutyParams `yaml:"params" json:"params"`
}

// PagerdutyParams holds PagerDuty notification channel parameters.
type PagerdutyParams struct {
	RoutingKey string `yaml:"routing_key" json:"routingKey"`
	Severity   string `yaml:"severity" json:"severity"`
}

// Base returns the base notification channel.
func (c *PagerdutyNotifChannel) Base() *BaseNotifChannel {
	return c.BaseNotifChannel
}

// Validate validates the PagerDuty notification channel input.
func (c *PagerdutyNotifChannel) Validate() error {
	if err := c.BaseNotifChannel.Validate(); err != nil {
		return err
	}

	if c.Params.RoutingKey == "" {
		return validerr.Empty("routing_key")
	}

	switch c.Params.Severity {
	case "":
		return validerr.Empty("severity")
	case "critical", "error", "warning", "info":
		// okay
	default:
		return validerr.UnsupportedValue("severity", c.Params.Severity)
	}

	return nil
}

// ServicenowNotifChannel represents a ServiceNow notification channel fixture input.
type ServicenowNotifChannel struct {
	*BaseNotifChannel `yaml:",inline" bun:",inherit"`

	Params ServicenowParams `json:"params"`
}

// ServicenowParams holds ServiceNow notification channel parameters.
type ServicenowParams struct {
	URL string `yaml:"url" json:"url"`

	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`

	Category    string `yaml:"category" json:"category,omitempty"`
	SubCategory string `yaml:"subcategory" json:"subcategory,omitempty"`

	Impact   string `yaml:"impact" json:"impact,omitempty"`
	Urgency  string `yaml:"urgency" json:"urgency,omitempty"`
	Severity string `yaml:"severity" json:"severity,omitempty"`

	CallerID   string `yaml:"caller_id" json:"callerId,omitempty"`
	Group      string `yaml:"group" json:"group,omitempty"`
	AssignedTo string `yaml:"assigned_to" json:"assignedTo,omitempty"`
	OpenedBy   string `yaml:"opened_by" json:"openedBy,omitempty"`

	Notify  string `yaml:"notify" json:"notify,omitempty"`
	DueDate string `yaml:"due_date" json:"dueDate,omitempty"`
}

// Base returns the base notification channel.
func (c *ServicenowNotifChannel) Base() *BaseNotifChannel {
	return c.BaseNotifChannel
}

// Validate validates the ServiceNow notification channel input.
func (c *ServicenowNotifChannel) Validate() error {
	if err := c.BaseNotifChannel.Validate(); err != nil {
		return err
	}
	if err := c.ValidateWebhookURL(c.Params.URL); err != nil {
		return err
	}

	if c.Params.Username == "" {
		return validerr.Empty("username")
	}
	if c.Params.Password == "" {
		return validerr.Empty("password")
	}

	if c.Params.Urgency != "" {
		switch c.Params.Urgency {
		case "1", "2", "3":
			// ok
		default:
			return validerr.UnsupportedValue("urgency", c.Params.Urgency)
		}
	}
	if c.Params.Impact != "" {
		switch c.Params.Impact {
		case "1", "2", "3":
			// ok
		default:
			return validerr.UnsupportedValue("impact", c.Params.Impact)
		}
	}
	if c.Params.Severity != "" {
		switch c.Params.Severity {
		case "1", "2", "3", "4", "5":
			// ok
		default:
			return validerr.UnsupportedValue("severity", c.Params.Severity)
		}
	}
	if c.Params.Notify != "" {
		switch c.Params.Notify {
		case "1", "2":
			// ok
		default:
			return validerr.UnsupportedValue("notify", c.Params.Notify)
		}
	}

	return nil
}

// OpsgenieNotifChannel represents an Opsgenie notification channel fixture input.
type OpsgenieNotifChannel struct {
	*BaseNotifChannel `yaml:",inline" bun:",inherit"`

	Params OpsgenieParams `yaml:"params" json:"params"`
}

// OpsgenieParams holds Opsgenie notification channel parameters.
type OpsgenieParams struct {
	APIKey   string                 `yaml:"api_key" json:"apiKey"`
	Priority opsgeniealert.Priority `yaml:"priority" json:"priority"`
}

// Base returns the base notification channel.
func (c *OpsgenieNotifChannel) Base() *BaseNotifChannel {
	return c.BaseNotifChannel
}

// Validate validates the Opsgenie notification channel input.
func (c *OpsgenieNotifChannel) Validate() error {
	if err := c.BaseNotifChannel.Validate(); err != nil {
		return err
	}

	if c.Params.APIKey == "" {
		return validerr.Empty("api_key")
	}
	if c.Params.Priority == "" {
		return validerr.Empty("priority")
	}

	return nil
}

// TelegramNotifChannel represents a Telegram notification channel fixture input.
type TelegramNotifChannel struct {
	*BaseNotifChannel `yaml:",inline" bun:",inherit"`

	Params TelegramParams `yaml:"params" json:"params"`
}

// TelegramParams holds Telegram notification channel parameters.
type TelegramParams struct {
	ChatID int64 `yaml:"params" json:"chatId"`
}

// Base returns the base notification channel.
func (c *TelegramNotifChannel) Base() *BaseNotifChannel {
	return c.BaseNotifChannel
}

// Validate validates the Telegram notification channel input.
func (c *TelegramNotifChannel) Validate() error {
	if err := c.BaseNotifChannel.Validate(); err != nil {
		return err
	}
	if c.Params.ChatID == 0 {
		return validerr.Zero("chat_id")
	}
	return nil
}

// TeamsNotifChannel represents a Microsoft Teams notification channel fixture input.
type TeamsNotifChannel struct {
	*BaseNotifChannel `yaml:",inline" bun:",inherit"`

	Params TeamsParams `yaml:"params" json:"params"`
}

// TeamsParams holds Teams notification channel parameters.
type TeamsParams struct {
	WebhookURL string `yaml:"webhook_url" json:"webhookUrl"`
}

// Base returns the base notification channel.
func (c *TeamsNotifChannel) Base() *BaseNotifChannel {
	return c.BaseNotifChannel
}

// Validate validates the Teams notification channel input.
func (c *TeamsNotifChannel) Validate() error {
	if err := c.BaseNotifChannel.Validate(); err != nil {
		return err
	}
	if err := c.ValidateWebhookURL(c.Params.WebhookURL); err != nil {
		return err
	}
	return nil
}

// WebhookNotifChannel represents a webhook notification channel fixture input.
type WebhookNotifChannel struct {
	*BaseNotifChannel `yaml:",inline" bun:",inherit"`

	Params WebhookParams `yaml:"params" json:"params"`
}

// WebhookParams holds webhook notification channel parameters.
type WebhookParams struct {
	URL     string `yaml:"url" json:"url"`
	Payload any    `yaml:"payload" json:"payload"`
}

// Base returns the base notification channel.
func (c *WebhookNotifChannel) Base() *BaseNotifChannel {
	return c.BaseNotifChannel
}

// Validate validates the webhook notification channel input.
func (c *WebhookNotifChannel) Validate() error {
	if err := c.BaseNotifChannel.Validate(); err != nil {
		return err
	}
	if err := c.ValidateWebhookURL(c.Params.URL); err != nil {
		return err
	}
	return nil
}
