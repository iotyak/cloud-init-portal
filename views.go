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
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Cloud-Init Portal</title>
  <style>
    :root {
      --color-primary: #111111;
      --color-secondary: #555555;
      --color-surface: #ffffff;
      --color-surface-subtle: #f5f5f5;
      --color-border: #dddddd;
      --color-error-bg: #ffe9e9;
      --color-success-bg: #e9fff0;
      --color-code-bg: #111111;
      --color-code-fg: #eeeeee;
      --color-danger: #b40000;
      --color-danger-fg: #ffffff;

      --radius-sm: 4px;
      --radius-md: 8px;

      --space-label-top: 0.6rem;
      --space-control-pad: 0.55rem;
      --space-card-pad: 0.9rem;
      --space-card-gap: 1rem;
      --space-stack-gap: 0.7rem;
      --space-table-cell: 0.55rem;
      --space-code-pad: 0.8rem;

      --font-body: sans-serif;
      --font-code: monospace;
    }

    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: var(--font-body);
      color: var(--color-primary);
      line-height: 1.4;
      font-size: 17px;
      background: var(--color-surface);
    }

    main {
      max-width: 980px;
      margin: 2rem auto;
      padding: 0 1rem;
    }

    h1 { font-size: 2em; margin: 0 0 0.35rem; }
    h2 { font-size: 1.5em; margin: 0 0 0.6rem; }
    h3 { font-size: 1.17em; margin: 0.6rem 0; }
    p { margin: 0 0 0.8rem; }
    .muted { color: var(--color-secondary); }

    .stack { display: grid; gap: var(--space-card-gap); }

    .card {
      border: 1px solid var(--color-border);
      border-radius: var(--radius-md);
      padding: var(--space-card-pad);
      background: var(--color-surface);
    }

    .feedback {
      margin-bottom: var(--space-stack-gap);
      border: 1px solid var(--color-border);
      padding: 0.7rem;
      border-radius: var(--radius-sm);
    }
    .feedback.err { background: var(--color-error-bg); }
    .feedback.ok { background: var(--color-success-bg); font-size: 18px; }

    .form-grid {
      display: grid;
      grid-template-columns: 1fr;
      gap: var(--space-stack-gap);
    }

    .field {
      display: grid;
      grid-template-columns: 1fr;
      gap: 0.4rem;
    }

    label {
      display: block;
      margin-top: var(--space-label-top);
      font-weight: 700;
      font-size: 16px;
      line-height: 1.3;
    }

    input, select {
      width: 100%;
      padding: var(--space-control-pad);
      border: 1px solid var(--color-border);
      border-radius: var(--radius-sm);
      font-size: 16px;
      line-height: 1.2;
      color: var(--color-primary);
      background: var(--color-surface);
    }

    .actions {
      display: flex;
      flex-wrap: wrap;
      gap: var(--space-stack-gap);
      margin-top: 0.8rem;
    }

    button {
      margin-top: 0.8rem;
      padding: 0.6rem 1rem;
      border: 1px solid var(--color-primary);
      border-radius: var(--radius-sm);
      font-size: 16px;
      line-height: 1.2;
      cursor: pointer;
      background: var(--color-surface);
      color: var(--color-primary);
    }

    button.primary {
      background: var(--color-primary);
      color: #ffffff;
      font-weight: 700;
      letter-spacing: 0.02em;
      text-transform: uppercase;
    }

    button.danger {
      background: var(--color-danger);
      color: var(--color-danger-fg);
      border: none;
      border-radius: 0;
    }

    hr {
      border: 0;
      border-top: 1px solid var(--color-border);
      margin: 0.7rem 0;
    }

    .status-table {
      width: 100%;
      border-collapse: collapse;
    }
    .status-table th,
    .status-table td {
      border: 1px solid var(--color-border);
      padding: var(--space-table-cell);
      text-align: left;
      vertical-align: top;
    }
    .status-table th {
      width: 220px;
      background: var(--color-surface-subtle);
      font-weight: 700;
    }

    pre {
      background: var(--color-code-bg);
      color: var(--color-code-fg);
      padding: var(--space-code-pad);
      overflow-x: auto;
      border-radius: var(--radius-sm);
      font-family: var(--font-code);
      font-size: 14.4px;
      line-height: 1.4;
    }

    @media (min-width: 840px) {
      .field {
        grid-template-columns: 240px 1fr;
        align-items: center;
      }
      .field label {
        margin-top: 0;
      }
    }
  </style>
