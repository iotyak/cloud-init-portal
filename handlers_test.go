package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatusEndpoint(t *testing.T) {
	store := NewStore()
	if err := store.SetCurrent(&ActiveConfig{
		Hostname:     "edge-300",
		StaticIP:     "192.168.50.300",
		TemplateName: "example",
		BoxTypeName:  "nuc-dual-nic",
		CreatedAt:    time.Now(),
	}); err != nil {
		t.Fatalf("SetCurrent error: %v", err)
	}

	s := &Server{Store: store}
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rr := httptest.NewRecorder()

	s.HandleStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code=%d want 200", rr.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if payload["status"] != StatusReady {
		t.Fatalf("payload status=%v want %q", payload["status"], StatusReady)
	}
	if payload["hostname"] != "edge-300" {
		t.Fatalf("payload hostname=%v want edge-300", payload["hostname"])
	}
}
