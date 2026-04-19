package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistenceRoundTrip(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state", "portal-state.json")

	store, err := NewStoreWithPersistence(stateFile)
	if err != nil {
		t.Fatalf("NewStoreWithPersistence error: %v", err)
	}

	cfg := &ActiveConfig{
		Hostname:     "edge-persist-1",
		StaticIP:     "192.168.50.210",
		CIDR:         "24",
		TemplateName: "example",
		BoxTypeName:  "nuc-dual-nic",
		CreatedAt:    time.Now(),
	}
	if err := store.SetCurrent(cfg); err != nil {
		t.Fatalf("SetCurrent error: %v", err)
	}

	reloaded, err := NewStoreWithPersistence(stateFile)
	if err != nil {
		t.Fatalf("reloaded NewStoreWithPersistence error: %v", err)
	}
	got := reloaded.GetCurrent()
	if got == nil || got.Hostname != "edge-persist-1" {
		t.Fatalf("reloaded current=%+v", got)
	}

	if _, err := reloaded.ManualConsume(); err != nil {
		t.Fatalf("ManualConsume error: %v", err)
	}
	if reloaded.GetCurrent() != nil {
		t.Fatal("expected no active config after manual consume")
	}

	reloadedAgain, err := NewStoreWithPersistence(stateFile)
	if err != nil {
		t.Fatalf("third NewStoreWithPersistence error: %v", err)
	}
	if reloadedAgain.GetCurrent() != nil {
		t.Fatal("expected no active config after reload")
	}
	if reloadedAgain.CurrentStatus().Status != StatusConsumed {
		t.Fatalf("status=%q want %q", reloadedAgain.CurrentStatus().Status, StatusConsumed)
	}
}