</head>
<body>
  <main>
    <header>
      <h1>Cloud-Init Provision Portal</h1>
      <p class="muted">Configure exactly one provisioning request, then review generated details and consumed state.</p>
      <p><a href="/logs">View Event History</a></p>
    </header>

    {{if .Error}}<div class="feedback err"><strong>Error:</strong> {{.Error}}</div>{{end}}
    {{if .Message}}<div class="feedback ok"><strong>{{.Message}}</strong></div>{{end}}

    <div class="stack">
      <section class="card" id="config-card">
        <h2>Create Provisioning Config</h2>
        <form method="post" action="/provision" class="form-grid">
          <div class="field">
            <label for="template">Cloud-init Template</label>
            <select id="template" name="template" required>
              {{range .TemplateNames}}<option value="{{.}}">{{.}}</option>{{end}}
            </select>
          </div>

          <div class="field">
            <label for="box_type">Box Type (HW Vendor/Model)</label>
            <select id="box_type" name="box_type" required>
              {{range .BoxTypes}}<option value="{{.Name}}">{{.Name}} (bootstrap={{.BootstrapInterface}}, production={{.ProductionInterface}})</option>{{end}}
            </select>
          </div>

          <div class="field">
            <label for="hostname">Hostname</label>
            <input id="hostname" name="hostname" required placeholder="edge-001" />
          </div>

          <hr />

          <div class="field">
            <label for="static_ip">IP Address</label>
            <input id="static_ip" name="static_ip" required placeholder="192.168.50.10" />
          </div>

          <div class="field">
            <label for="cidr">CIDR</label>
            <input id="cidr" name="cidr" value="24" />
          </div>

          <div class="field">
            <label for="gateway">Gateway</label>
            <input id="gateway" name="gateway" value="192.168.50.1" />
          </div>

          <div class="field">
            <label for="dns">DNS Servers</label>
            <input id="dns" name="dns" value="1.1.1.1,8.8.8.8" />
          </div>

          <div class="actions">
            <button type="submit" class="primary">Generate Config</button>
          </div>
        </form>
      </section>

      <section class="card" id="status-card">
        <h2>Generated Config Details &amp; Consumption Status</h2>
        <table class="status-table">
          <tr><th>Hostname</th><td id="st-hostname">{{if .Status.Hostname}}{{.Status.Hostname}}{{else}}-{{end}}</td></tr>
          <tr><th>IP Address</th><td id="st-ip">{{if .Status.StaticIP}}{{.Status.StaticIP}}{{else}}-{{end}}</td></tr>
          <tr><th>Template</th><td id="st-template">{{if .Status.TemplateName}}{{.Status.TemplateName}}{{else}}-{{end}}</td></tr>
          <tr><th>Box Type</th><td id="st-box">{{if .Status.BoxTypeName}}{{.Status.BoxTypeName}}{{else}}-{{end}}</td></tr>
          <tr><th>Generated Timestamp</th><td id="st-generated">{{fmtTime .Status.GeneratedAt}}</td></tr>
          <tr><th>Consumed</th><td id="st-consumed">{{if eq .Status.Status "Consumed"}}Yes{{else}}No{{end}}</td></tr>
          <tr><th>Status</th><td id="st-status">{{.Status.Status}}</td></tr>
        </table>

        <div class="actions">
          <button type="button" onclick="refreshStatus()">Refresh Status</button>
          <form method="post" action="/consume" id="consume-form">
            <button type="submit">Mark Consumed</button>
          </form>
          <form method="post" action="/force-replace" id="force-replace-form" onsubmit="return confirmForceReplace();" style="{{if .Status.Active}}display:block{{else}}display:none{{end}};">
            <button type="submit" class="danger">Force Replace Current</button>
          </form>
        </div>
      </section>

      {{if .Success}}
      <section class="card" id="generated-card">
        <h2>Generated Provisioning Output</h2>
        <p><strong>Config ready for:</strong> <code>{{.Success.Hostname}}</code></p>
        <ul>
          <li>User-data URL: <a href="{{.Success.UserDataURL}}">{{.Success.UserDataURL}}</a></li>
          <li>Meta-data URL: <a href="{{.Success.MetaDataURL}}">{{.Success.MetaDataURL}}</a></li>
        </ul>
        <h3>Suggested iPXE Kernel Arg</h3>
        <pre>{{.Success.IPXEExample}}</pre>
        <h3>Suggested Fetch Test</h3>
        <pre>{{.Success.CurlExample}}</pre>
      </section>
      {{end}}
    </div>
  </main>

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
  setText("st-consumed", st.status === "Consumed" ? "Yes" : "No");

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
