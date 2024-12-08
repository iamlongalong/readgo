package readgo

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// setupTestFiles creates test files in the given directory
func setupTestFiles(t *testing.T, dir string) {
	t.Helper()

	// Create test files structure
	files := map[string]string{
		"testdata/basic/main.go": `package basic

import (
	"context"
)

// ComplexInterface defines a complex interface for testing
type ComplexInterface interface {
	Method1(ctx context.Context) error
	Method2(s string, i int) (bool, error)
	Method3(data []byte) string
}

// User represents a user in the system
type User struct {
	ID   int
	Name string
}

// String implements fmt.Stringer
func (u *User) String() string {
	return u.Name
}`,

		"testdata/multi/file1.go": `package multi

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

func (s *DefaultService) Process(ctx context.Context, data []byte) error {
	return nil
}

func (s *DefaultService) Close() error {
	s.running = false
	return nil
}`,

		"testdata/multi/file2.go": `package multi

import (
	"context"
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

func (m *Manager) ProcessAll(ctx context.Context, data []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, svc := range m.services {
		if err := svc.Process(ctx, data); err != nil {
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
		err := os.MkdirAll(d, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", d, err)
		}
	}

	// Write test files
	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Create go.mod file
	setupTestModule(t, dir)
}

func TestAnalyzeProject(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid project",
			path:    tmpDir,
			wantErr: false,
		},
		{
			name:    "Non-existent project",
			path:    filepath.Join(tmpDir, "nonexistent"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(
				WithWorkDir(tt.path),
				WithCacheTTL(time.Minute),
			)
			result, err := analyzer.AnalyzeProject(context.Background(), ".")
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assertNoError(t, err)
				if result.Name != "testmod" {
					t.Errorf("Expected project name to be 'testmod', got %q", result.Name)
				}
				assertDirExists(t, filepath.Join(tt.path, "testdata"))
			} else {
				assertError(t, err)
			}
		})
	}
}

func TestAnalyzeFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "Valid file",
			filePath: filepath.Join("testdata", "basic", "main.go"),
			wantErr:  false,
		},
		{
			name:     "Non-existent file",
			filePath: "nonexistent.go",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(
				WithWorkDir(tmpDir),
				WithCacheTTL(time.Minute),
			)
			result, err := analyzer.AnalyzeFile(context.Background(), tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assertNoError(t, err)
				assertFileExists(t, filepath.Join(tmpDir, tt.filePath))

				// Check if we found the expected types
				var foundUser, foundInterface bool
				for _, typ := range result.Types {
					switch typ.Name {
					case "User":
						foundUser = true
						if typ.Type != "struct{ID int; Name string}" {
							t.Errorf("User type has wrong structure: %s", typ.Type)
						}
					case "ComplexInterface":
						foundInterface = true
						if !strings.Contains(typ.Type, "interface") {
							t.Errorf("ComplexInterface is not an interface type: %s", typ.Type)
						}
					}
				}

				if !foundUser {
					t.Error("User type not found in analysis results")
				}
				if !foundInterface {
					t.Error("ComplexInterface not found in analysis results")
				}
			} else {
				assertError(t, err)
			}
		})
	}
}

func TestFindType(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	tests := []struct {
		name     string
		pkgPath  string
		typeName string
		wantErr  bool
	}{
		{
			name:     "Valid type",
			pkgPath:  "./testdata/basic",
			typeName: "User",
			wantErr:  false,
		},
		{
			name:     "Non-existent type",
			pkgPath:  "./testdata/basic",
			typeName: "NonExistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(
				WithWorkDir(tmpDir),
				WithCacheTTL(time.Minute),
			)
			result, err := analyzer.FindType(context.Background(), tt.pkgPath, tt.typeName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assertNoError(t, err)
				assertDeepEqual(t, result.Name, tt.typeName)
			} else {
				assertError(t, err)
			}
		})
	}
}

func TestFindInterface(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	tests := []struct {
		name          string
		pkgPath       string
		interfaceName string
		wantErr       bool
	}{
		{
			name:          "Valid interface",
			pkgPath:       "./testdata/basic",
			interfaceName: "ComplexInterface",
			wantErr:       false,
		},
		{
			name:          "Non-existent interface",
			pkgPath:       "./testdata/basic",
			interfaceName: "NonExistent",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(
				WithWorkDir(tmpDir),
				WithCacheTTL(time.Minute),
			)
			result, err := analyzer.FindInterface(context.Background(), tt.pkgPath, tt.interfaceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindInterface() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assertNoError(t, err)
				assertDeepEqual(t, result.Name, tt.interfaceName)
			} else {
				assertError(t, err)
			}
		})
	}
}

func TestCacheEffectiveness(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	analyzer := NewAnalyzer(
		WithWorkDir(tmpDir),
		WithCacheTTL(time.Minute),
	)

	// First call should miss cache
	start := time.Now()
	result1, err := analyzer.FindType(context.Background(), "./testdata/basic", "User")
	assertNoError(t, err)
	firstDuration := time.Since(start)

	// Second call should hit cache
	start = time.Now()
	result2, err := analyzer.FindType(context.Background(), "./testdata/basic", "User")
	assertNoError(t, err)
	secondDuration := time.Since(start)

	// Verify results are the same
	assertDeepEqual(t, result1, result2)

	// Second call should be significantly faster
	if secondDuration > firstDuration/2 {
		t.Errorf("Cache not effective: first=%v, second=%v", firstDuration, secondDuration)
	}

	// Verify cache statistics
	stats := analyzer.GetCacheStats()
	if stats["enabled"] != true {
		t.Error("Cache should be enabled")
	}
	if stats["type_entries"].(int) < 1 {
		t.Error("Cache should have at least one type entry")
	}
}

// setupTestModule creates a temporary Go module for testing
func setupTestModule(t *testing.T, dir string) {
	t.Helper()
	gomod := `module testmod

go 1.22.0

require (
	golang.org/x/tools v0.19.0
)
`
	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644)
	assertNoError(t, err)
}

// Test helper functions
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func assertDeepEqual(t *testing.T, a, b interface{}) {
	t.Helper()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("\nexpected: %#v\ngot: %#v", a, b)
	}
}

func assertContains(t *testing.T, str, substr string) {
	t.Helper()
	if !strings.Contains(str, substr) {
		t.Errorf("expected %q to contain %q", str, substr)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("file %s does not exist", path)
	}
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("directory %s does not exist", path)
		return
	}
	if !info.IsDir() {
		t.Errorf("%s exists but is not a directory", path)
	}
}
