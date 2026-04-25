package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

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

func (s *Server) HandleLogsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.Logger == nil {
		http.Error(w, "logger unavailable", http.StatusServiceUnavailable)
		return
	}

	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	eventFilter := r.URL.Query().Get("event")
	hostnameFilter := r.URL.Query().Get("hostname")

	events, err := s.Logger.ReadEvents(limit, eventFilter, hostnameFilter)
	if err != nil {
		http.Error(w, "failed to read logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(logsPayload{Events: events})
}
