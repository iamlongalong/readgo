package multi

import (
	"context"
	"sync"
)

// Manager manages multiple services
type Manager struct {
	services []Service
	mu       sync.RWMutex
}

// NewManager creates a new manager instance
func NewManager() *Manager {
	return &Manager{
		services: make([]Service, 0),
	}
}

// AddService adds a service to the manager
func (m *Manager) AddService(svc Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services = append(m.services, svc)
}

// ProcessAll processes data through all services
func (m *Manager) ProcessAll(ctx context.Context, data []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, svc := range m.services {
		if err := svc.Process(ctx, data); err != nil {
			return err
		}
	}
	return nil
}

// CloseAll closes all services
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, svc := range m.services {
		if err := svc.Close(); err != nil {
			return err
		}
	}
	return nil
}
