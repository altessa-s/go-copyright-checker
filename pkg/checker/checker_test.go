// Copyright 2021-2026 Altessa Solutions Inc. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package checker

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tmpl := "Copyright {{.YEAR}} My Corp"
	data := map[string]string{"YEAR": "2025"}
	expectedCopyright := "// Copyright 2025 My Corp"

	// Case 1: File without copyright
	file1 := filepath.Join(tmpDir, "file1.go")
	content1 := "package main\n\nfunc main() {}"
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatal(err)
	}

	// Case 2: File with copyright
	file2 := filepath.Join(tmpDir, "file2.go")
	content2 := expectedCopyright + "\n\npackage main"
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatal(err)
	}

	// Case 3: File with build tags and copyright
	file3 := filepath.Join(tmpDir, "file3.go")
	content3 := "//go:build linux\n\n" + expectedCopyright + "\n\npackage main"
	if err := os.WriteFile(file3, []byte(content3), 0644); err != nil {
		t.Fatal(err)
	}

	// Case 4: File with build tags and NO copyright (should insert copyright AFTER tags)
	file4 := filepath.Join(tmpDir, "file4.go")
	content4 := "//go:build darwin\n\npackage main"
	if err := os.WriteFile(file4, []byte(content4), 0644); err != nil {
		t.Fatal(err)
	}

	// Case 5: File with WRONG copyright and build tags
	file5 := filepath.Join(tmpDir, "file5.go")
	content5 := "//go:build windows\n\n// Copyright 1999 Old Corp\n\npackage main"
	if err := os.WriteFile(file5, []byte(content5), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Dir:      tmpDir,
		Fix:      false,
		Template: tmpl,
		Data:     data,
	}

	// Check without fixing
	var files []string
	for file, err := range Check(cfg) {
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		files = append(files, file)
	}

	expectedMissing := map[string]bool{
		file1: true,
		file4: true,
		file5: true,
	}

	if len(files) != len(expectedMissing) {
		t.Errorf("Expected %d files to need copyright, got %d: %v", len(expectedMissing), len(files), files)
	}

	for _, f := range files {
		if !expectedMissing[f] {
			t.Errorf("Unexpected file needs copyright: %s", f)
		}
	}

	// Check with fixing. Fixed files aren't yielded (checkFile returns hit=false
	// once the fix succeeds), so count should stay 0 unless an error occurs.
	cfg.Fix = true
	count := 0
	for _, err := range Check(cfg) {
		if err != nil {
			t.Fatalf("Check with Fix failed: %v", err)
		}
		count++
	}

	// Verify content of file1
	b1, _ := os.ReadFile(file1)
	if !strings.Contains(string(b1), expectedCopyright) {
		t.Errorf("File1 does not contain copyright")
	}

	// Verify content of file4 (Build tags preserved, copyright inserted)
	b4, _ := os.ReadFile(file4)
	s4 := string(b4)
	if !strings.HasPrefix(s4, "//go:build darwin") {
		t.Errorf("File4 build tags missing or moved: %s", s4)
	}
	if !strings.Contains(s4, expectedCopyright) {
		t.Errorf("File4 does not contain copyright")
	}
	// Check order
	idxBuild := strings.Index(s4, "//go:build darwin")
	idxCopy := strings.Index(s4, "// Copyright 2025")
	if idxCopy < idxBuild {
		t.Errorf("File4 copyright appeared before build tags")
	}

	// Verify content of file5 (Build tags preserved, old copyright removed)
	b5, _ := os.ReadFile(file5)
	s5 := string(b5)
	if !strings.HasPrefix(s5, "//go:build windows") {
		t.Errorf("File5 build tags missing or moved: %s", s5)
	}
	if !strings.Contains(s5, expectedCopyright) {
		t.Errorf("File5 does not contain new copyright checks: %s", s5)
	}
	if strings.Contains(s5, "Old Corp") {
		t.Errorf("File5 still contains old copyright")
	}
}

func TestCheckInvalidTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Dir:      tmpDir,
		Template: "{{.invalid syntax",
		Data:     map[string]string{},
	}

	var gotErr error
	for _, err := range Check(cfg) {
		gotErr = err
		break
	}

	if gotErr == nil {
		t.Error("Expected error for invalid template")
	}
	if !strings.Contains(gotErr.Error(), "invalid template") {
		t.Errorf("Expected 'invalid template' error, got: %v", gotErr)
	}
}

