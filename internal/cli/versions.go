package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ileanmjr88/compendium/internal/registry"
	"github.com/ileanmjr88/compendium/internal/ui"
	"github.com/spf13/cobra"
)

var versionsCmd = &cobra.Command{
	Use:   "versions [tool]",
	Short: "List available versions of tools in the registry",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		indexURL := registry.ResolveSource("public")
		client, err := registry.NewClient(indexURL)
		if err != nil {
			ui.Print(ui.Fail, "fetching index", err.Error())
			os.Exit(1)
		}

		if len(args) == 0 {
			printAll(client.ListAll())
			return
		}
		result, err := client.ListVersions(args[0])
		if err != nil {
			ui.Print(ui.Fail, "lookup", err.Error())
			os.Exit(1)
		}
		printOne(result)
	},
}

func printOne(t *registry.ToolVersions) {
	fmt.Println()
	fmt.Printf("  %s available versions:\n\n", t.Name)
	for _, v := range t.Versions {
		if v == t.Latest {
			fmt.Printf("  %s   ← latest\n", v)
		} else {
			fmt.Printf("  %s\n", v)
		}
	}
	fmt.Println()
	fmt.Println("  run 'compendium install' after updating compendium.toml")
}

func printAll(rows []registry.IndexRow) {
	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	var currentKind string
	for _, r := range rows {
		if r.Kind != currentKind {
			if currentKind != "" {
				_ = w.Flush()
				fmt.Println()
			}
			title := "Languages"
			if r.Kind == "tools" {
				title = "Tools"
			}
			fmt.Printf("  %s:\n", title)
			currentKind = r.Kind
		}
		plural := "versions"
		if r.Count == 1 {
			plural = "version"
		}
		_, _ = fmt.Fprintf(w, "    %s\t%s\t(%d %s)\n", r.Name, r.Latest, r.Count, plural)
	}
	_ = w.Flush()
	fmt.Println()
	fmt.Println("  run 'compendium versions <tool>' to see all versions of one tool")
}

func init() {
	rootCmd.AddCommand(versionsCmd)
}
