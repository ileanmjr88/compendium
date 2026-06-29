package cli

import (
	"errors"
	"os"
	"runtime"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
	"github.com/ileanmjr88/compendium/internal/installer"
	"github.com/ileanmjr88/compendium/internal/lockfile"
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

		lock := &lockfile.Lockfile{}
		lockExist := false
		lfLoaded, err := lockfile.Load("compendium.lock")
		switch {
		case err == nil:
			lock = lfLoaded
			lockExist = true
		case errors.Is(err, os.ErrNotExist):
			// first run
		default:
			ui.Print(ui.Fail, "loading lockfile", err.Error())
			os.Exit(1)
		}

		changes := lockfile.Diff(*cfg, lock, runtime.GOOS, runtime.GOARCH)
		drifted := len(changes) > 0
		if drifted {
			if lockExist {
				ui.Print(ui.Warning, "lockfile out of date", "run `compendium install`")
			} else {
				ui.Print(ui.Warning, "no lockfile", "run `compendium install`")
			}
			printChanges(changes)
		}

		paths, err := env.NewPaths()
		if err != nil {
			ui.Print(ui.Fail, "getting paths", err.Error())
			os.Exit(1)
		}

		items := installer.Resolve(*cfg)

		if len(items) == 0 {
			if drifted {
				os.Exit(1)
			}
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

		if !drifted && missing == 0 {
			ui.Print(ui.Success, "environment in sync", "")
			return
		}
		if missing > 0 {
			ui.Print(ui.Warning, "environment out of sync", "run 'compendium install' to update")
		}
		os.Exit(1)
	},
}

func printChanges(changes []lockfile.Change) {
	for _, c := range changes {
		section := "language"
		switch c.Section {
		case lockfile.SectionTool:
			section = "tool"
		case lockfile.SectionPackage:
			section = "package"
		}

		switch c.Kind {
		case lockfile.Added:
			ui.Print(ui.Warning, section+" "+c.Name, "+ "+c.NewSpec)
		case lockfile.Removed:
			ui.Print(ui.Warning, section+" "+c.Name, "- "+c.OldSpec)
		case lockfile.SpecChanged:
			ui.Print(ui.Warning, section+" "+c.Name, c.OldSpec+" → "+c.NewSpec)
		case lockfile.PlatformMissing:
			ui.Print(ui.Warning, section+" "+c.Name, runtime.GOOS+"/"+runtime.GOARCH+" missing")
		}
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
