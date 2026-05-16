package cli

import (
	"fmt"

	"github.com/ileanmjr88/compendium/internal/shell"
	"github.com/spf13/cobra"
)

var deactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Print shell script that undoes activate (source it)",
	Long: `Print a POSIX shell script that restores PATH and per-language environment
variables to the values they had before activate.

Like activate, the script has to run in your current shell to take effect.
Source the output rather than executing it.`,
	Example:      "  source <(compendium deactivate)",
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(shell.DeactivateScript())
	},
}

func init() {
	rootCmd.AddCommand(deactivateCmd)
}
