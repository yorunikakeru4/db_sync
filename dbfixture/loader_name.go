package dbfixture

// Loader names used to register and look up loaders in the fixture registry.
// These constants must match the names used in YAML/JSON fixture files.
const (
	ModelOrg       = "org"
	ModelUser      = "user"
	ModelUserToken = "user_token"
	ModelUserNotification = "user_notification"
	ModelOrgUser   = "org_user"

	ModelProject      = "project"
	ModelProjectToken = "project_token"
	ModelProjectUser  = "project_user"

	ModelTeam        = "team"
	ModelTeamUser    = "team_user"
	ModelTeamProject = "team_project"

	ModelDashboard       = "dashboard"
	ModelProjectTag      = "project_tag"
	ModelTaggedDashboard = "tagged_dashboard"

	ModelMetricMonitor = "metric_monitor"
	ModelErrorMonitor  = "error_monitor"

	ModelMetricAlert      = "metric_alert"
	ModelErrorAlert       = "error_alert"
	ModelMetricAlertEvent = "metric_alert_event"
	ModelErrorAlertEvent  = "error_alert_event"
	ModelAlertNote        = "alert_note"
	ModelAlertAssignment  = "alert_assignment"

	// Notification channel loader constants are in loader_notif_channel_gen.go.

	ModelSpan            = "span"
	ModelLog             = "log"
	ModelEvent           = "event"
	ModelTraceOverride   = "trace_override"
	ModelDatapoint       = "datapoint"
	ModelLogGroupingRule = "log_grouping_rule"
)
