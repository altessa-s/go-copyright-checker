// Copyright 2021-2026 Altessa Solutions Inc. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case 1: Valid config file
	validConfig := `
variables:
  YEAR: "2025"
  COMPANY: "Test Corp"
template: |
  Copyright {{.YEAR}} {{.COMPANY}}
  All rights reserved.
`
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(validConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Variables["YEAR"] != "2025" {
		t.Errorf("Expected YEAR=2025, got %s", cfg.Variables["YEAR"])
	}
	if cfg.Variables["COMPANY"] != "Test Corp" {
		t.Errorf("Expected COMPANY='Test Corp', got %s", cfg.Variables["COMPANY"])
	}
	if cfg.Template == "" {
		t.Error("Template should not be empty")
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/to/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Invalid YAML content
	invalidConfig := `
variables: [this is not valid yaml
  nested: wrong
`
	configFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(configFile, []byte(invalidConfig), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = LoadConfig(configFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestLoadConfigEmptyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Empty config file
	configFile := filepath.Join(tmpDir, "empty.yaml")
	if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed on empty file: %v", err)
	}

	// Should have initialized Variables map
	if cfg.Variables == nil {
		t.Error("Variables should be initialized even for empty file")
	}
}

func TestLoadConfigTrimTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Config with whitespace in template
	configWithWhitespace := `
template: "   Copyright Test   "
`
	configFile := filepath.Join(tmpDir, "whitespace.yaml")
	if err := os.WriteFile(configFile, []byte(configWithWhitespace), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Template != "Copyright Test" {
		t.Errorf("Template should be trimmed, got '%s'", cfg.Template)
	}
}
