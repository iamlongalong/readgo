package readgo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestFiles creates test files in the given directory
func setupTestFiles(t *testing.T, dir string) {
	t.Helper()

	// Create test files structure
	files := map[string]string{
		"testdata/basic/main.go": `package basic

// User represents a user in the system
type User struct {
	ID   int
	Name string
}

// String implements fmt.Stringer
func (u *User) String() string {
	return u.Name
}

// ComplexInterface defines a complex interface for testing
type ComplexInterface interface {
	Method1() error
	Method2(s string, i int) (bool, error)
	Method3(data []byte) string
}

// Method1 implements a basic method
func Method1() error {
	return nil
}

// Method2 implements a method with multiple parameters and results
func Method2(s string, i int) (bool, error) {
	return true, nil
}

// Method3 implements a method with byte slice parameter
func Method3(data []byte) string {
	return string(data)
}

// Method4 implements a method with complex parameters
func Method4(data map[string]interface{}) error {
	return nil
}

// Method5 implements a method with variadic parameters
func Method5(prefix string, values ...interface{}) (string, error) {
	return prefix, nil
}

// Method6 implements a method with channel parameters
func Method6(input chan string, output chan<- interface{}) error {
	return nil
}`,

		"testdata/multi/file1.go": `package multi

// Service represents a service interface
type Service interface {
	Process(data []byte) error
	Close() error
}

// DefaultService implements the Service interface
type DefaultService struct {
	running bool
}

func (s *DefaultService) Process(data []byte) error {
	return nil
}

func (s *DefaultService) Close() error {
	s.running = false
	return nil
}`,

		"testdata/multi/file2.go": `package multi

import (
	"sync"
)

// Manager manages multiple services
type Manager struct {
	services []Service
	mu       sync.RWMutex
}

func (m *Manager) AddService(svc Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services = append(m.services, svc)
}

func (m *Manager) ProcessAll(data []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, svc := range m.services {
		if err := svc.Process(data); err != nil {
			return err
		}
	}
	return nil
}`,
	}

	// Create base directories
	testDataDir := filepath.Join(dir, "testdata")
	dirs := []string{
		filepath.Join(testDataDir, "basic"),
		filepath.Join(testDataDir, "multi"),
	}

	for _, d := range dirs {
		err := os.MkdirAll(d, 0750)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", d, err)
		}
	}

	// Write test files
	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		err := os.WriteFile(fullPath, []byte(content), 0600)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Create go.mod file
	setupTestModule(t, dir)
}

// setupTestModule creates a test go.mod file and initializes the module
func setupTestModule(t *testing.T, dir string) {
	t.Helper()

	content := `module testmod

go 1.21

require (
	golang.org/x/tools v0.1.1
)
`
	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Run go mod tidy to download dependencies and create go.sum
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to run go mod tidy: %v\nOutput: %s", err, out)
	}
}
