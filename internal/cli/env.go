package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
	"github.com/ileanmjr88/compendium/internal/shell"
	"github.com/ileanmjr88/compendium/internal/ui"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print resolved environment variables without changing the shell",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Load Config
		cfg, result, err := config.Load("compendium.toml")
		if err != nil {
			ui.Print(ui.Fail, "loading config", err.Error())
			os.Exit(1)
		}

		if result.HasErrors() {
			for _, e := range result.Errors {
				ui.Print(ui.Fail, e, "")
			}
			os.Exit(1)
		}

		if result.HasWarnings() {
			for _, w := range result.Warnings {
				ui.Print(ui.Warning, w, "")
			}
		}

		// 2. Resolve project name
		if cfg.Compendium.Name == "" {
			dir, _ := os.Getwd()
			cfg.Compendium.Name = filepath.Base(dir)
		}

		// 3. Set up paths
		paths, err := env.NewPaths()
		if err != nil {
			ui.Print(ui.Fail, "setting up paths", err.Error())
			os.Exit(1)
		}

		_, envVars, warnings := shell.ResolveEnv(*cfg, paths)
		for _, w := range warnings {
			ui.Print(ui.Warning, w, "")
		}

		fmt.Println()
		fmt.Println("Environment variables:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "  NAME\tACTION\tVALUE")
		for _, v := range envVars {
			_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\n", v.Name, v.Action, v.Value)
		}
		_ = w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
