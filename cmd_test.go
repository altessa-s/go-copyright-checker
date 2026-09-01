// Copyright 2021-2026 Altessa Solutions Inc. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: nil,
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: nil,
		},
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all duplicates",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uniqueStrings(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("uniqueStrings() got %v, want %v", result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("uniqueStrings() got %v, want %v", result, tt.expected)
					return
				}
			}
		})
	}
}

// newTestCheckCmd creates a fresh check command for testing
func newTestCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "check",
		Short:        "Check for missing copyright headers in Go files.",
		RunE:         run,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
	}
	cmd.Flags().StringP("config", "c", ".copyright.yaml", "path to the configuration file")
	cmd.Flags().StringP("exclude", "e", "", "comma-separated list of directories to exclude from the check")
	cmd.Flags().BoolP("fix", "f", false, "fix missing headers in Go files")
	return cmd
}

func TestCmdCheckRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cmd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Go file with copyright
	fileWithCopyright := filepath.Join(tmpDir, "good.go")
	content := fmt.Sprintf("// Copyright %d Altessa Solutions Inc. All rights reserved.\n", time.Now().Year()) +
		"// Use of this source code is governed by license that can be found in\n" +
		"// the LICENSE file.\n\n" +
		"package main"
	if err := os.WriteFile(fileWithCopyright, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCheckCmd()
	cmd.SetArgs([]string{tmpDir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("cmd.Execute() error = %v, want nil", err)
	}
}

func TestCmdCheckRunWithMissingCopyright(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cmd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Go file without copyright
	fileWithoutCopyright := filepath.Join(tmpDir, "bad.go")
	if err := os.WriteFile(fileWithoutCopyright, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCheckCmd()
	cmd.SetArgs([]string{tmpDir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	if err == nil {
		t.Error("cmd.Execute() should return an error for missing copyright")
	}
}

func TestCmdCheckRunWithFix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cmd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Go file without copyright
	file := filepath.Join(tmpDir, "tofix.go")
	if err := os.WriteFile(file, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCheckCmd()
	cmd.SetArgs([]string{"--fix", tmpDir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("cmd.Execute() with --fix error = %v", err)
	}

	// Verify the file now has copyright
	content, _ := os.ReadFile(file)
	if !strings.HasPrefix(string(content), "//") {
		t.Errorf("File should have copyright after fix, got: %s", string(content)[:50])
	}
}

func TestCmdCheckRunWithExclude(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cmd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an excluded directory with a file without copyright
	excludeDir := filepath.Join(tmpDir, "excluded")
	if err := os.MkdirAll(excludeDir, 0755); err != nil {
		t.Fatal(err)
	}
	excludedFile := filepath.Join(excludeDir, "excluded.go")
	if err := os.WriteFile(excludedFile, []byte("package excluded"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a good file in the main directory
	goodFile := filepath.Join(tmpDir, "good.go")
	content := fmt.Sprintf("// Copyright %d Altessa Solutions Inc. All rights reserved.\n", time.Now().Year()) +
		"// Use of this source code is governed by license that can be found in\n" +
		"// the LICENSE file.\n\n" +
		"package main"
	if err := os.WriteFile(goodFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCheckCmd()
	cmd.SetArgs([]string{"--exclude", "excluded", tmpDir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("cmd.Execute() with --exclude error = %v", err)
	}
}

func TestCmdCheckRunWithConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cmd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config file
	configContent := `
variables:
  COMPANY: "Test Company"
template: |
  Copyright {{.COMPANY}}
`
	configFile := filepath.Join(tmpDir, ".copyright.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file with matching copyright
	file := filepath.Join(tmpDir, "configured.go")
	content := "// Copyright Test Company\n\npackage main"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCheckCmd()
	cmd.SetArgs([]string{tmpDir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("cmd.Execute() with config error = %v", err)
	}
}

func TestCmdCheckRunWithConfigVariablesOnly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cmd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Config overrides only START_YEAR, no template: the default template
	// (and its year-range collapsing) must still apply.
	configContent := `
variables:
  START_YEAR: "2021"
`
	configFile := filepath.Join(tmpDir, ".copyright.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(tmpDir, "tofix.go")
	if err := os.WriteFile(file, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCheckCmd()
	cmd.SetArgs([]string{"--fix", tmpDir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute() with variables-only config error = %v", err)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf("// Copyright 2021-%d Altessa Solutions Inc. All rights reserved.", time.Now().Year())
	if !strings.Contains(string(content), want) {
		t.Errorf("expected default template with START_YEAR override, got: %s", string(content))
	}
}

func TestCmdCheckRunWithInvalidConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cmd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create invalid config file
	configFile := filepath.Join(tmpDir, ".copyright.yaml")
	if err := os.WriteFile(configFile, []byte("invalid: [yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a Go file
	file := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(file, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCheckCmd()
	cmd.SetArgs([]string{tmpDir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	if err == nil {
		t.Error("cmd.Execute() should return an error for invalid config")
	}
}

func TestCmdVersionRun(t *testing.T) {
	// Just execute the version command - it should not panic
	cmdVersionRun(nil, nil)
}
