// Copyright 2026 Ilean Monterrubio Jr
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/ileanmjr88/compendium/internal/cli"
	"os"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
