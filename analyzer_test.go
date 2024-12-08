package readgo

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func setupTestModule(t *testing.T, dir string) {
	goModContent := `module test

go 1.22.0
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}
}

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
	gomod := `module testmod

go 1.22.0

require (
	golang.org/x/tools v0.19.0
)
`
	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}
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
			path:    ".",
			wantErr: false,
		},
		{
			name:    "Non-existent project",
			path:    "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(
				WithWorkDir(tmpDir),
				WithCacheTTL(time.Minute),
			)
			result, err := analyzer.AnalyzeProject(context.Background(), tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if result.Name == "" {
					t.Error("AnalyzeProject() result name is empty")
				}
				if result.Path == "" {
					t.Error("AnalyzeProject() result path is empty")
				}
			}
		})
	}
}

func TestAnalyzeFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid file",
			path:    "testdata/basic/main.go",
			wantErr: false,
		},
		{
			name:    "Non-existent file",
			path:    "non-existent.go",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(
				WithWorkDir(tmpDir),
				WithCacheTTL(time.Minute),
			)
			result, err := analyzer.AnalyzeFile(context.Background(), tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if result.Name == "" {
					t.Error("AnalyzeFile() result name is empty")
				}
				if result.Path == "" {
					t.Error("AnalyzeFile() result path is empty")
				}
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
			if err == nil {
				if result.Name != tt.typeName {
					t.Errorf("FindType() got = %v, want %v", result.Name, tt.typeName)
				}
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
			if err == nil {
				if result.Name != tt.interfaceName {
					t.Errorf("FindInterface() got = %v, want %v", result.Name, tt.interfaceName)
				}
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
	if err != nil {
		t.Fatalf("First FindType() failed: %v", err)
	}
	firstDuration := time.Since(start)

	// Second call should hit cache
	start = time.Now()
	result2, err := analyzer.FindType(context.Background(), "./testdata/basic", "User")
	if err != nil {
		t.Fatalf("Second FindType() failed: %v", err)
	}
	secondDuration := time.Since(start)

	// Verify results are the same
	if result1.Name != result2.Name || result1.Package != result2.Package {
		t.Error("Cache returned different results")
	}

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

func BenchmarkTypeAnalysis(b *testing.B) {
	tests := []struct {
		name     string
		pkgPath  string
		typeName string
		useCache bool
	}{
		{
			name:     "Find type without cache",
			pkgPath:  "github.com/iamlongalong/readgo",
			typeName: "TypeInfo",
			useCache: false,
		},
		{
			name:     "Find type with cache",
			pkgPath:  "github.com/iamlongalong/readgo",
			typeName: "TypeInfo",
			useCache: true,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			var opts []Option
			if tt.useCache {
				opts = append(opts, WithCacheTTL(5*time.Minute))
			}
			analyzer := NewAnalyzer(opts...)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := analyzer.FindType(context.Background(), tt.pkgPath, tt.typeName)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkPackageAnalysis(b *testing.B) {
	tests := []struct {
		name     string
		pkgPath  string
		useCache bool
	}{
		{
			name:     "Analyze package without cache",
			pkgPath:  "github.com/iamlongalong/readgo",
			useCache: false,
		},
		{
			name:     "Analyze package with cache",
			pkgPath:  "github.com/iamlongalong/readgo",
			useCache: true,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			var opts []Option
			if tt.useCache {
				opts = append(opts, WithCacheTTL(5*time.Minute))
			}
			analyzer := NewAnalyzer(opts...)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := analyzer.AnalyzePackage(context.Background(), tt.pkgPath)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkFileAnalysis(b *testing.B) {
	tests := []struct {
		name     string
		filePath string
		useCache bool
	}{
		{
			name:     "Analyze file without cache",
			filePath: "testdata/basic/main.go",
			useCache: false,
		},
		{
			name:     "Analyze file with cache",
			filePath: "testdata/basic/main.go",
			useCache: true,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			var opts []Option
			if tt.useCache {
				opts = append(opts, WithCacheTTL(5*time.Minute))
			}
			analyzer := NewAnalyzer(opts...)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := analyzer.AnalyzeFile(context.Background(), tt.filePath)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkConcurrentAnalysis(b *testing.B) {
	files := []string{
		"testdata/basic/main.go",
		"testdata/multi/file1.go",
		"testdata/multi/file2.go",
	}

	tests := []struct {
		name        string
		concurrent  bool
		useCache    bool
		maxRoutines int
	}{
		{
			name:       "Sequential analysis without cache",
			concurrent: false,
			useCache:   false,
		},
		{
			name:       "Sequential analysis with cache",
			concurrent: false,
			useCache:   true,
		},
		{
			name:        "Concurrent analysis without cache",
			concurrent:  true,
			useCache:    false,
			maxRoutines: 2,
		},
		{
			name:        "Concurrent analysis with cache",
			concurrent:  true,
			useCache:    true,
			maxRoutines: 2,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			opts := []Option{
				WithConcurrentAnalysis(tt.concurrent),
			}
			if tt.useCache {
				opts = append(opts, WithCacheTTL(5*time.Minute))
			}
			if tt.maxRoutines > 0 {
				opts = append(opts, WithMaxConcurrentAnalysis(tt.maxRoutines))
			}
			analyzer := NewAnalyzer(opts...)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				for _, file := range files {
					if tt.concurrent {
						wg.Add(1)
						go func(f string) {
							defer wg.Done()
							_, err := analyzer.AnalyzeFile(context.Background(), f)
							if err != nil {
								b.Error(err)
							}
						}(file)
					} else {
						_, err := analyzer.AnalyzeFile(context.Background(), file)
						if err != nil {
							b.Fatal(err)
						}
					}
				}
				if tt.concurrent {
					wg.Wait()
				}
			}
		})
	}
}

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Clean up
	os.Exit(code)
}

// assertNoError fails the test if err is not nil
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// assertError fails the test if err is nil
func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// assertDeepEqual fails the test if a and b are not deeply equal
func assertDeepEqual(t *testing.T, a, b interface{}) {
	t.Helper()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("\nexpected: %#v\ngot: %#v", a, b)
	}
}

// assertContains fails the test if str does not contain substr
func assertContains(t *testing.T, str, substr string) {
	t.Helper()
	if !strings.Contains(str, substr) {
		t.Errorf("expected %q to contain %q", str, substr)
	}
}

// assertNotContains fails the test if str contains substr
func assertNotContains(t *testing.T, str, substr string) {
	t.Helper()
	if strings.Contains(str, substr) {
		t.Errorf("expected %q to not contain %q", str, substr)
	}
}

// assertFileExists fails the test if the file does not exist
func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("file %s does not exist", path)
	}
}

// assertDirExists fails the test if the directory does not exist
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
