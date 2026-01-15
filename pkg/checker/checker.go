// Copyright 2021-2026 Altessa Solutions Inc. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package checker

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"text/template"
)

const (
	defaultFilePerm = 0o644
)

// Config holds the configuration for the copyright checker.
type Config struct {
	Dir         string
	Fix         bool
	ExcludeDirs []string
	Template    string
	Data        map[string]string
}

// Check checks and optionally fixes copyright headers in the specified directory.
// It returns an iterator that yields files that needed changes (or errors).
// The iterator yields (filename, error). If error is nil, the file was identified as needing a fix
// (and fixed if Config.Fix is true).
func Check(cfg Config) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		// 1. Prepare configuration
		excludeDirs := make(map[string]struct{}, len(cfg.ExcludeDirs))
		for _, dir := range cfg.ExcludeDirs {
			excludeDirs[dir] = struct{}{}
		}

		// Render the template once
		tpl, err := template.New("copyright").Parse(cfg.Template)
		if err != nil {
			yield("", fmt.Errorf("invalid template: %w", err))
			return
		}

		var buff bytes.Buffer
		if err = tpl.Execute(&buff, cfg.Data); err != nil {
			yield("", fmt.Errorf("failed to execute template: %w", err))
			return
		}
		copyright := strings.TrimSpace(buff.String())

		// 2. Concurrency Setup
		// We use a worker pool pattern.
		// - walker: walks the directory and sends paths to 'jobs'
		// - workers: read 'jobs', process files, send results to 'results'
		// - main loop: reads 'results' and calls yield

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		jobs := make(chan string)
		type result struct {
			path string
			err  error
			hit  bool // true if the file needed changes/was flagged
		}
		results := make(chan result)

		// Start Walker
		go func() {
			defer close(jobs)
			//nolint:errcheck // Inner errors are propagated via the results channel
			_ = filepath.WalkDir(cfg.Dir, func(path string, d fs.DirEntry, err error) error {
				if ctx.Err() != nil {
					return filepath.SkipAll
				}
				if err != nil {
					// Send walk errors to results, but keep walking if possible?
					// Usually walk errors are fatal for that branch.
					select {
					case results <- result{path: path, err: err}:
					case <-ctx.Done():
						return filepath.SkipAll
					}
					return nil
				}

				if d.IsDir() {
					if _, ok := excludeDirs[d.Name()]; ok {
						return filepath.SkipDir
					}
					if (strings.HasPrefix(d.Name(), ".") && d.Name() != ".") || strings.HasPrefix(d.Name(), "_") {
						return filepath.SkipDir
					}
					return nil
				}

				// Send job
				select {
				case jobs <- path:
				case <-ctx.Done():
					return filepath.SkipAll
				}
				return nil
			})
		}()

		// Start Workers
		var wg sync.WaitGroup
		numWorkers := runtime.GOMAXPROCS(0)
		wg.Add(numWorkers)

		for i := 0; i < numWorkers; i++ {
			go func() {
				defer wg.Done()
				for path := range jobs {
					// Check if we should stop
					if ctx.Err() != nil {
						return
					}

					hit, err := checkFile(cfg.Dir, path, cfg.Fix, copyright)
					if hit || err != nil {
						select {
						case results <- result{path: path, err: err, hit: hit}:
						case <-ctx.Done():
							return
						}
					}
				}
			}()
		}

		// Closer
		go func() {
			wg.Wait()
			close(results)
		}()

		// 3. Consume Results
		for res := range results {
			if !yield(res.path, res.err) {
				cancel() // Stop producers
				return
			}
		}
	}
}

