package main

import (
	"fmt"
	"net/http"
	"time"
)

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

	templateName := r.FormValue("template")
	boxTypeName := r.FormValue("box_type")
	hostname := r.FormValue("hostname")
	staticIP := r.FormValue("static_ip")
	cidr := r.FormValue("cidr")
	gateway := r.FormValue("gateway")
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

	baseURL := requestBaseURL(r, s.PublicBaseURL, s.TrustProxyHeaders)
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
