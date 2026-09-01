# Package Description

[![Test](https://github.com/altessa-s/go-copyright-checker/actions/workflows/test.yml/badge.svg)](https://github.com/altessa-s/go-copyright-checker/actions/workflows/test.yml)
[![Lint](https://github.com/altessa-s/go-copyright-checker/actions/workflows/lint.yml/badge.svg)](https://github.com/altessa-s/go-copyright-checker/actions/workflows/lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/altessa-s/go-copyright-checker)](https://goreportcard.com/report/github.com/altessa-s/go-copyright-checker)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/altessa-s/go-copyright-checker)](https://github.com/altessa-s/go-copyright-checker/releases)

Package `go-copyright-checker` provides a set of utilities and a CLI tool for checking and ensuring copyright headers in Go source files. It is designed to be easily integrated into CI/CD pipelines or used as a standalone developer tool.

> This package was originally developed as an internal project at Altessa Solutions Inc. and is now open-sourced under the MIT License.

> AI tools were used during the open-source transition for: creating GitHub workflows and configuration files, updating and consolidating documentation, generating usage examples, and reviewing code for consistency.

## API Reference

### Core

| Component | Description |
|-----------|-------------|
| `checker.Check` | Main entry point for checking files. Returns an iterator of files needing changes or errors. |
| `checker.Config` | Configuration structure for custom templates, exclusions, and fix mode. |

## Installation

### Using Go

```bash
go install github.com/altessa-s/go-copyright-checker@latest
```

### From Source

```bash
git clone https://github.com/altessa-s/go-copyright-checker.git
cd go-copyright-checker
make build
```

## Usage

### CLI Tool

#### Check Copyrights

To check all Go files in the current directory (recursively):

```bash
go-copyright-checker check .
```

If any files are missing headers or have incorrect headers, the command will exit with a non-zero status and list the files.

#### Fix Copyrights

To automatically fix missing or incorrect headers:

```bash
go-copyright-checker check . --fix
```

#### Configuration

You can configure the tool using a `.copyright.yaml` file or command-line flags.

**Configuration File (`.copyright.yaml`)**

```yaml
template: |
  Copyright {{yearRange .START_YEAR .YEAR}} Altessa Solutions Inc. All rights reserved.
  Use of this source code is governed by license that can be found in 
  the LICENSE file.
variables:
  YEAR: 2026
  START_YEAR: 2021
```

**Variables**

- `{{.YEAR}}`: Defaults to the current year. You can override it in the config file.
- `{{.START_YEAR}}`: Defaults to the current year (same as `{{.YEAR}}`). Set it explicitly in the config file to pin the year a project started.

**Template Functions**

- `yearRange START END`: Renders `START-END`, or just `END` if `START` and `END` are equal (avoids headers like `2026-2026`).

**Command Line Flags**

- `-c, --config`: Path to configuration file (default: `.copyright.yaml`).
- `-e, --exclude`: Comma-separated list of directories to exclude.
- `-f, --fix`: Fix missing or incorrect headers.

### Library Usage

You can also use the `checker` package in your own Go tools:

```go
package main

import (
	"fmt"
	"github.com/altessa-s/go-copyright-checker/pkg/checker"
)

func main() {
	cfg := checker.Config{
		Dir:      ".",
		Template: "Copyright {{.YEAR}} My Corp",
		Data:     map[string]string{"YEAR": "2026"},
	}

	for file, err := range checker.Check(cfg) {
		if err != nil {
			panic(err)
		}
		fmt.Printf("Missing copyright: %s\n", file)
	}
}
```