func checkFile(baseDir, filename string, fix bool, copyright string) (bool, error) {
	// Only check Go files.
	if !strings.HasSuffix(filename, ".go") {
		return false, nil
	}

	// Don't check testdata files.
	normalized := strings.TrimPrefix(filepath.ToSlash(filename), filepath.ToSlash(baseDir))

	// goyacc is the only file with a different copyright header.
	if strings.HasSuffix(normalized, "cmd/goyacc/yacc.go") {
		return false, nil
	}

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return false, err
	}

	// Don't require headers on generated files.
	if isGenerated(fset, parsed) {
		return false, nil
	}

	copyrightComps := strings.Split(copyright, "\n")
	for i := range copyrightComps {
		copyrightComps[i] = strings.TrimSpace(copyrightComps[i])
	}
	copyrightTrimmed := strings.Join(copyrightComps, "\n")

	shouldAddCopyright := true

	// Check existing comments
	for _, c := range parsed.Comments {
		// The copyright should appear before the package declaration.
		if c.Pos() > parsed.Package {
			break
		}

		comment := strings.TrimSpace(c.Text())
		if strings.EqualFold(comment, copyrightTrimmed) {
			shouldAddCopyright = false
			break
		}
	}

	if !shouldAddCopyright {
		return false, nil
	}

	if fix {
		return false, fixFile(fset, parsed, filename, copyrightComps)
	}

	return true, nil
}

func fixFile(fset *token.FileSet, parsed *ast.File, filename string, copyrightLines []string) error {
	// Separate build tags from other comments
	var buildTags []*ast.CommentGroup
	var remainingComments []*ast.CommentGroup

	for _, cg := range parsed.Comments {
		if isBuildTag(cg) {
			buildTags = append(buildTags, cg)
		} else {
			remainingComments = append(remainingComments, cg)
		}
	}

	// Filter out old copyright from the remaining comments
	parsed.Comments = filterComments(remainingComments, parsed.Package)

	// Now construct the file
	var generatedCode strings.Builder

	// 1. Write Build Tags
	if len(buildTags) > 0 {
		for _, cg := range buildTags {
			for _, c := range cg.List {
				generatedCode.WriteString(c.Text + "\n")
			}
		}
		generatedCode.WriteString("\n")
	}

	// 2. Write Copyright
	for i, line := range copyrightLines {
		generatedCode.WriteString("// " + line)
		if i < len(copyrightLines)-1 {
			generatedCode.WriteString("\n")
		}
	}
	generatedCode.WriteString("\n\n") // Blank line after copyright

	// 3. Write the rest
	if err := format.Node(&generatedCode, fset, parsed); err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(generatedCode.String()), defaultFilePerm)
}

func isBuildTag(cg *ast.CommentGroup) bool {
	if len(cg.List) == 0 {
		return false
	}
	first := cg.List[0].Text
	return strings.HasPrefix(strings.TrimSpace(first), "//go:build") ||
		strings.HasPrefix(strings.TrimSpace(first), "// +build")
}

// filterComments removes comments that come before the package declaration
// UNLESS they look like build tags (which we extracted separately) or we want to keep them?
// Actually, earlier we extracted build tags. ensuring we don't duplicate them.
// `filterComments` is now responsible for removing "Old Copyright" or "Header Comments".
func filterComments(comments []*ast.CommentGroup, packagePos token.Pos) []*ast.CommentGroup {
	var newComments []*ast.CommentGroup
	for _, group := range comments {
		// If the comment is after the package keyword, keep it always.
		if group.Pos() >= packagePos {
			newComments = append(newComments, group)
			continue
		}

		// If it's before package:
		// We already extracted build tags in `fixFile`, so if this IS a build tag,
		// we shouldn't keep it here (to avoid duplication).
		if isBuildTag(group) {
			continue
		}

		// It's a comment before package, and NOT a build tag.
		// Assume it's a copyright header or garbage that should be replaced.
		// Drop it.
	}
	return newComments
}

// Copied from golang.org/x/tools/gopls/internal/golang/util.go.
var generatedRx = regexp.MustCompile(`// .*DO NOT EDIT\.?`)

func isGenerated(fset *token.FileSet, file *ast.File) bool {
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			if matched := generatedRx.MatchString(comment.Text); !matched {
				continue
			}
			// Check if comment is at the beginning of the line in source.
			if pos := fset.Position(comment.Slash); pos.Column == 1 {
				return true
			}
		}
	}
	return false
}
