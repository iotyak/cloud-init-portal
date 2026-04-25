package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestHandleLogsAPIMethodGuard(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/logs", nil)
	rr := httptest.NewRecorder()

	srv.HandleLogsAPI(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleLogsAPIReturnsFilteredEvents(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "provision.log")
	logger, err := NewProvisionLogger(logPath)
	if err != nil {
		t.Fatalf("NewProvisionLogger error: %v", err)
	}
	t.Cleanup(func() { _ = logger.Close() })

	logger.LogEvent(&ActiveConfig{Hostname: "edge-a", TemplateName: "example", BoxTypeName: "nuc-dual-nic"}, "generated")
	logger.LogEvent(&ActiveConfig{Hostname: "edge-b", TemplateName: "example", BoxTypeName: "nuc-dual-nic"}, "generated")
	logger.LogEvent(&ActiveConfig{Hostname: "edge-a", TemplateName: "example", BoxTypeName: "nuc-dual-nic"}, "consumed")

	srv := newTestServer(t)
	srv.Logger = logger

	req := httptest.NewRequest(http.MethodGet, "/api/logs?event=generated&hostname=edge-a&limit=10", nil)
	rr := httptest.NewRecorder()
	srv.HandleLogsAPI(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%q", rr.Code, http.StatusOK, rr.Body.String())
	}

	var payload struct {
		Events []LogEvent `json:"events"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(payload.Events) != 1 {
		t.Fatalf("events len=%d want 1", len(payload.Events))
	}
	if payload.Events[0].Hostname != "edge-a" || payload.Events[0].Event != "generated" {
		t.Fatalf("unexpected payload event: %+v", payload.Events[0])
	}
}

func TestHandleLogsPage(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "provision.log")
	logger, err := NewProvisionLogger(logPath)
	if err != nil {
		t.Fatalf("NewProvisionLogger error: %v", err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	logger.LogEvent(&ActiveConfig{Hostname: "edge-a", TemplateName: "example", BoxTypeName: "nuc-dual-nic"}, "generated")

	srv := newTestServer(t)
	srv.Logger = logger

	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	rr := httptest.NewRecorder()
	srv.HandleLogsPage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusOK)
	}
	if body := rr.Body.String(); body == "" {
		t.Fatalf("expected logs page body")
	}
}
