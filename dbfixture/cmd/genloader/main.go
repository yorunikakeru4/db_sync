// Command genloader generates boilerplate loader code for the dbfixture package.
//
// Currently generates notification channel loaders. Run via:
//
//	go generate ./dbfixture/...
package main

import (
	"bytes"
	"embed"
	"go/format"
	"log"
	"os"
	"text/template"
)

//go:embed *.tmpl
var templates embed.FS

// channelDef describes a notification channel type for code generation.
type channelDef struct {
	Name         string // PascalCase type stem, e.g. "Slack"
	LoaderSuffix string // snake_case prefix for loader name, e.g. "slack"
	ChannelConst string // suffix for models.NotifChannel<X>, e.g. "Slack"
	SeedField    string // field name in SeedData.NotifChannels, e.g. "Slack"
}

var channels = []channelDef{
	{Name: "Slack", LoaderSuffix: "slack", ChannelConst: "Slack", SeedField: "Slack"},
	{Name: "GoogleChat", LoaderSuffix: "google_chat", ChannelConst: "GoogleChat", SeedField: "GoogleChat"},
	{Name: "Mattermost", LoaderSuffix: "mattermost", ChannelConst: "Mattermost", SeedField: "Mattermost"},
	{Name: "Pagerduty", LoaderSuffix: "pagerduty", ChannelConst: "Pagerduty", SeedField: "Pagerduty"},
	{Name: "Servicenow", LoaderSuffix: "servicenow", ChannelConst: "Servicenow", SeedField: "Servicenow"},
	{Name: "Opsgenie", LoaderSuffix: "opsgenie", ChannelConst: "Opsgenie", SeedField: "Opsgenie"},
	{Name: "Telegram", LoaderSuffix: "telegram", ChannelConst: "Telegram", SeedField: "Telegram"},
	{Name: "Teams", LoaderSuffix: "teams", ChannelConst: "Teams", SeedField: "Teams"},
	{Name: "Webhook", LoaderSuffix: "webhook", ChannelConst: "Webhook", SeedField: "Webhook"},
}

func main() {
	generateFromTemplate("notif_channel.go.tmpl", "loader_notif_channel_gen.go", channels)
}

// generateFromTemplate executes an embedded template file with the given data
// and writes the gofmt'd result to outFile in the current directory.
func generateFromTemplate(tmplFile, outFile string, data any) {
	tmpl, err := template.ParseFS(templates, tmplFile)
	if err != nil {
		log.Fatalf("parse template %s: %v", tmplFile, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Fatalf("execute template %s: %v", tmplFile, err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("gofmt %s: %v\n\nRaw output:\n%s", outFile, err, buf.Bytes())
	}

	if err := os.WriteFile(outFile, formatted, 0o644); err != nil {
		log.Fatal(err)
	}
}
