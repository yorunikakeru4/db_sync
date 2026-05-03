package seedinput

// SeedData holds fixture data to be loaded into the database.
// Used for YAML/JSON unmarshaling in test fixtures and HTTP API.
type SeedData struct {
	Update bool `yaml:"update" json:"update"`
	Delete bool `yaml:"delete" json:"delete"`

	Users      []*User      `yaml:"users" json:"users"`
	UserTokens []*UserToken `yaml:"user_tokens" json:"userTokens"`
	UserNotifications []*UserNotification `yaml:"user_notifications" json:"userNotifications"`

	Orgs     []*Org     `yaml:"orgs" json:"orgs"`
	OrgUsers []*OrgUser `yaml:"org_users" json:"orgUsers"`

	Teams        []*Team        `yaml:"teams" json:"teams"`
	TeamUsers    []*TeamUser    `yaml:"team_users" json:"teamUsers"`
	TeamProjects []*TeamProject `yaml:"team_projects" json:"teamProjects"`

	Projects      []*Project      `yaml:"projects" json:"projects"`
	ProjectTokens []*ProjectToken `yaml:"project_tokens" json:"projectTokens"`
	ProjectUsers  []*ProjectUser  `yaml:"project_users" json:"projectUsers"`

	Dashboards       []*Dashboard       `yaml:"dashboards" json:"dashboards"`
	ProjectTags      []*ProjectTag      `yaml:"project_tags" json:"projectTags"`
	TaggedDashboards []*TaggedDashboard `yaml:"tagged_dashboards" json:"taggedDashboards"`

	MetricMonitors []*MetricMonitor `yaml:"metric_monitors" json:"metricMonitors"`
	ErrorMonitors  []*ErrorMonitor  `yaml:"error_monitors" json:"errorMonitors"`

	MetricAlerts      []*MetricAlert      `yaml:"metric_alerts" json:"-"`
	ErrorAlerts       []*ErrorAlert       `yaml:"error_alerts" json:"-"`
	MetricAlertEvents []*MetricAlertEvent `yaml:"metric_alert_events" json:"-"`
	ErrorAlertEvents  []*ErrorAlertEvent  `yaml:"error_alert_events" json:"-"`

	AlertNotes       []*AlertNote       `yaml:"alert_notes" json:"alertNotes"`
	AlertAssignments []*AlertAssignment `yaml:"alert_assignments" json:"alertAssignments"`

	NotifChannels struct {
		Slack      []*SlackNotifChannel      `yaml:"slack" json:"slack"`
		GoogleChat []*GoogleChatNotifChannel `yaml:"google_chat" json:"googleChat"`
		Mattermost []*MattermostNotifChannel `yaml:"mattermost" json:"mattermost"`
		Pagerduty  []*PagerdutyNotifChannel  `yaml:"pagerduty" json:"pagerduty"`
		Servicenow []*ServicenowNotifChannel `yaml:"servicenow" json:"servicenow"`
		Opsgenie   []*OpsgenieNotifChannel   `yaml:"opsgenie" json:"opsgenie"`
		Telegram   []*TelegramNotifChannel   `yaml:"telegram" json:"telegram"`
		Teams      []*TeamsNotifChannel      `yaml:"teams" json:"teams"`
		Webhook    []*WebhookNotifChannel    `yaml:"webhook" json:"webhook"`
	} `yaml:"notif_channels" json:"notifChannels"`

	Spans      []*Span      `yaml:"spans" json:"-"`
	Logs       []*Span      `yaml:"logs" json:"-"`
	Events     []*Span      `yaml:"events" json:"-"`
	Datapoints []*Datapoint `yaml:"datapoints" json:"-"`

	TraceOverrides []*TraceOverride `yaml:"trace_overrides" json:"traceOverrides"`

	LogGroupingRules []*LogGroupingRule `yaml:"log_grouping_rules" json:"logGroupingRules"`
}