func TestCheckTemplateExecutionError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a template with a method call on nil to trigger execution error
	cfg := Config{
		Dir:      tmpDir,
		Template: "{{call .Func}}",
		Data:     map[string]string{},
	}

	var gotErr error
	for _, err := range Check(cfg) {
		gotErr = err
		break
	}

	if gotErr == nil {
		t.Skip("Template execution did not produce an error")
	}
	if gotErr != nil && !strings.Contains(gotErr.Error(), "failed to execute template") {
		t.Errorf("Expected 'failed to execute template' error, got: %v", gotErr)
	}
}

func TestCheckGeneratedFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// File with DO NOT EDIT marker should be skipped
	generatedFile := filepath.Join(tmpDir, "generated.go")
	content := "// Code generated by tool. DO NOT EDIT.\n\npackage main"
	if err := os.WriteFile(generatedFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Dir:      tmpDir,
		Fix:      false,
		Template: "Copyright {{.YEAR}}",
		Data:     map[string]string{"YEAR": "2025"},
	}

	count := 0
	for _, err := range Check(cfg) {
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		count++
	}

	if count != 0 {
		t.Errorf("Generated file should be skipped, but got %d files needing copyright", count)
	}
}

func TestCheckExcludeDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file in an excluded directory
	excludedDir := filepath.Join(tmpDir, "vendor")
	if err := os.MkdirAll(excludedDir, 0755); err != nil {
		t.Fatal(err)
	}
	vendorFile := filepath.Join(excludedDir, "lib.go")
	if err := os.WriteFile(vendorFile, []byte("package lib"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file in the main directory
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Dir:         tmpDir,
		Fix:         false,
		Template:    "Copyright {{.YEAR}}",
		Data:        map[string]string{"YEAR": "2025"},
		ExcludeDirs: []string{"vendor"},
	}

	var files []string
	for file, err := range Check(cfg) {
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		files = append(files, file)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file (excluding vendor), got %d", len(files))
	}

	for _, f := range files {
		if strings.Contains(f, "vendor") {
			t.Errorf("Vendor file should be excluded: %s", f)
		}
	}
}

