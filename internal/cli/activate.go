package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
	"github.com/ileanmjr88/compendium/internal/shell"
	"github.com/spf13/cobra"
)

var activateCmd = &cobra.Command{
	Use:   "activate",
	Short: "Print shell script that adds Compendium tools to PATH (source it)",
	Long: `Print a POSIX shell script that prepends Compendium-managed tool directories
to PATH and exports per-language environment variables for the project.

The script has to run in your current shell to modify the environment.
That means you must source the output, not execute it directly.`,
	Example:      "  source <(compendium activate)",
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Load config
		cfg, result, err := config.Load("compendium.toml")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: loading config: %s\n", err.Error())
			os.Exit(1)
		}
		if result.HasErrors() {
			for _, e := range result.Errors {
				fmt.Fprintf(os.Stderr, "error: %s\n", e)
			}
			os.Exit(1)
		}
		for _, w := range result.Warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w)
		}

		// Resolve project name: explicit or cwd basename
		if cfg.Compendium.Name == "" {
			dir, _ := os.Getwd()
			cfg.Compendium.Name = filepath.Base(dir)
		}

		// 2. Set up paths
		paths, err := env.NewPaths()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: setting up paths: %s\n", err.Error())
			os.Exit(1)
		}

		// 3. Shell activate script generation
		dirs, envVars, warnings := shell.ResolveEnv(*cfg, paths)
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w)
		}

		fmt.Print(shell.ActivateScript(dirs, envVars))
	},
}

func init() {
	rootCmd.AddCommand(activateCmd)
}
