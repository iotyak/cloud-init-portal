package main

import (
	"bytes"
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
	Error         string
	Message       string
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
		dns = []string{"1.1.1.1", "8.8.8.8"}
	}

	if err := validateInput(templateName, boxTypeName, hostname, staticIP, cidr, s.Templates, s.BoxTypes); err != nil {
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

	s.renderSuccess(w, successData{
		Hostname:     hostname,
		TemplateName: templateName,
		BoxTypeName:  boxTypeName,
		UserDataURL:  userDataURL,
		MetaDataURL:  metaDataURL,
		IPXEExample:  fmt.Sprintf("kernel ... ds=nocloud-net;s=%s", seedURL),
		CurlExample:  fmt.Sprintf("curl -fsSL %s && curl -fsSL %s", userDataURL, metaDataURL),
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

func validateInput(templateName, boxTypeName, hostname, staticIP, cidr string, templates map[string]CloudInitTemplate, boxTypes map[string]BoxType) error {
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
		Error:         message,
	})
}

func (s *Server) renderIndex(w http.ResponseWriter, data indexData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = indexPageTemplate.Execute(w, data)
}

func (s *Server) renderSuccess(w http.ResponseWriter, data successData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = successPageTemplate.Execute(w, data)
}

var indexPageTemplate = htmltmpl.Must(htmltmpl.New("index").Parse(`<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <title>Cloud-Init Portal</title>
  <style>
    body { font-family: sans-serif; max-width: 860px; margin: 2rem auto; line-height: 1.4; }
    label { display: block; margin-top: 0.6rem; font-weight: 600; }
    input, select { width: 100%; padding: 0.45rem; }
    button { margin-top: 1rem; padding: 0.6rem 1rem; }
    pre { background: #111; color: #eee; padding: 0.8rem; overflow-x: auto; }
    .err { background: #ffe9e9; border: 1px solid #e88; padding: 0.7rem; }
    .ok { background: #e9fff0; border: 1px solid #7c7; padding: 0.7rem; }
    .card { border: 1px solid #ddd; padding: 0.8rem; margin-top: 1rem; }
  </style>
</head>
<body>
  <h1>Cloud-Init Provision Portal</h1>
  <p>One active config at a time. Create a config, then let target fetch <code>/user-data</code> and <code>/meta-data</code>.</p>

  {{if .Error}}<div class="err"><strong>Error:</strong> {{.Error}}</div>{{end}}
  {{if .Message}}<div class="ok">{{.Message}}</div>{{end}}

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

  {{if .Current}}
  <div class="card">
    <h2>Current Active Config</h2>
    <ul>
      <li>hostname: <code>{{.Current.Hostname}}</code></li>
      <li>template: <code>{{.Current.TemplateName}}</code></li>
      <li>box type: <code>{{.Current.BoxTypeName}}</code></li>
      <li>production IP: <code>{{.Current.StaticIP}}/{{.Current.CIDR}}</code></li>
      <li>served user-data: <code>{{.Current.UserDataServed}}</code></li>
      <li>served meta-data: <code>{{.Current.MetaDataServed}}</code></li>
    </ul>
    <form method="post" action="/consume">
      <button type="submit">Mark Consumed / Clear Active Config</button>
    </form>
  </div>
  {{end}}
</body>
</html>`))

var successPageTemplate = htmltmpl.Must(htmltmpl.New("success").Parse(`<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <title>Provisioning Ready</title>
  <style>
    body { font-family: sans-serif; max-width: 860px; margin: 2rem auto; line-height: 1.4; }
    pre { background: #111; color: #eee; padding: 0.8rem; overflow-x: auto; }
  </style>
</head>
<body>
  <h1>Provisioning Config Generated</h1>
  <p>Hostname <strong>{{.Hostname}}</strong> is ready.</p>
  <ul>
    <li>Template: <code>{{.TemplateName}}</code></li>
    <li>Box type: <code>{{.BoxTypeName}}</code></li>
    <li>User-data URL: <a href="{{.UserDataURL}}">{{.UserDataURL}}</a></li>
    <li>Meta-data URL: <a href="{{.MetaDataURL}}">{{.MetaDataURL}}</a></li>
  </ul>

  <h2>Suggested iPXE kernel arg</h2>
  <pre>{{.IPXEExample}}</pre>

  <h2>Suggested fetch test</h2>
  <pre>{{.CurlExample}}</pre>

  <p><a href="/">Back to portal</a></p>
</body>
</html>`))
