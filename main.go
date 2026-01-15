// Copyright 2021-2026 Altessa Solutions Inc. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package main

import (
	"os"

	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var RootCmd = &cobra.Command{
	Use:          "go-copyright",
	Short:        "This tools checks for missing copyright headers in Go files.",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: false,
}

func main() {
	RootCmd.SetArgs(os.Args[1:])
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
