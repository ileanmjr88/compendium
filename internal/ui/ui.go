package ui

import (
	"fmt"

	"github.com/schollz/progressbar/v3"
)

const (
	Green  = "\033[32m"
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Reset  = "\033[0m"
)

const (
	Action   = Blue + "→" + Reset
	Success  = Green + "✓" + Reset
	Fail     = Red + "✗" + Reset
	Download = Yellow + "↓" + Reset
	Warning  = Yellow + "!" + Reset
)

func Print(symbol, message, detail string) {
	if detail != "" {
		fmt.Printf("  %s %s  %s\n", symbol, message, detail)
	} else {
		fmt.Printf("  %s %s\n", symbol, message)
	}
}

func PrintDetail(symbol, message, detail string) {
	if detail != "" {
		fmt.Printf("    %s %s  %s\n", symbol, message, detail)
	} else {
		fmt.Printf("    %s %s\n", symbol, message)
	}
}

func ProgressBar(size int64, name string) *progressbar.ProgressBar {
	return progressbar.NewOptions64(size,
		progressbar.OptionSetDescription("  "+Download+" "+name),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerPadding: " ",
			BarStart:      "|",
			BarEnd:        "|",
		}),
	)
}
