package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type ProvisionLogger struct {
	mu sync.Mutex
	f  *os.File
}

func NewProvisionLogger(path string) (*ProvisionLogger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &ProvisionLogger{f: f}, nil
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
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf(
		"timestamp=%s event=%s hostname=%s static_ip=%s template=%s box_type=%s dns=%s\n",
		time.Now().Format(time.RFC3339),
		event,
		cfg.Hostname,
		cfg.StaticIP,
		cfg.TemplateName,
		cfg.BoxTypeName,
		strings.Join(cfg.DNS, ","),
	)
	_, _ = l.f.WriteString(line)
}
