package gencfg

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

func loadTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("failed to read testdata/%s: %v", name, err)
	}
	return data
}

// parseYAML unmarshals YAML bytes into a map for easy comparison.
func parseYAML(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var result map[string]any
	if err := yaml.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse YAML: %v\n%s", err, string(data))
	}
	return result
}

// getNestedValue traverses a map[string]any by dot-separated key path.
func getNestedValue(t *testing.T, m map[string]any, keyPath string) any {
	t.Helper()
	keys := strings.Split(keyPath, ".")
	var current any = m
	for _, key := range keys {
		cm, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("key %q: expected map, got %T", key, current)
		}
		current, ok = cm[key]
		if !ok {
			t.Fatalf("key %q not found in map (path: %s)", key, keyPath)
		}
	}
	return current
}

func TestProcessNoTemplates(t *testing.T) {
	src := loadTestdata(t, "no_templates.yaml")
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	// When there are no templates, output should re-parse to the same structure.
	expected := parseYAML(t, src)
	actual := parseYAML(t, result)

	if v := actual["server"].(map[string]any)["host"]; v != expected["server"].(map[string]any)["host"] {
		t.Errorf("host: got %v, want %v", v, expected["server"].(map[string]any)["host"])
	}
	if v := actual["server"].(map[string]any)["port"]; v != expected["server"].(map[string]any)["port"] {
		t.Errorf("port: got %v, want %v", v, expected["server"].(map[string]any)["port"])
	}
	if v := actual["server"].(map[string]any)["debug"]; v != expected["server"].(map[string]any)["debug"] {
		t.Errorf("debug: got %v, want %v", v, expected["server"].(map[string]any)["debug"])
	}
}

func TestProcessBasicExpansion(t *testing.T) {
	src := loadTestdata(t, "basic_expansion.yaml")
	rootDir := "/test/project"
	result, err := Process(src, WithRootDir(rootDir))
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	server := parsed["server"].(map[string]any)

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("os.Hostname() error: %v", err)
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"hostname", hostname},
		{"arch", runtime.GOARCH},
		{"os", runtime.GOOS},
		{"project_dir", rootDir},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := server[tt.key].(string)
			if !ok {
				t.Fatalf("key %q: expected string, got %T (%v)", tt.key, server[tt.key], server[tt.key])
			}
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}

	// CPUs should be expanded to an integer
	cpus := server["cpus"]
	cpuInt, ok := cpus.(int)
	if !ok {
		t.Fatalf("cpus: expected int, got %T (%v)", cpus, cpus)
	}
	if cpuInt != runtime.NumCPU() {
		t.Errorf("cpus: got %d, want %d", cpuInt, runtime.NumCPU())
	}

	// Name should be the YAML field name "name"
	name, ok := server["name"].(string)
	if !ok {
		t.Fatalf("name: expected string, got %T (%v)", server["name"], server["name"])
	}
	if name != "name" {
		t.Errorf("name: got %q, want %q", name, "name")
	}
}

func TestProcessTypeCoercion(t *testing.T) {
	src := loadTestdata(t, "type_coercion.yaml")
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	types := parsed["types"].(map[string]any)

	tests := []struct {
		key      string
		expected any
	}{
		{"bool_true", true},
		{"bool_false", false},
		{"integer", 42},
		{"float_val", 3.14},
		{"string_val", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := types[tt.key]
			if got != tt.expected {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.expected, tt.expected)
			}
		})
	}
}

func TestProcessSprigFunctions(t *testing.T) {
	src := loadTestdata(t, "sprig_functions.yaml")
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	sprig := parsed["sprig"].(map[string]any)

	tests := []struct {
		key      string
		expected any
	}{
		{"upper", "HELLO"},
		{"lower", "world"},
		{"trimmed", "spaces"},
		{"default_val", "fallback"},
		{"default_with_val", "actual"},
		{"ternary_true", "yes"},
		{"ternary_false", "no"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := sprig[tt.key]
			if got != tt.expected {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.expected, tt.expected)
			}
		})
	}
}

