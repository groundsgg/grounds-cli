package render

import (
	"os"

	"github.com/fatih/color"
)

// SetEnabled wires fatih/color to honour --no-color and NO_COLOR env
// var. Call once at startup from the root command.
func SetEnabled(disable bool) {
	if disable || os.Getenv("NO_COLOR") != "" {
		color.NoColor = true
		return
	}
	// Auto-detect from terminal capability (fatih/color does this by
	// default for color.NoColor=false).
	color.NoColor = false
}

func Green(s string) string  { return color.New(color.FgGreen).SprintFunc()(s) }
func Yellow(s string) string { return color.New(color.FgYellow).SprintFunc()(s) }
func Red(s string) string    { return color.New(color.FgRed).SprintFunc()(s) }
func Bold(s string) string   { return color.New(color.Bold).SprintFunc()(s) }
