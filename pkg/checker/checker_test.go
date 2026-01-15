// Copyright 2021-2026 Altessa Solutions Inc. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package checker

import (
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

	// Check with fixing
	cfg.Fix = true
	count := 0
	for _, err := range Check(cfg) {
		if err != nil {
			t.Fatalf("Check with Fix failed: %v", err)
		}
		count++ // In the new implementation fixed files are also yielded?
		// Wait, let's check `checker.go` implementation.
		// "if hit || err != nil { ... yield(path, err) }"
		// If Fix=true, checkFile returns (false, nil) IF it successfully fixed the file?
		// Or does it return (true, nil) if it *needed* fix?
		// checkFile() returns (hit, err).
		// If fix=true, checkFile calls fixFile() and returns (false, err) [line 204].
		// Line 204: `if fix { return false, fixFile(...) }`
		// So if fix is successful, it returns (false, nil).
		// So Check() will NOT yield fixed files if Fix=true and no error occurred.
		// That means count should be 0 unless there was an error.
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
