package gencfg

import (
	"errors"
	"testing"

	validator "github.com/go-playground/validator/v10"
)

func TestValidateValidStruct(t *testing.T) {
	type config struct {
		Host string `validate:"required"`
		Port int    `validate:"required,min=1,max=65535"`
	}
	cfg := &config{Host: "localhost", Port: 8080}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateRequiredFieldMissing(t *testing.T) {
	type config struct {
		Host string `validate:"required"`
		Port int    `validate:"required"`
	}
	cfg := &config{Host: "", Port: 0}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		t.Fatalf("expected validator.ValidationErrors, got %T: %v", err, err)
	}

	// Both fields should fail
	fields := make(map[string]bool)
	for _, fe := range validationErrors {
		fields[fe.Field()] = true
	}
	if !fields["Host"] {
		t.Error("expected Host field to fail validation")
	}
	if !fields["Port"] {
		t.Error("expected Port field to fail validation")
	}
}

func TestValidateMinMax(t *testing.T) {
	type config struct {
		Port int `validate:"min=1,max=65535"`
	}

	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid_low", 1, false},
		{"valid_mid", 8080, false},
		{"valid_high", 65535, false},
		{"too_low", 0, true},
		{"too_high", 65536, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config{Port: tt.port}
			err := Validate(cfg)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for port=%d, got nil", tt.port)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error for port=%d, got: %v", tt.port, err)
			}
		})
	}
}

func TestValidateStringConstraints(t *testing.T) {
	type config struct {
		Name  string `validate:"required,min=3,max=20"`
		Email string `validate:"required,email"`
	}

	tests := []struct {
		name    string
		cfg     config
		wantErr bool
	}{
		{"valid", config{Name: "myapp", Email: "test@example.com"}, false},
		{"name_too_short", config{Name: "ab", Email: "test@example.com"}, true},
		{"invalid_email", config{Name: "myapp", Email: "not-an-email"}, true},
		{"empty_name", config{Name: "", Email: "test@example.com"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			err := Validate(&cfg)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateNestedStruct(t *testing.T) {
	type database struct {
		Host string `validate:"required"`
		Port int    `validate:"required,min=1"`
	}
	type config struct {
		DB database `validate:"required"`
	}

	// Valid nested struct
	cfg := &config{DB: database{Host: "db.local", Port: 5432}}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected no error for valid nested struct, got: %v", err)
	}

	// Invalid nested struct (missing required field)
	cfg = &config{DB: database{Host: "", Port: 5432}}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid nested struct, got nil")
	}

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		t.Fatalf("expected validator.ValidationErrors, got %T", err)
	}
	// The failing field should be the nested Host
	found := false
	for _, fe := range validationErrors {
		if fe.StructNamespace() == "config.DB.Host" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected DB.Host to fail validation")
	}
}

func TestValidateWithAdditionalChecks(t *testing.T) {
	type config struct {
		MinPort int `validate:"required,min=1"`
		MaxPort int `validate:"required,min=1"`
	}

	// Custom check: MaxPort must be greater than MinPort
	customCheck := func(sl validator.StructLevel) {
		cfg := sl.Current().Interface().(config)
		if cfg.MaxPort <= cfg.MinPort {
			sl.ReportError(cfg.MaxPort, "MaxPort", "MaxPort", "gtfield", "MinPort")
		}
	}

	// Valid: MaxPort > MinPort
	cfg := &config{MinPort: 1000, MaxPort: 2000}
	if err := Validate(cfg, WithAdditionalChecks(customCheck)); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Invalid: MaxPort <= MinPort
	cfg = &config{MinPort: 2000, MaxPort: 1000}
	err := Validate(cfg, WithAdditionalChecks(customCheck))
	if err == nil {
		t.Fatal("expected error for MaxPort <= MinPort, got nil")
	}

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		t.Fatalf("expected validator.ValidationErrors, got %T", err)
	}
	found := false
	for _, fe := range validationErrors {
		if fe.Field() == "MaxPort" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected MaxPort to fail custom validation")
	}

	// Equal ports should also fail
	cfg = &config{MinPort: 1000, MaxPort: 1000}
	if err := Validate(cfg, WithAdditionalChecks(customCheck)); err == nil {
		t.Error("expected error for equal ports, got nil")
	}
}

func TestValidateWithoutAdditionalChecks(t *testing.T) {
	// Ensure Validate works correctly without any options
	type config struct {
		Name string `validate:"required"`
	}

	cfg := &config{Name: "test"}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateNoTags(t *testing.T) {
	// Struct with no validate tags should always pass
	type config struct {
		Name string
		Port int
	}

	cfg := &config{}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected no error for struct with no validate tags, got: %v", err)
	}
}

func TestValidateMultipleTagFailures(t *testing.T) {
	type config struct {
		Host string `validate:"required"`
		Port int    `validate:"required,min=1"`
		Name string `validate:"required,min=3"`
	}

	// All fields invalid
	cfg := &config{}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		t.Fatalf("expected validator.ValidationErrors, got %T", err)
	}
	if len(validationErrors) < 3 {
		t.Errorf("expected at least 3 validation errors, got %d", len(validationErrors))
	}
}

func TestValidateOneofTag(t *testing.T) {
	type config struct {
		Env string `validate:"required,oneof=dev staging production"`
	}

	tests := []struct {
		name    string
		env     string
		wantErr bool
	}{
		{"valid_dev", "dev", false},
		{"valid_staging", "staging", false},
		{"valid_production", "production", false},
		{"invalid_value", "testing", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config{Env: tt.env}
			err := Validate(cfg)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for env=%q, got nil", tt.env)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error for env=%q, got: %v", tt.env, err)
			}
		})
	}
}

func TestValidateSliceField(t *testing.T) {
	type config struct {
		Tags []string `validate:"required,min=1"`
	}

	// Valid: non-empty slice
	cfg := &config{Tags: []string{"web"}}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Invalid: empty slice
	cfg = &config{Tags: []string{}}
	if err := Validate(cfg); err == nil {
		t.Error("expected error for empty slice, got nil")
	}

	// Invalid: nil slice
	cfg = &config{Tags: nil}
	if err := Validate(cfg); err == nil {
		t.Error("expected error for nil slice, got nil")
	}
}