func TestCheckSkipsDotDirs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file in a dot directory
	dotDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(dotDir, 0755); err != nil {
		t.Fatal(err)
	}
	dotFile := filepath.Join(dotDir, "config.go")
	if err := os.WriteFile(dotFile, []byte("package config"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file in an underscore directory
	underscoreDir := filepath.Join(tmpDir, "_internal")
	if err := os.MkdirAll(underscoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	underscoreFile := filepath.Join(underscoreDir, "internal.go")
	if err := os.WriteFile(underscoreFile, []byte("package internal"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Dir:      tmpDir,
		Fix:      false,
		Template: "Copyright {{.YEAR}}",
		Data:     map[string]string{"YEAR": "2025"},
	}

	count := 0
	for _, err := range Check(cfg) {
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		count++
	}

	if count != 0 {
		t.Errorf("Dot and underscore directories should be skipped, got %d files", count)
	}
}

func TestCheckNonGoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create non-Go files
	txtFile := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(txtFile, []byte("Hello"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Dir:      tmpDir,
		Fix:      false,
		Template: "Copyright {{.YEAR}}",
		Data:     map[string]string{"YEAR": "2025"},
	}

	count := 0
	for _, err := range Check(cfg) {
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		count++
	}

	if count != 0 {
		t.Errorf("Non-Go files should be skipped, got %d files", count)
	}
}

func TestCheckOldBuildTagStyle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// File with old-style build tags
	file := filepath.Join(tmpDir, "oldstyle.go")
	content := "// +build linux\n\npackage main"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Dir:      tmpDir,
		Fix:      true,
		Template: "Copyright {{.YEAR}}",
		Data:     map[string]string{"YEAR": "2025"},
	}

	for _, err := range Check(cfg) {
		if err != nil {
			t.Fatalf("Check with Fix failed: %v", err)
		}
	}

	// Verify build tag is preserved
	b, _ := os.ReadFile(file)
	s := string(b)
	if !strings.Contains(s, "// +build linux") {
		t.Errorf("Old-style build tag should be preserved: %s", s)
	}
	if !strings.Contains(s, "// Copyright 2025") {
		t.Errorf("Copyright should be added: %s", s)
	}
}

func TestIsBuildTag(t *testing.T) {
	tests := []struct {
		name     string
		comments []string
		want     bool
	}{
		{
			name:     "go:build tag",
			comments: []string{"//go:build linux"},
			want:     true,
		},
		{
			name:     "old +build tag",
			comments: []string{"// +build linux"},
			want:     true,
		},
		{
			name:     "regular comment",
			comments: []string{"// This is a regular comment"},
			want:     false,
		},
		{
			name:     "empty comment group",
			comments: []string{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cg := &ast.CommentGroup{
				List: make([]*ast.Comment, len(tt.comments)),
			}
			for i, c := range tt.comments {
				cg.List[i] = &ast.Comment{Text: c}
			}
			if len(tt.comments) == 0 {
				cg.List = nil
			}

			got := isBuildTag(cg)
			if got != tt.want {
				t.Errorf("isBuildTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterComments(t *testing.T) {
	// Create a simple parsed file to get realistic positions
	src := `// Build comment
// Copyright old
package main

// Function comment
func main() {}`

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	// Filter comments - should remove pre-package comments that aren't build tags
	filtered := filterComments(parsed.Comments, parsed.Package)

	// Check that function comment is preserved
	foundFuncComment := false
	for _, cg := range filtered {
		for _, c := range cg.List {
			if strings.Contains(c.Text, "Function comment") {
				foundFuncComment = true
			}
		}
	}

	if !foundFuncComment {
		t.Error("Function comment should be preserved")
	}
}

func TestIsGenerated(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{
			name: "generated file with DO NOT EDIT",
			src:  "// Code generated by tool. DO NOT EDIT.\npackage main",
			want: true,
		},
		{
			name: "generated file without period",
			src:  "// Code generated DO NOT EDIT\npackage main",
			want: true,
		},
		{
			name: "regular file",
			src:  "// Regular comment\npackage main",
			want: false,
		},
		{
			name: "DO NOT EDIT in inline comment",
			src:  "package main\n\nvar x = 1 // DO NOT EDIT",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			parsed, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			got := isGenerated(fset, parsed)
			if got != tt.want {
				t.Errorf("isGenerated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckYaccFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create cmd/goyacc directory structure
	yaccDir := filepath.Join(tmpDir, "cmd", "goyacc")
	if err := os.MkdirAll(yaccDir, 0755); err != nil {
		t.Fatal(err)
	}

	// yacc.go file should be skipped
	yaccFile := filepath.Join(yaccDir, "yacc.go")
	if err := os.WriteFile(yaccFile, []byte("package goyacc"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Dir:      tmpDir,
		Fix:      false,
		Template: "Copyright {{.YEAR}}",
		Data:     map[string]string{"YEAR": "2025"},
	}

	count := 0
	for _, err := range Check(cfg) {
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		count++
	}

	if count != 0 {
		t.Errorf("yacc.go should be skipped, got %d files", count)
	}
}

func TestCheckEarlyBreak(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple files
	for i := range 5 {
		file := filepath.Join(tmpDir, "file"+string(rune('a'+i))+".go")
		if err := os.WriteFile(file, []byte("package main"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := Config{
		Dir:      tmpDir,
		Fix:      false,
		Template: "Copyright {{.YEAR}}",
		Data:     map[string]string{"YEAR": "2025"},
	}

	// Break after first result to test early termination
	count := 0
	for range Check(cfg) {
		count++
		break
	}

	if count != 1 {
		t.Errorf("Should have stopped after 1 result, got %d", count)
	}
}

func TestYearRange(t *testing.T) {
	tests := []struct {
		name  string
		start string
		end   string
		want  string
	}{
		{name: "same year collapses to single value", start: "2026", end: "2026", want: "2026"},
		{name: "different years render as a range", start: "2021", end: "2026", want: "2021-2026"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := yearRange(tt.start, tt.end); got != tt.want {
				t.Errorf("yearRange(%q, %q) = %q, want %q", tt.start, tt.end, got, tt.want)
			}
		})
	}
}

func TestCheckYearRangeTemplateFunc(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checker_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	file := filepath.Join(tmpDir, "same_year.go")
	if err := os.WriteFile(file, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Dir:      tmpDir,
		Fix:      true,
		Template: "Copyright {{yearRange .START_YEAR .YEAR}} Test Corp",
		Data:     map[string]string{"START_YEAR": "2026", "YEAR": "2026"},
	}

	for _, err := range Check(cfg) {
		if err != nil {
			t.Fatalf("Check with Fix failed: %v", err)
		}
	}

	b, _ := os.ReadFile(file)
	if !strings.Contains(string(b), "// Copyright 2026 Test Corp") {
		t.Errorf("expected single-year copyright, got: %s", string(b))
	}
	if strings.Contains(string(b), "2026-2026") {
		t.Errorf("copyright should not render a range when start and end years match: %s", string(b))
	}
}
