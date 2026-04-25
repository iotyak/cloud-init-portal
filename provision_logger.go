package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type LogEvent struct {
	Timestamp    string   `json:"timestamp"`
	Event        string   `json:"event"`
	Hostname     string   `json:"hostname"`
	StaticIP     string   `json:"static_ip,omitempty"`
	CIDR         string   `json:"cidr,omitempty"`
	Gateway      string   `json:"gateway,omitempty"`
	TemplateName string   `json:"template_name,omitempty"`
	BoxTypeName  string   `json:"box_type,omitempty"`
	DNS          []string `json:"dns,omitempty"`
}

type ProvisionLogger struct {
	mu   sync.Mutex
	f    *os.File
	path string
}

func NewProvisionLogger(path string) (*ProvisionLogger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &ProvisionLogger{f: f, path: path}, nil
}

func (l *ProvisionLogger) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	return l.f.Close()
}

func (l *ProvisionLogger) LogEvent(cfg *ActiveConfig, event string) {
	if l == nil || cfg == nil {
		return
	}

	entry := LogEvent{
		Timestamp:    time.Now().Format(time.RFC3339),
		Event:        event,
		Hostname:     cfg.Hostname,
		StaticIP:     cfg.StaticIP,
		CIDR:         cfg.CIDR,
		Gateway:      cfg.Gateway,
		TemplateName: cfg.TemplateName,
		BoxTypeName:  cfg.BoxTypeName,
		DNS:          cfg.DNS,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = fmt.Fprintf(l.f, "%s\n", string(line))
}

func (l *ProvisionLogger) ReadEvents(limit int, eventFilter string, hostnameFilter string) ([]LogEvent, error) {
	if l == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	rf, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEvent{}, nil
		}
		return nil, err
	}
	defer rf.Close()

	eventFilter = strings.TrimSpace(eventFilter)
	hostnameFilter = strings.TrimSpace(hostnameFilter)

	var all []LogEvent
	s := bufio.NewScanner(rf)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		var entry LogEvent
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if eventFilter != "" && entry.Event != eventFilter {
			continue
		}
		if hostnameFilter != "" && entry.Hostname != hostnameFilter {
			continue
		}
		all = append(all, entry)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	if len(all) == 0 {
		return []LogEvent{}, nil
	}

	out := make([]LogEvent, 0, minInt(limit, len(all)))
	for i := len(all) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, all[i])
	}
	return out, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
