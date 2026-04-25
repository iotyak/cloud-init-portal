package main

import (
	"errors"
	"fmt"
	"time"
)

const (
	QueueStatusPending   = "pending"
	QueueStatusActive    = "active"
	QueueStatusCompleted = "completed"
	QueueStatusCancelled = "cancelled"
	QueueStatusFailed    = "failed"
)

type QueueItem struct {
	ID           string     `json:"id"`
	Hostname     string     `json:"hostname"`
	TemplateName string     `json:"template_name"`
	BoxTypeName  string     `json:"box_type_name"`
	StaticIP     string     `json:"static_ip"`
	CIDR         string     `json:"cidr"`
	Gateway      string     `json:"gateway"`
	DNS          []string   `json:"dns"`
	CreatedAt    time.Time  `json:"created_at"`
	InstanceID   string     `json:"instance_id"`
	UserData     string     `json:"user_data"`
	MetaData     string     `json:"meta_data"`
	EnqueuedAt   time.Time  `json:"enqueued_at"`
	ActivatedAt  *time.Time `json:"activated_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Status       string     `json:"status"`
	Reason       string     `json:"reason,omitempty"`
}

type QueueState struct {
	Active    *QueueItem `json:"active"`
	Pending   []QueueItem `json:"pending"`
	Completed []QueueItem `json:"completed"`
	Failed    []QueueItem `json:"failed"`
}

type QueueSnapshot struct {
	Active    *QueueItem
	Pending   []QueueItem
	Completed []QueueItem
	Failed    []QueueItem
}

func newQueueState() QueueState {
	return QueueState{
		Pending:   []QueueItem{},
		Completed: []QueueItem{},
		Failed:    []QueueItem{},
	}
}

func queueItemFromConfig(id string, cfg ActiveConfig, status string, enqueuedAt time.Time) QueueItem {
	item := QueueItem{
		ID:           id,
		Hostname:     cfg.Hostname,
		TemplateName: cfg.TemplateName,
		BoxTypeName:  cfg.BoxTypeName,
		StaticIP:     cfg.StaticIP,
		CIDR:         cfg.CIDR,
		Gateway:      cfg.Gateway,
		DNS:          append([]string(nil), cfg.DNS...),
		CreatedAt:    cfg.CreatedAt,
		InstanceID:   cfg.InstanceID,
		UserData:     cfg.UserData,
		MetaData:     cfg.MetaData,
		EnqueuedAt:   enqueuedAt,
		Status:       status,
	}
	if status == QueueStatusActive {
		now := time.Now()
		item.ActivatedAt = &now
	}
	return item
}

func (q QueueItem) toActiveConfig() *ActiveConfig {
	cfg := &ActiveConfig{
		Hostname:     q.Hostname,
		TemplateName: q.TemplateName,
		BoxTypeName:  q.BoxTypeName,
		StaticIP:     q.StaticIP,
		CIDR:         q.CIDR,
		Gateway:      q.Gateway,
		DNS:          append([]string(nil), q.DNS...),
		CreatedAt:    q.CreatedAt,
		InstanceID:   q.InstanceID,
		UserData:     q.UserData,
		MetaData:     q.MetaData,
	}
	return cfg
}

func (s *Store) QueueSnapshot() QueueSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap := QueueSnapshot{
		Pending:   append([]QueueItem(nil), s.queue.Pending...),
		Completed: append([]QueueItem(nil), s.queue.Completed...),
		Failed:    append([]QueueItem(nil), s.queue.Failed...),
	}
	if s.queue.Active != nil {
		activeCopy := *s.queue.Active
		snap.Active = &activeCopy
	}
	return snap
}

func (s *Store) CancelQueueItem(id string) (*QueueItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == "" {
		return nil, errors.New("queue id required")
	}
	if s.queue.Active != nil && s.queue.Active.ID == id {
		return nil, errors.New("cannot cancel active queue item")
	}

	for i := range s.queue.Pending {
		if s.queue.Pending[i].ID != id {
			continue
		}
		item := s.queue.Pending[i]
		now := time.Now()
		item.Status = QueueStatusCancelled
		item.Reason = QueueStatusCancelled
		item.FinishedAt = &now
		s.queue.Pending = append(s.queue.Pending[:i], s.queue.Pending[i+1:]...)
		s.queue.Failed = append(s.queue.Failed, item)
		s.persistLocked("queue_cancel")
		return &item, nil
	}

	return nil, fmt.Errorf("queue item %q not found", id)
}

func (s *Store) Enqueue(cfg *ActiveConfig) (QueueItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cfg == nil {
		return QueueItem{}, errors.New("nil config")
	}
	if _, exists := s.consumedHostnames[cfg.Hostname]; exists {
		return QueueItem{}, fmt.Errorf("hostname %q already consumed in this process; choose a new hostname", cfg.Hostname)
	}
	if s.queueContainsHostnameLocked(cfg.Hostname) {
		return QueueItem{}, fmt.Errorf("hostname %q already exists in active/pending queue", cfg.Hostname)
	}

	cfgCopy := *cfg
	var item QueueItem
	if s.current == nil {
		item = s.setActiveLocked(cfgCopy)
		s.persistLocked("set_current")
	} else {
		item = s.enqueuePendingLocked(cfgCopy)
		s.persistLocked("enqueue_pending")
	}
	return item, nil
}

func (s *Store) ListQueue() QueueSnapshot {
	return s.QueueSnapshot()
}

func (s *Store) CompleteActive(reason string) (*QueueItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return nil, errors.New("no active config")
	}
	status := QueueStatusCompleted
	if reason == "force_replaced" {
		status = QueueStatusFailed
	} else {
		s.consumedHostnames[s.current.Hostname] = time.Now()
	}
	finished := s.finishActiveLocked(status, reason)
	if !s.promoteNextLocked() {
		if status == QueueStatusCompleted {
			if finished != nil {
				s.status = statusFromConfigValue(finished.toActiveConfigValue(), StatusConsumed, false)
			}
		} else {
			s.status = ProvisionStatus{Status: StatusNoActive, Active: false}
		}
	}
	s.persistLocked("complete_active")
	if finished == nil {
		return nil, errors.New("active queue item missing")
	}
	return finished, nil
}

func (q QueueItem) toActiveConfigValue() ActiveConfig {
	return ActiveConfig{
		Hostname:     q.Hostname,
		TemplateName: q.TemplateName,
		BoxTypeName:  q.BoxTypeName,
		StaticIP:     q.StaticIP,
		CIDR:         q.CIDR,
		Gateway:      q.Gateway,
		DNS:          append([]string(nil), q.DNS...),
		CreatedAt:    q.CreatedAt,
		InstanceID:   q.InstanceID,
		UserData:     q.UserData,
		MetaData:     q.MetaData,
	}
}

func (s *Store) PromoteNext() (*QueueItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current != nil {
		return nil, errors.New("active config already exists")
	}
	if !s.promoteNextLocked() {
		return nil, errors.New("no pending queue item")
	}
	s.persistLocked("queue_promote_next")
	if s.queue.Active == nil {
		return nil, errors.New("promotion failed")
	}
	activeCopy := *s.queue.Active
	return &activeCopy, nil
}

func (s *Store) nextQueueIDLocked() string {
	s.nextQueueID++
	return fmt.Sprintf("q-%06d", s.nextQueueID)
}

func (s *Store) queueContainsHostnameLocked(hostname string) bool {
	if s.queue.Active != nil && s.queue.Active.Hostname == hostname {
		return true
	}
	for _, item := range s.queue.Pending {
		if item.Hostname == hostname {
			return true
		}
	}
	return false
}

func (s *Store) enqueuePendingLocked(cfg ActiveConfig) QueueItem {
	item := queueItemFromConfig(s.nextQueueIDLocked(), cfg, QueueStatusPending, time.Now())
	s.queue.Pending = append(s.queue.Pending, item)
	return item
}

func (s *Store) setActiveLocked(cfg ActiveConfig) QueueItem {
	item := queueItemFromConfig(s.nextQueueIDLocked(), cfg, QueueStatusActive, time.Now())
	s.queue.Active = &item
	s.current = item.toActiveConfig()
	s.status = statusFromConfigLocked(s.current, StatusReady, true)
	return item
}

func (s *Store) promoteNextLocked() bool {
	if len(s.queue.Pending) == 0 {
		s.queue.Active = nil
		s.current = nil
		return false
	}
	next := s.queue.Pending[0]
	s.queue.Pending = s.queue.Pending[1:]
	now := time.Now()
	next.Status = QueueStatusActive
	next.ActivatedAt = &now
	next.FinishedAt = nil
	next.Reason = ""
	s.queue.Active = &next
	s.current = next.toActiveConfig()
	s.status = statusFromConfigLocked(s.current, StatusReady, true)
	return true
}

func (s *Store) finishActiveLocked(status string, reason string) *QueueItem {
	if s.queue.Active == nil {
		return nil
	}
	item := *s.queue.Active
	now := time.Now()
	item.Status = status
	item.Reason = reason
	item.FinishedAt = &now
	s.queue.Active = nil

	s.current = nil
	if status == QueueStatusCompleted {
		s.queue.Completed = append(s.queue.Completed, item)
	} else {
		s.queue.Failed = append(s.queue.Failed, item)
	}
	return &item
}