func TestProcessWithArguments(t *testing.T) {
	src := loadTestdata(t, "arguments.yaml")
	result, err := Process(src,
		WithArgument("env", "production"),
		WithArgument("region", "us-east-1"),
	)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	config := parsed["config"].(map[string]any)

	tests := []struct {
		key      string
		expected string
	}{
		{"env", "production"},
		{"region", "us-east-1"},
		{"combined", "production-us-east-1"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := config[tt.key].(string)
			if !ok {
				t.Fatalf("key %q: expected string, got %T", tt.key, config[tt.key])
			}
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProcessDoNotExpandField(t *testing.T) {
	src := loadTestdata(t, "do_not_expand.yaml")
	result, err := Process(src, WithDoNotExpandField("literal"))
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	config := parsed["config"].(map[string]any)

	// "expanded" should be processed
	if got := config["expanded"]; got != "HELLO" {
		t.Errorf("expanded: got %v, want %q", got, "HELLO")
	}

	// "literal" should remain unexpanded
	if got := config["literal"]; got != "{{ .Hostname }}" {
		t.Errorf("literal: got %v, want %q", got, "{{ .Hostname }}")
	}

	// "also_expanded" should be processed
	if got := config["also_expanded"]; got != "world" {
		t.Errorf("also_expanded: got %v, want %q", got, "world")
	}
}

func TestProcessWithRootDir(t *testing.T) {
	src := loadTestdata(t, "basic_expansion.yaml")
	customDir := "/custom/project/dir"
	result, err := Process(src, WithRootDir(customDir))
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	projectDir := getNestedValue(t, parsed, "server.project_dir")
	if projectDir != customDir {
		t.Errorf("project_dir: got %v, want %q", projectDir, customDir)
	}
}

func TestProcessDefaultRootDir(t *testing.T) {
	// When no rootDir is provided, it should default to cwd.
	src := []byte(`config:
  dir: "{{ .ProjectDir }}"
`)
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	dir := getNestedValue(t, parsed, "config.dir")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}
	if dir != cwd {
		t.Errorf("project_dir: got %v, want %q", dir, cwd)
	}
}

func TestProcessJoinPath(t *testing.T) {
	src := loadTestdata(t, "join_path.yaml")
	rootDir := "/my/project"
	result, err := Process(src, WithRootDir(rootDir))
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	paths := parsed["paths"].(map[string]any)

	expectedSimple := filepath.Join("usr", "local", "bin")
	if got := paths["simple"]; got != expectedSimple {
		t.Errorf("simple: got %v, want %q", got, expectedSimple)
	}

	expectedWithRoot := filepath.Join(rootDir, "config", "app.yaml")
	if got := paths["with_root"]; got != expectedWithRoot {
		t.Errorf("with_root: got %v, want %q", got, expectedWithRoot)
	}
}

func TestProcessNestedTemplates(t *testing.T) {
	src := loadTestdata(t, "nested_templates.yaml")
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)

	hostname, _ := os.Hostname()

	// Deeply nested value
	deep := getNestedValue(t, parsed, "level1.level2.level3.hostname")
	if deep != hostname {
		t.Errorf("nested hostname: got %v, want %q", deep, hostname)
	}

	arch := getNestedValue(t, parsed, "level1.level2.level3.arch")
	if arch != runtime.GOARCH {
		t.Errorf("nested arch: got %v, want %q", arch, runtime.GOARCH)
	}

	sibling := getNestedValue(t, parsed, "level1.sibling")
	if sibling != runtime.GOOS {
		t.Errorf("sibling os: got %v, want %q", sibling, runtime.GOOS)
	}

	top := getNestedValue(t, parsed, "top_level")
	if top != "TOP" {
		t.Errorf("top_level: got %v, want %q", top, "TOP")
	}
}

func TestProcessMixedContent(t *testing.T) {
	src := loadTestdata(t, "mixed_content.yaml")
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	server := parsed["server"].(map[string]any)

	hostname, _ := os.Hostname()

	// Template-expanded fields
	if got := server["host"]; got != hostname {
		t.Errorf("host: got %v, want %q", got, hostname)
	}
	if got := server["name"]; got != "MYAPP" {
		t.Errorf("name: got %v, want %q", got, "MYAPP")
	}

	// Static fields should be preserved
	if got := server["port"]; got != 8080 {
		t.Errorf("port: got %v, want %d", got, 8080)
	}
	if got := server["debug"]; got != true {
		t.Errorf("debug: got %v, want %v", got, true)
	}

	// Nested metadata
	meta := server["metadata"].(map[string]any)
	if got := meta["version"]; got != "1.0" {
		t.Errorf("version: got %v, want %q", got, "1.0")
	}
	if got := meta["arch"]; got != runtime.GOARCH {
		t.Errorf("arch: got %v, want %q", got, runtime.GOARCH)
	}

	// Tags list with template expansion
	tags := server["tags"].([]any)
	if len(tags) != 2 {
		t.Fatalf("tags: expected 2 items, got %d", len(tags))
	}
	if tags[0] != "static_tag" {
		t.Errorf("tags[0]: got %v, want %q", tags[0], "static_tag")
	}
	if tags[1] != "dynamic_tag" {
		t.Errorf("tags[1]: got %v, want %q", tags[1], "dynamic_tag")
	}
}

func TestProcessFreeLocalPort(t *testing.T) {
	src := loadTestdata(t, "free_local_port.yaml")
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	services := parsed["services"].(map[string]any)

	httpPort, ok := services["http_port"].(int)
	if !ok {
		t.Fatalf("http_port: expected int, got %T (%v)", services["http_port"], services["http_port"])
	}
	grpcPort, ok := services["grpc_port"].(int)
	if !ok {
		t.Fatalf("grpc_port: expected int, got %T (%v)", services["grpc_port"], services["grpc_port"])
	}

	// Ports should be valid (1-65535)
	if httpPort < 1 || httpPort > 65535 {
		t.Errorf("http_port %d out of valid range", httpPort)
	}
	if grpcPort < 1 || grpcPort > 65535 {
		t.Errorf("grpc_port %d out of valid range", grpcPort)
	}

	// Ports should be different from each other
	if httpPort == grpcPort {
		t.Errorf("http_port and grpc_port should differ: both are %d", httpPort)
	}
}

func TestProcessEnvVar(t *testing.T) {
	src := loadTestdata(t, "env_var.yaml")

	// Set a test environment variable
	t.Setenv("GENCFG_TEST_VAR", "test_value_123")

	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	config := parsed["config"].(map[string]any)

	if got := config["test_env"]; got != "test_value_123" {
		t.Errorf("test_env: got %v, want %q", got, "test_value_123")
	}

	if got := config["with_default"]; got != "default_value" {
		t.Errorf("with_default: got %v, want %q", got, "default_value")
	}
}

func TestProcessContainerizedAndTesting(t *testing.T) {
	src := loadTestdata(t, "containerized.yaml")
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	rt := parsed["runtime"].(map[string]any)

	// When running under `go test`, Testing should be true
	if got := rt["testing"]; got != true {
		t.Errorf("testing: got %v, want %v", got, true)
	}

	// Containerized depends on the environment; just verify it is a bool
	containerized, ok := rt["containerized"].(bool)
	if !ok {
		t.Fatalf("containerized: expected bool, got %T (%v)", rt["containerized"], rt["containerized"])
	}
	_ = containerized // value depends on the runtime environment
}

func TestProcessInvalidTemplateSyntax(t *testing.T) {
	src := loadTestdata(t, "invalid_syntax.yaml")
	_, err := Process(src)
	if err == nil {
		t.Fatal("expected error for invalid template syntax, got nil")
	}
}

func TestProcessInvalidTemplateExecution(t *testing.T) {
	src := loadTestdata(t, "invalid_template.yaml")
	_, err := Process(src)
	if err == nil {
		t.Fatal("expected error for template accessing non-existent field, got nil")
	}
}

func TestProcessInvalidYAML(t *testing.T) {
	src := []byte(`{{{invalid yaml`)
	_, err := Process(src)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestProcessEmptyInput(t *testing.T) {
	// Empty YAML document
	src := []byte(``)
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error on empty input: %v", err)
	}
	// Result should be parseable (even if empty/null)
	var node yaml.Node
	if err := yaml.Unmarshal(result, &node); err != nil {
		t.Fatalf("result is not valid YAML: %v", err)
	}
}

func TestProcessMultipleOptions(t *testing.T) {
	src := []byte(`config:
  dir: "{{ .ProjectDir }}"
  env: "{{ index .Arguments \"env\" }}"
  literal: "{{ .Hostname }}"
  name: "{{ upper .Name }}"
`)
	result, err := Process(src,
		WithRootDir("/opt/app"),
		WithArgument("env", "staging"),
		WithDoNotExpandField("literal"),
	)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	config := parsed["config"].(map[string]any)

	if got := config["dir"]; got != "/opt/app" {
		t.Errorf("dir: got %v, want %q", got, "/opt/app")
	}
	if got := config["env"]; got != "staging" {
		t.Errorf("env: got %v, want %q", got, "staging")
	}
	if got := config["literal"]; got != "{{ .Hostname }}" {
		t.Errorf("literal: got %v, want %q", got, "{{ .Hostname }}")
	}
	if got := config["name"]; got != "NAME" {
		t.Errorf("name: got %v, want %q", got, "NAME")
	}
}

func TestProcessMultipleDoNotExpandFields(t *testing.T) {
	src := []byte(`config:
  a: "{{ upper \"hello\" }}"
  b: "{{ upper \"world\" }}"
  c: "{{ upper \"test\" }}"
`)
	result, err := Process(src,
		WithDoNotExpandField("a"),
		WithDoNotExpandField("c"),
	)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	config := parsed["config"].(map[string]any)

	if got := config["a"]; got != `{{ upper "hello" }}` {
		t.Errorf("a: got %v, want literal template", got)
	}
	if got := config["b"]; got != "WORLD" {
		t.Errorf("b: got %v, want %q", got, "WORLD")
	}
	if got := config["c"]; got != `{{ upper "test" }}` {
		t.Errorf("c: got %v, want literal template", got)
	}
}

func TestProcessSequenceExpansion(t *testing.T) {
	// Templates in sequence items should be expanded
	src := []byte(`items:
  - name: "{{ upper \"first\" }}"
    value: static
  - name: "{{ upper \"second\" }}"
    value: also_static
`)
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	items := parsed["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	first := items[0].(map[string]any)
	if got := first["name"]; got != "FIRST" {
		t.Errorf("items[0].name: got %v, want %q", got, "FIRST")
	}
	if got := first["value"]; got != "static" {
		t.Errorf("items[0].value: got %v, want %q", got, "static")
	}

	second := items[1].(map[string]any)
	if got := second["name"]; got != "SECOND" {
		t.Errorf("items[1].name: got %v, want %q", got, "SECOND")
	}
}

func TestProcessPreservesNonStringTypes(t *testing.T) {
	// Non-string YAML values should pass through unchanged
	src := []byte(`config:
  int_val: 42
  float_val: 3.14
  bool_val: true
  null_val: null
  list_val:
    - 1
    - 2
    - 3
`)
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	config := parsed["config"].(map[string]any)

	if got := config["int_val"]; got != 42 {
		t.Errorf("int_val: got %v (%T), want 42", got, got)
	}
	if got := config["float_val"]; got != 3.14 {
		t.Errorf("float_val: got %v (%T), want 3.14", got, got)
	}
	if got := config["bool_val"]; got != true {
		t.Errorf("bool_val: got %v (%T), want true", got, got)
	}
	if got := config["null_val"]; got != nil {
		t.Errorf("null_val: got %v (%T), want nil", got, got)
	}
	list := config["list_val"].([]any)
	if len(list) != 3 {
		t.Fatalf("list_val: expected 3 items, got %d", len(list))
	}
}

func TestProcessConditionalTemplate(t *testing.T) {
	src := []byte(`config:
  mode: '{{ if .Testing }}test{{ else }}production{{ end }}'
`)
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	mode := getNestedValue(t, parsed, "config.mode")
	if mode != "test" {
		t.Errorf("mode: got %v, want %q", mode, "test")
	}
}

func TestProcessFreeLocalPortUniqueness(t *testing.T) {
	// Multiple calls to freeLocalPort within a single Process should produce unique ports
	src := []byte(`ports:
  p1: "{{ freeLocalPort }}"
  p2: "{{ freeLocalPort }}"
  p3: "{{ freeLocalPort }}"
  p4: "{{ freeLocalPort }}"
`)
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	ports := parsed["ports"].(map[string]any)

	seen := make(map[int]string)
	for key, val := range ports {
		port, ok := val.(int)
		if !ok {
			t.Fatalf("%s: expected int, got %T (%v)", key, val, val)
		}
		if port < 1 || port > 65535 {
			t.Errorf("%s: port %d out of valid range", key, port)
		}
		if prev, exists := seen[port]; exists {
			t.Errorf("duplicate port %d found in %s and %s", port, prev, key)
		}
		seen[port] = key
	}
}

func TestProcessStringThatLooksLikeTemplateButIsNot(t *testing.T) {
	// A string with {{ }} that is a valid Go template but produces a string result
	src := []byte(`config:
  greeting: "{{ \"hello world\" }}"
`)
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	greeting := getNestedValue(t, parsed, "config.greeting")
	if greeting != "hello world" {
		t.Errorf("greeting: got %v, want %q", greeting, "hello world")
	}
}

func TestProcessWithArgumentOverwrite(t *testing.T) {
	src := []byte(`config:
  val: '{{ index .Arguments "key" }}'
`)
	// Last WithArgument for the same key should win
	result, err := Process(src,
		WithArgument("key", "first"),
		WithArgument("key", "second"),
	)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	val := getNestedValue(t, parsed, "config.val")
	if val != "second" {
		t.Errorf("val: got %v, want %q", val, "second")
	}
}

func TestProcessLargeYAML(t *testing.T) {
	// Generate a YAML with many fields to verify the walker handles scale
	var sb strings.Builder
	sb.WriteString("config:\n")
	const numFields = 100
	for i := range numFields {
		sb.WriteString("  field_" + strconv.Itoa(i) + ": '{{ upper \"val" + strconv.Itoa(i) + "\" }}'\n")
	}

	result, err := Process([]byte(sb.String()))
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	config := parsed["config"].(map[string]any)

	for i := range numFields {
		key := "field_" + strconv.Itoa(i)
		expected := "VAL" + strconv.Itoa(i)
		if got := config[key]; got != expected {
			t.Errorf("%s: got %v, want %q", key, got, expected)
		}
	}
}

func TestProcessMultipleDocumentsNotSupported(t *testing.T) {
	// YAML with multiple documents (---) - Process should handle at least the first
	src := []byte(`---
first:
  val: "{{ upper \"hello\" }}"
---
second:
  val: "{{ upper \"world\" }}"
`)
	// yaml.Unmarshal only handles the first document, so this should work
	// for at least the first document.
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	val := getNestedValue(t, parsed, "first.val")
	if val != "HELLO" {
		t.Errorf("first.val: got %v, want %q", val, "HELLO")
	}
}

func TestProcessTemplateInKey(t *testing.T) {
	// Template syntax in a YAML key should NOT be expanded
	// (only values are expanded, not keys)
	src := []byte(`config:
  "{{ upper \"key\" }}": some_value
`)
	result, err := Process(src)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	parsed := parseYAML(t, result)
	config := parsed["config"].(map[string]any)

	// The key should remain as-is (not expanded)
	if _, ok := config[`{{ upper "key" }}`]; !ok {
		// Check what keys actually exist
		for k := range config {
			t.Logf("found key: %q", k)
		}
		t.Error("expected template key to remain unexpanded")
	}
}
