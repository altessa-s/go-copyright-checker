// Copyright 2021-2026 Altessa Solutions Inc. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package main

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Variables map[string]string `yaml:"variables"`
	Template  string            `yaml:"template"`
}

func LoadConfig(file string) (*Config, error) {
	c := &Config{
		Variables: make(map[string]string),
	}

	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	c.Template = strings.TrimSpace(c.Template)

	return c, nil
}
