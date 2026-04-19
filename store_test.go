package main

import (
	"testing"
	"time"
)

func TestStoreStatusLifecycle(t *testing.T) {
	s := NewStore()

	status := s.CurrentStatus()
	if status.Status != StatusNoActive {
		t.Fatalf("initial status=%q want %q", status.Status, StatusNoActive)
	}

	cfg := &ActiveConfig{
		Hostname:     "edge-100",
		StaticIP:     "192.168.50.100",
		TemplateName: "example",
		BoxTypeName:  "nuc-dual-nic",
		CreatedAt:    time.Now(),
	}
	if err := s.SetCurrent(cfg); err != nil {
		t.Fatalf("SetCurrent error: %v", err)
	}

	status = s.CurrentStatus()
	if status.Status != StatusReady {
		t.Fatalf("status after set=%q want %q", status.Status, StatusReady)
	}

	if _, consumed, err := s.ServeUserData(); err != nil || consumed {
		t.Fatalf("ServeUserData err=%v consumed=%v", err, consumed)
	}
	status = s.CurrentStatus()
	if status.Status != StatusUserDataServed {
		t.Fatalf("status after user-data=%q want %q", status.Status, StatusUserDataServed)
	}

	if _, consumed, err := s.ServeMetaData(); err != nil || !consumed {
		t.Fatalf("ServeMetaData err=%v consumed=%v", err, consumed)
	}
	status = s.CurrentStatus()
	if status.Status != StatusConsumed {
		t.Fatalf("status after consume=%q want %q", status.Status, StatusConsumed)
	}
	if status.Hostname != "edge-100" {
		t.Fatalf("consumed hostname=%q want edge-100", status.Hostname)
	}
}

func TestForceReplaceClearsActiveAndResetsStatus(t *testing.T) {
	s := NewStore()
	cfg := &ActiveConfig{
		Hostname:     "edge-200",
		StaticIP:     "192.168.50.200",
		TemplateName: "example",
		BoxTypeName:  "nuc-dual-nic",
		CreatedAt:    time.Now(),
	}
	if err := s.SetCurrent(cfg); err != nil {
		t.Fatalf("SetCurrent error: %v", err)
	}

	oldCfg, err := s.ForceReplace()
	if err != nil {
		t.Fatalf("ForceReplace error: %v", err)
	}
	if oldCfg.Hostname != "edge-200" {
		t.Fatalf("force replace hostname=%q want edge-200", oldCfg.Hostname)
	}
	if s.GetCurrent() != nil {
		t.Fatal("expected current config to be cleared")
	}

	status := s.CurrentStatus()
	if status.Status != StatusNoActive {
		t.Fatalf("status after force replace=%q want %q", status.Status, StatusNoActive)
	}
}
