package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	StatusNoActive       = "No active config"
	StatusReady          = "Ready"
	StatusUserDataServed = "User-data served"
	StatusMetaDataServed = "Meta-data served"
	StatusConsumed       = "Consumed"
)

type BoxType struct {
	Name                string
	BootstrapInterface  string
	ProductionInterface string
}

type CloudInitTemplate struct {
	Name     string
	Filename string
	Raw      string
	Compiled TextTemplate
}

type RenderData struct {
	Hostname            string
	BootstrapInterface  string
	ProductionInterface string
	ProductionAddress   string
	Gateway             string
	DNS                 []string
}

type ActiveConfig struct {
	Hostname       string
	TemplateName   string
	BoxTypeName    string
	StaticIP       string
	CIDR           string
	Gateway        string
	DNS            []string
	CreatedAt      time.Time
	InstanceID     string
	UserData       string
	MetaData       string
	UserDataServed bool
	MetaDataServed bool
}

type ProvisionStatus struct {
	Hostname     string
	StaticIP     string
	TemplateName string
	BoxTypeName  string
	Status       string
	GeneratedAt  time.Time
	Active       bool
}

type Store struct {
	mu                sync.Mutex
	current           *ActiveConfig
	consumedHostnames map[string]time.Time
	status            ProvisionStatus
	stateFile         string
	queue             QueueState
	nextQueueID       int64
}

func NewStore() *Store {
	store, err := NewStoreWithPersistence("")
	if err != nil {
		panic(err)
	}
	return store
}

func NewStoreWithPersistence(stateFile string) (*Store, error) {
	s := &Store{
		consumedHostnames: make(map[string]time.Time),
		status: ProvisionStatus{
			Status: StatusNoActive,
			Active: false,
		},
		stateFile: strings.TrimSpace(stateFile),
		queue:     newQueueState(),
	}
	if err := s.loadState(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) SetCurrent(cfg *ActiveConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cfg == nil {
		return errors.New("nil config")
	}
	if _, exists := s.consumedHostnames[cfg.Hostname]; exists {
		return fmt.Errorf("hostname %q already consumed in this process; choose a new hostname", cfg.Hostname)
	}
	if s.queueContainsHostnameLocked(cfg.Hostname) {
		return fmt.Errorf("hostname %q already exists in active/pending queue", cfg.Hostname)
	}

	cfgCopy := *cfg
	if s.current == nil {
		s.setActiveLocked(cfgCopy)
		s.persistLocked("set_current")
		return nil
	}

	s.enqueuePendingLocked(cfgCopy)
	s.persistLocked("enqueue_pending")
	return nil
}

func (s *Store) GetCurrent() *ActiveConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil
	}
	copy := *s.current
	return &copy
}

func (s *Store) CurrentStatus() ProvisionStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Store) ServeUserData() (cfg *ActiveConfig, consumed bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return nil, false, errors.New("no active config")
	}
	s.current.UserDataServed = true
	copy := *s.current

	consumed, promoted, consumedCfg := s.maybeConsumeLocked()
	if consumed {
		if promoted {
			s.status = statusFromConfigLocked(s.current, StatusReady, true)
		} else {
			s.status = statusFromConfigValue(consumedCfg, StatusConsumed, false)
		}
	} else {
		s.status = statusFromConfigLocked(s.current, StatusUserDataServed, true)
	}
	s.persistLocked("serve_user_data")

	return &copy, consumed, nil
}

func (s *Store) ServeMetaData() (cfg *ActiveConfig, consumed bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return nil, false, errors.New("no active config")
	}
	s.current.MetaDataServed = true
	copy := *s.current

	consumed, promoted, consumedCfg := s.maybeConsumeLocked()
	if consumed {
		if promoted {
			s.status = statusFromConfigLocked(s.current, StatusReady, true)
		} else {
			s.status = statusFromConfigValue(consumedCfg, StatusConsumed, false)
		}
	} else {
		s.status = statusFromConfigLocked(s.current, StatusMetaDataServed, true)
	}
	s.persistLocked("serve_meta_data")

	return &copy, consumed, nil
}

func (s *Store) ManualConsume() (*ActiveConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, errors.New("no active config")
	}
	cfg := *s.current
	s.consumedHostnames[s.current.Hostname] = time.Now()
	s.finishActiveLocked(QueueStatusCompleted, "consumed_manual")
	if !s.promoteNextLocked() {
		s.status = statusFromConfigValue(cfg, StatusConsumed, false)
	}
	s.persistLocked("manual_consume")
	return &cfg, nil
}

