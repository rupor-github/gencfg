package gencfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// NOTE: not save for multiple concurrent tests!
var count int

func (sanitizeTestFunctions) CountTestCall(name, data string) error {
	fmt.Println("CountTestCall with", name, data)
	count++
	return nil
}

type Config1 struct {
	Field1 string `yaml:"field1" sanitize:"test_call=CountTestCall"`
	Field2 string `yaml:"field2" sanitize:"test_call=CountTestCall"`
}

type Config2 struct {
	Dns []string `yaml:"dns" sanitize:"test_call=CountTestCall"`
}

type Config3 struct {
	Inside Config1 `yaml:"inside"`
}

type configAll struct {
	Field0          string              `yaml:"field0" sanitize:"test_call=CountTestCall"`
	Field00         *Config1            `yaml:"field00"`
	Uninitialized   *Config1            `yaml:"uninitialized"`
	SliceOfConfigs  []Config1           `yaml:"sliceOfConfigs"`
	SliceOfConfigs1 []*Config1          `yaml:"sliceOfConfigs1"`
	ArrayOfConfigs  [2]Config1          `yaml:"arrayOfConfigs"`
	ArrayOfConfigs1 [2]*Config1         `yaml:"arrayOfConfigs1"`
	MapOfConfigs    map[string]Config1  `yaml:"mapOfConfigs"`
	MapOfConfigs1   map[string]*Config1 `yaml:"mapOfConfigs1"`
	SliceOfStrings  []string            `yaml:"sliceOfStrings" sanitize:"test_call=CountTestCall"`
	SliceOfAny      []Config2           `yaml:"sliceOfAny"`
	StuctInStruct   Config3             `yaml:"stuctInStruct"`
	unsupported     *Config1            `yaml:"unsupported"`
}

var (
	c1 = Config1{"p11", "p12"}
	c2 = Config1{"p21", "p22"}
	c3 = Config1{"parr31", "parr32"}
	c4 = Config1{"parr41", "parr42"}
	c5 = Config1{"pmap51", "pmap52"}
	c6 = Config1{"pmap61", "pmap62"}
)

var testData = configAll{
	Field0: "data0",
	Field00: &Config1{
		Field1: "data01",
		Field2: "data02",
	},
	SliceOfConfigs: []Config1{
		{
			Field1: "data1",
			Field2: "data2",
		},
		{
			Field1: "data3",
			Field2: "data4",
		},
	},
	SliceOfConfigs1: []*Config1{&c1, &c2},
	ArrayOfConfigs: [2]Config1{
		{
			Field1: "data_arr1",
			Field2: "data_arr2",
		},
		{
			Field1: "data_arr3",
			Field2: "data_arr4",
		},
	},
	ArrayOfConfigs1: [2]*Config1{&c3, &c4},
	MapOfConfigs: map[string]Config1{
		"key1": {
			Field1: "data_map1",
			Field2: "data_map2",
		},
		"key2": {
			Field1: "data_map1",
			Field2: "data_map2",
		},
	},
	MapOfConfigs1:  map[string]*Config1{"key3": &c5, "key4": &c6},
	SliceOfStrings: []string{"s1", "s2"},
	SliceOfAny:     []Config2{{Dns: []string{"dns1", "dns2"}}, {Dns: []string{"dns3", "dns4"}}},
	StuctInStruct: Config3{
		Inside: Config1{"data_inside1", "data_inside2"},
	},
	unsupported: &c1,
}

func TestSanitize(t *testing.T) {
	t.Log("TestSanitize")
	const expected = 35
	if err := Sanitize(&testData); err != nil {
		t.Fatal(err)
	}
	if count != expected {
		t.Fatalf("count=%d, expected=%d", count, expected)
	}
	t.Logf("count=%d", count)
}

func TestOneOfOrTag(t *testing.T) {
	type testConfig struct {
		Value string `yaml:"value" sanitize:"oneof_or_tag=option1 option2 option3 path_clean"`
	}

	tests := []struct {
		name        string
		input       string
		expected    string
		shouldError bool
	}{
		{
			name:        "value in oneof list",
			input:       "option1",
			expected:    "option1",
			shouldError: false,
		},
		{
			name:        "another value in oneof list",
			input:       "option2",
			expected:    "option2",
			shouldError: false,
		},
		{
			name:        "value not in list triggers tag operation",
			input:       "/some//path/../to/clean",
			expected:    "/some/to/clean",
			shouldError: false,
		},
		{
			name:        "empty value triggers tag operation",
			input:       "",
			expected:    "",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &testConfig{Value: tt.input}
			err := Sanitize(cfg)
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if cfg.Value != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, cfg.Value)
				}
			}
		})
	}
}

