package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProvisionLoggerWritesJSONEvent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provision.log")

	logger, err := NewProvisionLogger(path)
	if err != nil {
		t.Fatalf("NewProvisionLogger error: %v", err)
	}
	defer logger.Close()

	cfg := &ActiveConfig{
		Hostname:     "edge-777",
		StaticIP:     "192.168.50.77",
		CIDR:         "24",
		Gateway:      "192.168.50.1",
		TemplateName: "example",
		BoxTypeName:  "nuc-dual-nic",
		DNS:          []string{"1.1.1.1", "8.8.8.8"},
	}
	logger.LogEvent(cfg, "generated")
	_ = logger.Close()

	events, err := logger.ReadEvents(10, "", "")
	if err != nil {
		t.Fatalf("ReadEvents error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len=%d want 1", len(events))
	}

	e := events[0]
	if e.Event != "generated" {
		t.Fatalf("event=%q want generated", e.Event)
	}
	if e.Hostname != "edge-777" {
		t.Fatalf("hostname=%q want edge-777", e.Hostname)
	}
	if e.TemplateName != "example" {
		t.Fatalf("template=%q want example", e.TemplateName)
	}
	if len(e.DNS) != 2 {
		t.Fatalf("dns len=%d want 2", len(e.DNS))
	}
}

func TestProvisionLoggerReadEventsFilter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provision.log")

	logger, err := NewProvisionLogger(path)
	if err != nil {
		t.Fatalf("NewProvisionLogger error: %v", err)
	}
	defer logger.Close()

	logger.LogEvent(&ActiveConfig{Hostname: "edge-1", TemplateName: "example", BoxTypeName: "nuc-dual-nic"}, "generated")
	logger.LogEvent(&ActiveConfig{Hostname: "edge-2", TemplateName: "example", BoxTypeName: "nuc-dual-nic"}, "generated")
	logger.LogEvent(&ActiveConfig{Hostname: "edge-1", TemplateName: "example", BoxTypeName: "nuc-dual-nic"}, "consumed")

	events, err := logger.ReadEvents(50, "generated", "edge-1")
	if err != nil {
		t.Fatalf("ReadEvents error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len=%d want 1", len(events))
	}
	if events[0].Hostname != "edge-1" || events[0].Event != "generated" {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}

func TestProvisionLoggerNilSafety(t *testing.T) {
	var logger *ProvisionLogger
	logger.LogEvent(&ActiveConfig{Hostname: "edge-888"}, "generated")
	if err := logger.Close(); err != nil {
		t.Fatalf("nil logger close err=%v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "provision.log")
	logger2, err := NewProvisionLogger(path)
	if err != nil {
		t.Fatalf("NewProvisionLogger error: %v", err)
	}
	logger2.LogEvent(nil, "generated")
	_ = logger2.Close()

	if _, err := os.ReadFile(path); err != nil {
		t.Fatalf("expected readable log file: %v", err)
	}
}
