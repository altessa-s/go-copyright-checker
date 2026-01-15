// Copyright 2021-2026 Altessa Solutions Inc. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"runtime"
	"runtime/debug"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var cmdVersion = &cobra.Command{
	Use:   "version",
	Short: "Displays service version information",
	Run:   cmdVersionRun,
}

var (
	// Version is the version of the application.
	Version = "0.0.0"

	// BuildTime is the build time of the application.
	buildTime = ""

	// Commit is the commit hash of the application.
	Commit = ""

	// GoVersion is Go tree's version string.
	goVersion = runtime.Version()
	goOS      = runtime.GOOS
	goArch    = runtime.GOARCH
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && Commit == "" {
				Commit = setting.Value
				if len(Commit) >= 6 { //nolint:mnd
					Commit = Commit[0:6] //nolint:mnd
				}
				Commit = strings.ToUpper(Commit)
			}

			if setting.Key == "vcs.time" && buildTime == "" {
				buildTime = setting.Value
			}
		}
	}

	RootCmd.AddCommand(cmdVersion)
}

func cmdVersionRun(_ *cobra.Command, _ []string) {
	m := map[string]string{
		"version":   Version,
		"revision":  Commit,
		"buildDate": buildTime,
		"goVersion": goVersion,
		"platform":  goOS + "/" + goArch,
	}
	t := template.Must(template.New("version").Parse(versionInfoTmpl))

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "version", m); err != nil {
		panic(err)
	}

	println(strings.TrimSpace(buf.String()))
}

var versionInfoTmpl = `
version {{.version}} (revision: {{.revision}})
  build date:       {{.buildDate}}
  go version:       {{.goVersion}}
  platform:         {{.platform}}
`