func TestPathClean(t *testing.T) {
	type testConfig struct {
		Path string `yaml:"path" sanitize:"path_clean"`
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "double slashes",
			input:    "/some//path",
			expected: "/some/path",
		},
		{
			name:     "parent directory references",
			input:    "/some/path/../to/file",
			expected: "/some/to/file",
		},
		{
			name:     "current directory references",
			input:    "/some/./path/./to/file",
			expected: "/some/path/to/file",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &testConfig{Path: tt.input}
			if err := Sanitize(cfg); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if cfg.Path != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, cfg.Path)
			}
		})
	}
}

func TestPathAbs(t *testing.T) {
	type testConfig struct {
		Path string `yaml:"path" sanitize:"path_abs"`
	}

	tests := []struct {
		name         string
		input        string
		expectPrefix string
		skipEmpty    bool
	}{
		{
			name:         "relative path",
			input:        "test/path",
			expectPrefix: "/",
			skipEmpty:    false,
		},
		{
			name:         "already absolute",
			input:        "/absolute/path",
			expectPrefix: "/",
			skipEmpty:    false,
		},
		{
			name:         "empty string",
			input:        "",
			expectPrefix: "",
			skipEmpty:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &testConfig{Path: tt.input}
			if err := Sanitize(cfg); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.skipEmpty && cfg.Path == "" {
				return
			}
			if !strings.HasPrefix(cfg.Path, tt.expectPrefix) {
				t.Errorf("expected path to start with %q, got %q", tt.expectPrefix, cfg.Path)
			}
		})
	}
}

func TestPathToSlash(t *testing.T) {
	type testConfig struct {
		Path string `yaml:"path" sanitize:"path_toslash"`
	}

	// Test with path constructed using OS-specific separator
	nativePath := filepath.Join("some", "path", "to", "file")
	expectedPath := "some/path/to/file"

	cfg := &testConfig{Path: nativePath}
	if err := Sanitize(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg.Path != expectedPath {
		t.Errorf("expected %q, got %q", expectedPath, cfg.Path)
	}

	// Test already forward slashes
	cfg = &testConfig{Path: "/usr/local/bin"}
	if err := Sanitize(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg.Path != "/usr/local/bin" {
		t.Errorf("expected %q, got %q", "/usr/local/bin", cfg.Path)
	}

	// Test empty string
	cfg = &testConfig{Path: ""}
	if err := Sanitize(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg.Path != "" {
		t.Errorf("expected empty string, got %q", cfg.Path)
	}
}

func TestAssureDirExists(t *testing.T) {
	type testConfig struct {
		Dir string `yaml:"dir" sanitize:"assure_dir_exists"`
	}

	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test", "nested", "dir")

	cfg := &testConfig{Dir: testDir}
	if err := Sanitize(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Errorf("directory was not created: %s", testDir)
	}

	// Test empty string
	cfg = &testConfig{Dir: ""}
	if err := Sanitize(cfg); err != nil {
		t.Errorf("unexpected error with empty string: %v", err)
	}
}

func TestAssureDirExistsForFile(t *testing.T) {
	type testConfig struct {
		File string `yaml:"file" sanitize:"assure_dir_exists_for_file"`
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test", "nested", "file.txt")

	cfg := &testConfig{File: testFile}
	if err := Sanitize(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parentDir := filepath.Dir(testFile)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Errorf("parent directory was not created: %s", parentDir)
	}

	// Test empty string
	cfg = &testConfig{File: ""}
	if err := Sanitize(cfg); err != nil {
		t.Errorf("unexpected error with empty string: %v", err)
	}
}

func TestAssureFileAccess(t *testing.T) {
	type testConfig struct {
		File string `yaml:"file" sanitize:"assure_file_access"`
	}

	// Test existing file
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	cfg := &testConfig{File: tmpFile.Name()}
	if err := Sanitize(cfg); err != nil {
		t.Errorf("unexpected error with existing file: %v", err)
	}

	// Test non-existing file
	cfg = &testConfig{File: "/non/existing/file.txt"}
	if err := Sanitize(cfg); err == nil {
		t.Errorf("expected error for non-existing file, got none")
	}

	// Test empty string
	cfg = &testConfig{File: ""}
	if err := Sanitize(cfg); err != nil {
		t.Errorf("unexpected error with empty string: %v", err)
	}
}
