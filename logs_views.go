package main

import (
	htmltmpl "html/template"
	"net/http"
)

func renderLogsPage(w http.ResponseWriter, data logsPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = logsPageTemplate.Execute(w, data)
}

var logsPageTemplate = htmltmpl.Must(htmltmpl.New("logs").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Cloud-Init Portal Logs</title>
  <style>
    body { font-family: sans-serif; max-width: 980px; margin: 2rem auto; line-height: 1.4; font-size: 16px; padding: 0 1rem; }
    .card { border: 1px solid #ddd; border-radius: 8px; padding: 0.9rem; margin-top: 1rem; }
    .muted { color: #555; }
    table { width: 100%; border-collapse: collapse; margin-top: 0.8rem; }
    th, td { border: 1px solid #ddd; padding: 0.55rem; text-align: left; vertical-align: top; }
    th { background: #f5f5f5; }
    label { display: inline-block; margin-right: 0.4rem; font-weight: 700; }
    input { margin-right: 0.8rem; padding: 0.4rem; }
    button { padding: 0.5rem 0.8rem; cursor: pointer; }
    .err { background: #ffe9e9; border: 1px solid #e88; padding: 0.7rem; margin-top: 0.7rem; }
  </style>
</head>
<body>
  <h1>Provisioning Event History</h1>
  <p class="muted">Append-only event history from the provisioning logger.</p>
  <p><a href="/">Back to Config</a></p>

  <div class="card">
    <form method="get" action="/logs">
      <label for="event">Event</label>
      <input id="event" name="event" value="{{.EventFilter}}" placeholder="generated" />
      <label for="hostname">Hostname</label>
      <input id="hostname" name="hostname" value="{{.HostnameFilter}}" placeholder="edge-001" />
      <label for="limit">Limit</label>
      <input id="limit" name="limit" value="{{.Limit}}" style="width: 70px;" />
      <button type="submit">Apply Filters</button>
    </form>

    {{if .Error}}<div class="err"><strong>Error:</strong> {{.Error}}</div>{{end}}

    <table>
      <thead>
        <tr>
          <th>Timestamp</th>
          <th>Event</th>
          <th>Hostname</th>
          <th>Template</th>
          <th>Box Type</th>
          <th>IP</th>
        </tr>
      </thead>
      <tbody>
      {{if .Events}}
        {{range .Events}}
          <tr>
            <td>{{.Timestamp}}</td>
            <td>{{.Event}}</td>
            <td>{{.Hostname}}</td>
            <td>{{.TemplateName}}</td>
            <td>{{.BoxTypeName}}</td>
            <td>{{.StaticIP}}{{if .CIDR}}/{{.CIDR}}{{end}}</td>
          </tr>
        {{end}}
      {{else}}
        <tr><td colspan="6">No events found.</td></tr>
      {{end}}
      </tbody>
    </table>
  </div>
</body>
</html>`))
