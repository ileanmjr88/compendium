package cli

import (
	"os"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
	"github.com/ileanmjr88/compendium/internal/installer"
	"github.com/ileanmjr88/compendium/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check whether installed tools match compendium.toml",
	Run: func(cmd *cobra.Command, args []string) {
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

		paths, err := env.NewPaths()
		if err != nil {
			ui.Print(ui.Fail, "getting paths", err.Error())
			os.Exit(1)
		}

		items := installer.Resolve(*cfg)

		if len(items) == 0 {
			ui.Print(ui.Success, "nothing to check", "")
			return
		}

		missing := 0
		for _, item := range items {
			if paths.CheckDir(item.Kind, item.Name, item.Version) {
				ui.Print(ui.Success, item.Name, item.Version)
			} else {
				ui.Print(ui.Fail, item.Name, item.Version+" not installed")
				missing++
			}
		}

		if missing == 0 {
			ui.Print(ui.Success, "environment in sync", "")
		} else if missing > 0 {
			ui.Print(ui.Warning, "environment out of sync", "run 'compendium install' to update")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
