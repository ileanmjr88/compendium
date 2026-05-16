package cli

import (
	"fmt"
	"runtime"

	"github.com/ileanmjr88/compendium/internal/buildinfo"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Compendium binary version, commit, and platform",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("compendium %s\n", buildinfo.Version)
		fmt.Printf("  %-7s %s\n", "commit", buildinfo.Commit)
		fmt.Printf("  %-7s %s/%s\n", "os/arch", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
