package server

import (
	"context"
	"fmt"
	"sort"
	"sync"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/protobuf/proto"
)

// PushConfigStore tracks task push notification configs.
type PushConfigStore interface {
	Set(ctx context.Context, taskID, configID string, config *a2av1.TaskPushNotificationConfig) (*a2av1.TaskPushNotificationConfig, error)
	Get(ctx context.Context, taskID, configID string) (*a2av1.TaskPushNotificationConfig, error)
	List(ctx context.Context, taskID string, pageSize int32) ([]*a2av1.TaskPushNotificationConfig, error)
	Delete(ctx context.Context, taskID, configID string) error
}

// MemoryPushConfigStore stores push notification configs in memory.
type MemoryPushConfigStore struct {
	mu      sync.RWMutex
	configs map[string]map[string]*a2av1.TaskPushNotificationConfig
}

// NewMemoryPushConfigStore creates a new in-memory push notification store.
func NewMemoryPushConfigStore() *MemoryPushConfigStore {
	return &MemoryPushConfigStore{
		configs: make(map[string]map[string]*a2av1.TaskPushNotificationConfig),
	}
}

func (s *MemoryPushConfigStore) Set(ctx context.Context, taskID, configID string, config *a2av1.TaskPushNotificationConfig) (*a2av1.TaskPushNotificationConfig, error) {
	if taskID == "" || configID == "" {
		return nil, fmt.Errorf("task id and config id are required")
	}
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.configs[taskID]; !ok {
		s.configs[taskID] = make(map[string]*a2av1.TaskPushNotificationConfig)
	}
	s.configs[taskID][configID] = cloneTaskPushNotificationConfig(config)
	return cloneTaskPushNotificationConfig(config), nil
}

func (s *MemoryPushConfigStore) Get(ctx context.Context, taskID, configID string) (*a2av1.TaskPushNotificationConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	taskConfigs, ok := s.configs[taskID]
	if !ok {
		return nil, fmt.Errorf("config %q not found", configID)
	}
	config, ok := taskConfigs[configID]
	if !ok {
		return nil, fmt.Errorf("config %q not found", configID)
	}
	return cloneTaskPushNotificationConfig(config), nil
}

func (s *MemoryPushConfigStore) List(ctx context.Context, taskID string, pageSize int32) ([]*a2av1.TaskPushNotificationConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	taskConfigs, ok := s.configs[taskID]
	if !ok {
		return nil, nil
	}
	ids := make([]string, 0, len(taskConfigs))
	for id := range taskConfigs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	if pageSize <= 0 {
		pageSize = 50
	}
	limit := int(pageSize)
	if limit > len(ids) {
		limit = len(ids)
	}
	out := make([]*a2av1.TaskPushNotificationConfig, 0, limit)
	for _, id := range ids[:limit] {
		out = append(out, cloneTaskPushNotificationConfig(taskConfigs[id]))
	}
	return out, nil
}

func (s *MemoryPushConfigStore) Delete(ctx context.Context, taskID, configID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskConfigs, ok := s.configs[taskID]
	if !ok {
		return fmt.Errorf("config %q not found", configID)
	}
	if _, ok := taskConfigs[configID]; !ok {
		return fmt.Errorf("config %q not found", configID)
	}
	delete(taskConfigs, configID)
	if len(taskConfigs) == 0 {
		delete(s.configs, taskID)
	}
	return nil
}

func cloneTaskPushNotificationConfig(config *a2av1.TaskPushNotificationConfig) *a2av1.TaskPushNotificationConfig {
	if config == nil {
		return nil
	}
	return proto.Clone(config).(*a2av1.TaskPushNotificationConfig)
}