func (s *Store) ForceReplace() (*ActiveConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, errors.New("no active config")
	}
	cfg := *s.current
	s.finishActiveLocked(QueueStatusFailed, "force_replaced")
	if !s.promoteNextLocked() {
		s.status = ProvisionStatus{Status: StatusNoActive, Active: false}
	}
	s.persistLocked("force_replace")
	return &cfg, nil
}

func (s *Store) maybeConsumeLocked() (consumed bool, promoted bool, consumedCfg ActiveConfig) {
	if s.current == nil {
		return false, false, ActiveConfig{}
	}
	if !(s.current.UserDataServed && s.current.MetaDataServed) {
		return false, false, ActiveConfig{}
	}
	consumedCfg = *s.current
	s.consumedHostnames[s.current.Hostname] = time.Now()
	s.finishActiveLocked(QueueStatusCompleted, "consumed")
	promoted = s.promoteNextLocked()
	return true, promoted, consumedCfg
}

func statusFromConfigLocked(cfg *ActiveConfig, state string, active bool) ProvisionStatus {
	if cfg == nil {
		return ProvisionStatus{Status: StatusNoActive, Active: false}
	}
	return statusFromConfigValue(*cfg, state, active)
}

func statusFromConfigValue(cfg ActiveConfig, state string, active bool) ProvisionStatus {
	return ProvisionStatus{
		Hostname:     cfg.Hostname,
		StaticIP:     fmt.Sprintf("%s/%s", cfg.StaticIP, cfg.CIDR),
		TemplateName: cfg.TemplateName,
		BoxTypeName:  cfg.BoxTypeName,
		Status:       state,
		GeneratedAt:  cfg.CreatedAt,
		Active:       active,
	}
}

func ParseDNS(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

type persistedState struct {
	Current           *ActiveConfig        `json:"current"`
	ConsumedHostnames map[string]time.Time `json:"consumed_hostnames"`
	Status            ProvisionStatus      `json:"status"`
	Queue             QueueState           `json:"queue"`
	NextQueueID       int64                `json:"next_queue_id"`
}

func (s *Store) persistLocked(action string) {
	if s.stateFile == "" {
		return
	}
	if err := s.saveStateLocked(); err != nil {
		log.Printf("store persistence warning action=%s err=%v", action, err)
	}
}

func (s *Store) loadState() error {
	if s.stateFile == "" {
		return nil
	}
	b, err := os.ReadFile(s.stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read state file: %w", err)
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil
	}

	var state persistedState
	if err := json.Unmarshal(b, &state); err != nil {
		return fmt.Errorf("unmarshal state file: %w", err)
	}
	if state.ConsumedHostnames == nil {
		state.ConsumedHostnames = make(map[string]time.Time)
	}
	if state.Queue.Pending == nil {
		state.Queue.Pending = []QueueItem{}
	}
	if state.Queue.Completed == nil {
		state.Queue.Completed = []QueueItem{}
	}
	if state.Queue.Failed == nil {
		state.Queue.Failed = []QueueItem{}
	}

	s.current = state.Current
	s.consumedHostnames = state.ConsumedHostnames
	if state.Status.Status == "" {
		if s.current != nil {
			s.status = statusFromConfigLocked(s.current, StatusReady, true)
		} else {
			s.status = ProvisionStatus{Status: StatusNoActive, Active: false}
		}
	} else {
		s.status = state.Status
	}
	s.queue = state.Queue
	s.nextQueueID = state.NextQueueID

	if s.queue.Active == nil && s.current != nil {
		id := fmt.Sprintf("q-%06d", s.nextQueueID+1)
		s.nextQueueID++
		item := queueItemFromConfig(id, *s.current, QueueStatusActive, s.current.CreatedAt)
		s.queue.Active = &item
	}
	if s.current == nil && s.queue.Active != nil {
		s.current = s.queue.Active.toActiveConfig()
	}
	if s.current != nil && (s.status.Status == "" || !s.status.Active) {
		s.status = statusFromConfigLocked(s.current, StatusReady, true)
	}
	if s.current == nil && s.status.Status == "" {
		s.status = ProvisionStatus{Status: StatusNoActive, Active: false}
	}

	return nil
}

func (s *Store) saveStateLocked() error {
	if s.stateFile == "" {
		return nil
	}

	state := persistedState{
		Current:           s.current,
		ConsumedHostnames: s.consumedHostnames,
		Status:            s.status,
		Queue:             s.queue,
		NextQueueID:       s.nextQueueID,
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := atomicWriteFile(s.stateFile, b, 0o600); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	return nil
}
