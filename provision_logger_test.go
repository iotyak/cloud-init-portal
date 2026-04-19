package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProvisionLoggerWritesEvent(t *testing.T) {
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
		TemplateName: "example",
		BoxTypeName:  "nuc-dual-nic",
		DNS:          []string{"1.1.1.1", "8.8.8.8"},
	}
	logger.LogEvent(cfg, "generated")
	_ = logger.Close()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	line := string(b)
	for _, expect := range []string{"event=generated", "hostname=edge-777", "template=example", "dns=1.1.1.1,8.8.8.8"} {
		if !strings.Contains(line, expect) {
			t.Fatalf("log line missing %q in %q", expect, line)
		}
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
}
