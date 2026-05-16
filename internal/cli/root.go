package cli

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "compendium",
	Short: "Reproducible developer environments, declared in one config.",
}

func Execute() error {
	return rootCmd.Execute()
}
