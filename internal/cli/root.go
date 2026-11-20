// Copyright 2026 Ilean Monterrubio Jr
// SPDX-License-Identifier: Apache-2.0

package cli

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "compendium",
	Short: "Reproducible developer environments, declared in one config.",
}

func Execute() error {
	return rootCmd.Execute()
}
