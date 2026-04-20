package main

import (
	htmltmpl "html/template"
	"net/http"
	"time"
)

func (s *Server) renderError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	s.renderIndex(w, indexData{
		TemplateNames: TemplateNames(s.Templates),
		BoxTypes:      sortedBoxTypes(s.BoxTypes),
		Current:       s.Store.GetCurrent(),
		Status:        s.Store.CurrentStatus(),
		Error:         message,
	})
}

func (s *Server) renderIndex(w http.ResponseWriter, data indexData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = indexPageTemplate.Execute(w, data)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

var indexPageTemplate = htmltmpl.Must(htmltmpl.New("index").Funcs(htmltmpl.FuncMap{"fmtTime": formatTime}).Parse(`<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <title>Cloud-Init Portal</title>
  <style>
    body { font-family: sans-serif; max-width: 980px; margin: 2rem auto; line-height: 1.4; font-size: 17px; }
    label { display: block; margin-top: 0.6rem; font-weight: 700; }
    input, select { width: 100%; padding: 0.55rem; font-size: 16px; }
    button { margin-top: 0.8rem; padding: 0.6rem 1rem; font-size: 16px; cursor: pointer; }
    pre { background: #111; color: #eee; padding: 0.8rem; overflow-x: auto; }
    .err { background: #ffe9e9; border: 1px solid #e88; padding: 0.7rem; margin-bottom: 0.7rem; }
    .ok { background: #e9fff0; border: 1px solid #7c7; padding: 0.7rem; margin-bottom: 0.7rem; font-size: 18px; }
    .card { border: 1px solid #ddd; padding: 0.9rem; margin-top: 1rem; border-radius: 8px; }
    .status-table { width: 100%; border-collapse: collapse; }
    .status-table th, .status-table td { border: 1px solid #ddd; padding: 0.55rem; text-align: left; }
    .status-table th { background: #f5f5f5; width: 220px; }
    .danger { background: #b40000; color: #fff; border: none; }
    .row { display: flex; gap: 0.7rem; flex-wrap: wrap; }
    .muted { color: #555; }
  </style>
</head>
<body>
  <h1>Cloud-Init Provision Portal</h1>
  <p class="muted">One active config at a time. Create a config, then let target fetch <code>/user-data</code> and <code>/meta-data</code>.</p>

  {{if .Error}}<div class="err"><strong>Error:</strong> {{.Error}}</div>{{end}}
  {{if .Message}}<div class="ok"><strong>{{.Message}}</strong></div>{{end}}

  {{if .Success}}
  <div class="card">
    <h2>Provisioning Instructions</h2>
    <p><strong>Config ready for box:</strong> <code>{{.Success.Hostname}}</code></p>
    <ul>
      <li>User-data URL: <a href="{{.Success.UserDataURL}}">{{.Success.UserDataURL}}</a></li>
      <li>Meta-data URL: <a href="{{.Success.MetaDataURL}}">{{.Success.MetaDataURL}}</a></li>
    </ul>
    <h3>Suggested iPXE kernel arg</h3>
    <pre>{{.Success.IPXEExample}}</pre>
    <h3>Suggested fetch test</h3>
    <pre>{{.Success.CurlExample}}</pre>
  </div>
  {{end}}

  <div class="card" id="status-card">
    <h2>Current Provisioning Status</h2>
    <table class="status-table">
      <tr><th>Hostname</th><td id="st-hostname">{{if .Status.Hostname}}{{.Status.Hostname}}{{else}}-{{end}}</td></tr>
      <tr><th>Static IP</th><td id="st-ip">{{if .Status.StaticIP}}{{.Status.StaticIP}}{{else}}-{{end}}</td></tr>
      <tr><th>Selected Template</th><td id="st-template">{{if .Status.TemplateName}}{{.Status.TemplateName}}{{else}}-{{end}}</td></tr>
      <tr><th>Box Type</th><td id="st-box">{{if .Status.BoxTypeName}}{{.Status.BoxTypeName}}{{else}}-{{end}}</td></tr>
      <tr><th>Status</th><td id="st-status">{{.Status.Status}}</td></tr>
      <tr><th>Generated Timestamp</th><td id="st-generated">{{fmtTime .Status.GeneratedAt}}</td></tr>
    </table>

    <div class="row">
      <button type="button" onclick="refreshStatus()">Refresh Status</button>
      <form method="post" action="/consume" id="consume-form">
        <button type="submit">Mark Consumed</button>
      </form>
      <form method="post" action="/force-replace" id="force-replace-form" onsubmit="return confirmForceReplace();" style="{{if .Status.Active}}display:block{{else}}display:none{{end}};">
        <button type="submit" class="danger">Force Replace Current</button>
      </form>
    </div>
  </div>

  <div class="card">
    <h2>Create New Provisioning Config</h2>
    <form method="post" action="/provision">
      <label>Template</label>
      <select name="template" required>
        {{range .TemplateNames}}<option value="{{.}}">{{.}}</option>{{end}}
      </select>

      <label>Box Type</label>
      <select name="box_type" required>
        {{range .BoxTypes}}<option value="{{.Name}}">{{.Name}} (bootstrap={{.BootstrapInterface}}, production={{.ProductionInterface}})</option>{{end}}
      </select>

      <label>Hostname</label>
      <input name="hostname" required placeholder="edge-001" />

      <label>Production NIC Static IP</label>
      <input name="static_ip" required placeholder="192.168.50.10" />

      <label>CIDR</label>
      <input name="cidr" value="24" />

      <label>Gateway</label>
      <input name="gateway" value="192.168.50.1" />

      <label>DNS Servers (comma-separated)</label>
      <input name="dns" value="1.1.1.1,8.8.8.8" />

      <button type="submit">Generate Provision Config</button>
    </form>
  </div>

<script>
function setText(id, value) {
  var el = document.getElementById(id);
  if (!el) return;
  el.textContent = value && value.length ? value : "-";
}

function applyStatus(st) {
  setText("st-hostname", st.hostname || "-");
  setText("st-ip", st.static_ip || "-");
  setText("st-template", st.template_name || "-");
  setText("st-box", st.box_type || "-");
  setText("st-status", st.status || "No active config");
  setText("st-generated", st.generated_at || "-");

  var forceForm = document.getElementById("force-replace-form");
  if (forceForm) {
    forceForm.style.display = st.active ? "block" : "none";
  }
}

function refreshStatus() {
  fetch('/status', {cache: 'no-store'})
    .then(function(res) { return res.json(); })
    .then(function(data) { applyStatus(data); })
    .catch(function(err) { console.error('status refresh failed', err); });
}

function confirmForceReplace() {
  return window.confirm('This will discard the current config. Continue?');
}

setInterval(refreshStatus, 7000);
</script>
</body>
</html>`))
