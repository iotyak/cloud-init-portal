package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
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

type Store struct {
	mu                sync.Mutex
	current           *ActiveConfig
	consumedHostnames map[string]time.Time
}

func NewStore() *Store {
	return &Store{consumedHostnames: make(map[string]time.Time)}
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

	s.current = cfg
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

func (s *Store) ServeUserData() (cfg *ActiveConfig, consumed bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return nil, false, errors.New("no active config")
	}
	s.current.UserDataServed = true
	copy := *s.current
	consumed = s.maybeConsumeLocked()
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
	consumed = s.maybeConsumeLocked()
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
	s.current = nil
	return &cfg, nil
}

func (s *Store) maybeConsumeLocked() bool {
	if s.current == nil {
		return false
	}
	if !(s.current.UserDataServed && s.current.MetaDataServed) {
		return false
	}
	s.consumedHostnames[s.current.Hostname] = time.Now()
	s.current = nil
	return true
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
