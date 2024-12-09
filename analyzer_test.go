package readgo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
			typeName: "NonExistentType",
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
				if result.Name != tt.typeName {
					t.Errorf("FindType() got type name %q, want %q", result.Name, tt.typeName)
				}
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
			interfaceName: "NonExistentInterface",
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
				if result.Name != tt.interfaceName {
					t.Errorf("FindInterface() got interface name %q, want %q", result.Name, tt.interfaceName)
				}
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

	// First call should take longer (no cache)
	start := time.Now()
	_, err := analyzer.AnalyzeProject(context.Background(), ".")
	if err != nil {
		t.Fatalf("First AnalyzeProject() failed: %v", err)
	}
	firstDuration := time.Since(start)

	// Second call should be faster (cached)
	start = time.Now()
	_, err = analyzer.AnalyzeProject(context.Background(), ".")
	if err != nil {
		t.Fatalf("Second AnalyzeProject() failed: %v", err)
	}
	secondDuration := time.Since(start)

	if secondDuration >= firstDuration {
		t.Errorf("Cache not effective: second call (%v) not faster than first call (%v)", secondDuration, firstDuration)
	}
}

// Helper functions for assertions
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("Directory %s does not exist: %v", path, err)
		return
	}
	if !info.IsDir() {
		t.Errorf("Path %s exists but is not a directory", path)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("File %s does not exist: %v", path, err)
		return
	}
	if info.IsDir() {
		t.Errorf("Path %s exists but is a directory", path)
	}
}
