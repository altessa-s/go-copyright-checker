// Copyright 2021-2026 Altessa Solutions Inc. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/altessa-s/go-copyright-checker/pkg/checker"
)

var DefaultCopyrightTemplate = `Copyright 2021-{{.YEAR}} Altessa Solutions Inc. All rights reserved.
Use of this source code is governed by license that can be found in 
the LICENSE file.
`

var defaultExcludeDirs = []string{"vendor", "testdata", ".git", ".idea", "build", "bin"}

//nolint:gochecknoglobals
var cmdCheck = &cobra.Command{
	Use:          "check",
	Short:        "Check for missing copyright headers in Go files.",
	RunE:         run,
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
}

func init() {
	cmdCheck.Flags().StringP("config", "c", ".copyright.yaml",
		"path to the configuration file")
	cmdCheck.Flags().StringP("exclude", "e", "",
		"comma-separated list of directories to exclude from the check")
	cmdCheck.Flags().BoolP("fix", "f", false,
		"fix missing headers in Go files")

	RootCmd.AddCommand(cmdCheck)
}

func run(cmd *cobra.Command, args []string) error {
	path := args[0]

	templateContent := DefaultCopyrightTemplate
	variables := make(map[string]string)
	variables["YEAR"] = fmt.Sprint(time.Now().Year())

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	configPath = filepath.Clean(configPath)
	configPath = filepath.Join(path, configPath)

	if configPath, err = filepath.Abs(configPath); err != nil {
		return err
	}

	conf, err := LoadConfig(configPath)
	switch {
	case err != nil && os.IsNotExist(err):
		// No config file present; fall back to the default template.
	case err != nil:
		return fmt.Errorf("failed to load config: %w", err)
	default:
		for k, v := range conf.Variables {
			variables[strings.ToUpper(k)] = v
		}
		templateContent = conf.Template
	}

	var excludeDirs = make([]string, 0, len(defaultExcludeDirs))

	exclude, err := cmd.Flags().GetString("exclude")
	if err != nil {
		return err
	}

	if exclude != "" {
		for dir := range strings.SplitSeq(exclude, ",") {
			if trimmed := strings.TrimSpace(dir); trimmed != "" {
				excludeDirs = append(excludeDirs, trimmed)
			}
		}
	}

	excludeDirs = append(excludeDirs, defaultExcludeDirs...)
	excludeDirs = uniqueStrings(excludeDirs)

	fix, err := cmd.Flags().GetBool("fix")
	if err != nil {
		return err
	}

	// Using the new checker package
	cfg := checker.Config{
		Dir:         path,
		Fix:         fix,
		ExcludeDirs: excludeDirs,
		Template:    templateContent,
		Data:        variables,
	}

	//nolint:prealloc // We cannot know the number of errors upfront.
	var fileErrors []string

	// Consume the iterator
	for file, err := range checker.Check(cfg) {
		if err != nil {
			return err
		}
		fileErrors = append(fileErrors, file)
	}

	if len(fileErrors) > 0 {
		return fmt.Errorf("the following files are missing or have an incorrect copyright notice:\n%s",
			strings.Join(fileErrors, "\n"))
	}

	return nil
}

func uniqueStrings(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(s))
	for _, v := range s {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}
