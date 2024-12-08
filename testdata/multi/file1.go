package multi

import (
	"context"
)

// Service represents a service interface
type Service interface {
	Process(ctx context.Context, data []byte) error
	Close() error
}

// DefaultService implements the Service interface
type DefaultService struct {
	running bool
}

// Process implements Service.Process
func (s *DefaultService) Process(ctx context.Context, data []byte) error {
	return nil
}

// Close implements Service.Close
func (s *DefaultService) Close() error {
	s.running = false
	return nil
}

// NewService creates a new service instance
func NewService() Service {
	return &DefaultService{running: true}
}
