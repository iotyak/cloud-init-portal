package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	htmltmpl "html/template"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	Store     *Store
	Templates map[string]CloudInitTemplate
	BoxTypes  map[string]BoxType
	Logger    *ProvisionLogger
}

type indexData struct {
	TemplateNames []string
	BoxTypes      []BoxType
	Current       *ActiveConfig
	Status        ProvisionStatus
	Error         string
	Message       string
	Success       *successData
}

type successData struct {
	Hostname     string
	TemplateName string
	BoxTypeName  string
	UserDataURL  string
	MetaDataURL  string
	IPXEExample  string
	CurlExample  string
}

type statusPayload struct {
	Hostname     string `json:"hostname"`
	StaticIP     string `json:"static_ip"`
	TemplateName string `json:"template_name"`
	BoxTypeName  string `json:"box_type"`
	Status       string `json:"status"`
	GeneratedAt  string `json:"generated_at"`
	Active       bool   `json:"active"`
}

var hostnameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,62}$`)

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.renderIndex(w, indexData{
		TemplateNames: TemplateNames(s.Templates),
		BoxTypes:      sortedBoxTypes(s.BoxTypes),
		Current:       s.Store.GetCurrent(),
		Status:        s.Store.CurrentStatus(),
	})
}

func (s *Server) HandleProvision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, fmt.Sprintf("invalid form: %v", err), http.StatusBadRequest)
		return
	}

	templateName := strings.TrimSpace(r.FormValue("template"))
	boxTypeName := strings.TrimSpace(r.FormValue("box_type"))
	hostname := strings.TrimSpace(r.FormValue("hostname"))
	staticIP := strings.TrimSpace(r.FormValue("static_ip"))
	cidr := strings.TrimSpace(r.FormValue("cidr"))
	gateway := strings.TrimSpace(r.FormValue("gateway"))
	dns := ParseDNS(r.FormValue("dns"))
	if len(dns) == 0 {
		dns = []string{"10.4.99.99", "10.6.99.99"}
	}

	if err := validateInput(templateName, boxTypeName, hostname, staticIP, cidr, gateway, dns, s.Templates, s.BoxTypes); err != nil {
		s.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}

	prodAddress := fmt.Sprintf("%s/%s", staticIP, cidr)
	box := s.BoxTypes[boxTypeName]
	tpl := s.Templates[templateName]

	renderData := RenderData{
		Hostname:            hostname,
		BootstrapInterface:  box.BootstrapInterface,
		ProductionInterface: box.ProductionInterface,
		ProductionAddress:   prodAddress,
		Gateway:             gateway,
		DNS:                 dns,
	}

	userData, err := renderTemplate(tpl, renderData)
	if err != nil {
		s.renderError(w, fmt.Sprintf("render template failed: %v", err), http.StatusBadRequest)
		return
	}

	instanceID := fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
	cfg := &ActiveConfig{
		Hostname:     hostname,
		TemplateName: templateName,
		BoxTypeName:  boxTypeName,
		StaticIP:     staticIP,
		CIDR:         cidr,
		Gateway:      gateway,
		DNS:          dns,
		CreatedAt:    time.Now(),
		InstanceID:   instanceID,
		UserData:     userData,
		MetaData:     fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", instanceID, hostname),
	}

	if err := s.Store.SetCurrent(cfg); err != nil {
		s.renderError(w, err.Error(), http.StatusConflict)
		return
	}
	s.Logger.LogEvent(cfg, "generated")

	baseURL := requestBaseURL(r)
	userDataURL := baseURL + "/user-data"
	metaDataURL := baseURL + "/meta-data"
	seedURL := fmt.Sprintf("%s/", baseURL)

	s.renderIndex(w, indexData{
		TemplateNames: TemplateNames(s.Templates),
		BoxTypes:      sortedBoxTypes(s.BoxTypes),
		Current:       s.Store.GetCurrent(),
		Status:        s.Store.CurrentStatus(),
		Message:       fmt.Sprintf("Config ready for box: %s", hostname),
		Success: &successData{
			Hostname:     hostname,
			TemplateName: templateName,
			BoxTypeName:  boxTypeName,
			UserDataURL:  userDataURL,
			MetaDataURL:  metaDataURL,
			IPXEExample:  fmt.Sprintf("kernel ... ds=nocloud-net;s=%s", seedURL),
			CurlExample:  fmt.Sprintf("curl -fsSL %s && curl -fsSL %s", userDataURL, metaDataURL),
		},
	})
}

func (s *Server) HandleConsume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := s.Store.ManualConsume()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusConflict)
		return
	}
	s.Logger.LogEvent(cfg, "consumed_manual")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) HandleForceReplace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := s.Store.ForceReplace()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusConflict)
		return
	}
	s.Logger.LogEvent(cfg, "force_replaced")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	status := s.Store.CurrentStatus()
	payload := statusPayload{
		Hostname:     status.Hostname,
		StaticIP:     status.StaticIP,
		TemplateName: status.TemplateName,
		BoxTypeName:  status.BoxTypeName,
		Status:       status.Status,
		GeneratedAt:  formatTime(status.GeneratedAt),
		Active:       status.Active,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) HandleUserData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, consumed, err := s.Store.ServeUserData()
	if err != nil {
		http.Error(w, "no active config", http.StatusNotFound)
		return
	}
	s.Logger.LogEvent(cfg, "served_user_data")
	if consumed {
		s.Logger.LogEvent(cfg, "consumed")
	}
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	_, _ = w.Write([]byte(cfg.UserData))
}

func (s *Server) HandleMetaData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, consumed, err := s.Store.ServeMetaData()
	if err != nil {
		http.Error(w, "no active config", http.StatusNotFound)
		return
	}
	s.Logger.LogEvent(cfg, "served_meta_data")
	if consumed {
		s.Logger.LogEvent(cfg, "consumed")
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(cfg.MetaData))
}

func renderTemplate(t CloudInitTemplate, data RenderData) (string, error) {
	var buf bytes.Buffer
	if err := t.Compiled.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func validateInput(templateName, boxTypeName, hostname, staticIP, cidr, gateway string, dns []string, templates map[string]CloudInitTemplate, boxTypes map[string]BoxType) error {
	if _, ok := templates[templateName]; !ok {
		return errors.New("unknown template")
	}
	if _, ok := boxTypes[boxTypeName]; !ok {
		return errors.New("unknown box type")
	}
	if !hostnameRe.MatchString(hostname) {
		return errors.New("invalid hostname (use letters, numbers, dash; max 63 chars)")
	}
	if ip := net.ParseIP(staticIP); ip == nil {
		return errors.New("invalid static IP")
	}
	n, err := strconv.Atoi(cidr)
	if err != nil || n < 1 || n > 32 {
		return errors.New("invalid CIDR (expected 1-32)")
	}
	if strings.TrimSpace(gateway) != "" {
		if ip := net.ParseIP(gateway); ip == nil {
			return errors.New("invalid gateway IP")
		}
	}
	for _, dnsIP := range dns {
		if ip := net.ParseIP(strings.TrimSpace(dnsIP)); ip == nil {
			return fmt.Errorf("invalid DNS server IP: %s", dnsIP)
		}
	}
	return nil
}

func requestBaseURL(r *http.Request) string {
	host := r.Host
	if host == "" {
		host = "127.0.0.1:8080"
	}
	return "http://" + host
}

func sortedBoxTypes(boxTypes map[string]BoxType) []BoxType {
	names := BoxTypeNames(boxTypes)
	out := make([]BoxType, 0, len(names))
	for _, n := range names {
		out = append(out, boxTypes[n])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

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
