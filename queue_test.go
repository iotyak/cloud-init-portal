package main

import (
	"path/filepath"
	"testing"
	"time"
)

func testConfig(hostname string) *ActiveConfig {
	return &ActiveConfig{
		Hostname:     hostname,
		StaticIP:     "192.168.50.10",
		CIDR:         "24",
		Gateway:      "192.168.50.1",
		TemplateName: "example",
		BoxTypeName:  "nuc-dual-nic",
		CreatedAt:    time.Now(),
		InstanceID:   hostname + "-id",
	}
}

func TestQueueFIFOAndAutoPromotionOnManualConsume(t *testing.T) {
	s := NewStore()
	if err := s.SetCurrent(testConfig("edge-001")); err != nil {
		t.Fatalf("SetCurrent(edge-001) error: %v", err)
	}
	if err := s.SetCurrent(testConfig("edge-002")); err != nil {
		t.Fatalf("SetCurrent(edge-002) error: %v", err)
	}
	if err := s.SetCurrent(testConfig("edge-003")); err != nil {
		t.Fatalf("SetCurrent(edge-003) error: %v", err)
	}

	snap := s.QueueSnapshot()
	if snap.Active == nil || snap.Active.Hostname != "edge-001" {
		t.Fatalf("active=%+v want edge-001", snap.Active)
	}
	if len(snap.Pending) != 2 || snap.Pending[0].Hostname != "edge-002" || snap.Pending[1].Hostname != "edge-003" {
		t.Fatalf("pending=%+v", snap.Pending)
	}

	consumed, err := s.ManualConsume()
	if err != nil {
		t.Fatalf("ManualConsume error: %v", err)
	}
	if consumed.Hostname != "edge-001" {
		t.Fatalf("consumed hostname=%q want edge-001", consumed.Hostname)
	}

	current := s.GetCurrent()
	if current == nil || current.Hostname != "edge-002" {
		t.Fatalf("current=%+v want edge-002", current)
	}

	snap = s.QueueSnapshot()
	if len(snap.Completed) != 1 || snap.Completed[0].Hostname != "edge-001" {
		t.Fatalf("completed=%+v", snap.Completed)
	}
	if len(snap.Pending) != 1 || snap.Pending[0].Hostname != "edge-003" {
		t.Fatalf("pending after promote=%+v", snap.Pending)
	}
}

func TestQueueAutoPromotionOnServeLifecycleConsume(t *testing.T) {
	s := NewStore()
	if err := s.SetCurrent(testConfig("edge-010")); err != nil {
		t.Fatalf("SetCurrent(edge-010) error: %v", err)
	}
	if err := s.SetCurrent(testConfig("edge-011")); err != nil {
		t.Fatalf("SetCurrent(edge-011) error: %v", err)
	}

	if _, consumed, err := s.ServeUserData(); err != nil || consumed {
		t.Fatalf("ServeUserData err=%v consumed=%v", err, consumed)
	}
	if _, consumed, err := s.ServeMetaData(); err != nil || !consumed {
		t.Fatalf("ServeMetaData err=%v consumed=%v", err, consumed)
	}

	current := s.GetCurrent()
	if current == nil || current.Hostname != "edge-011" {
		t.Fatalf("current after auto consume=%+v want edge-011", current)
	}
	status := s.CurrentStatus()
	if !status.Active || status.Status != StatusReady {
		t.Fatalf("status=%+v want active ready", status)
	}
}

func TestQueueStatePersistsAcrossReload(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state", "portal-state.json")
	store, err := NewStoreWithPersistence(stateFile)
	if err != nil {
		t.Fatalf("NewStoreWithPersistence error: %v", err)
	}

	if err := store.SetCurrent(testConfig("edge-a")); err != nil {
		t.Fatalf("SetCurrent(edge-a) error: %v", err)
	}
	if err := store.SetCurrent(testConfig("edge-b")); err != nil {
		t.Fatalf("SetCurrent(edge-b) error: %v", err)
	}

	reloaded, err := NewStoreWithPersistence(stateFile)
	if err != nil {
		t.Fatalf("reloaded NewStoreWithPersistence error: %v", err)
	}
	snap := reloaded.QueueSnapshot()
	if snap.Active == nil || snap.Active.Hostname != "edge-a" {
		t.Fatalf("reloaded active=%+v", snap.Active)
	}
	if len(snap.Pending) != 1 || snap.Pending[0].Hostname != "edge-b" {
		t.Fatalf("reloaded pending=%+v", snap.Pending)
	}
}

func TestCancelPendingQueueItem(t *testing.T) {
	s := NewStore()
	if err := s.SetCurrent(testConfig("edge-x")); err != nil {
		t.Fatalf("SetCurrent(edge-x) error: %v", err)
	}
	if err := s.SetCurrent(testConfig("edge-y")); err != nil {
		t.Fatalf("SetCurrent(edge-y) error: %v", err)
	}

	snap := s.QueueSnapshot()
	if len(snap.Pending) != 1 {
		t.Fatalf("pending len=%d want 1", len(snap.Pending))
	}
	itemID := snap.Pending[0].ID
	cancelled, err := s.CancelQueueItem(itemID)
	if err != nil {
		t.Fatalf("CancelQueueItem error: %v", err)
	}
	if cancelled.Hostname != "edge-y" {
		t.Fatalf("cancelled=%+v", cancelled)
	}

	snap = s.QueueSnapshot()
	if len(snap.Pending) != 0 {
		t.Fatalf("pending len=%d want 0", len(snap.Pending))
	}
	if len(snap.Failed) != 1 || snap.Failed[0].ID != itemID {
		t.Fatalf("failed=%+v", snap.Failed)
	}
}
