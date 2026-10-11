package cli

import (
	"fmt"
	"github.com/MMH-4ALL/vantage-cli/internal/app"
	"github.com/MMH-4ALL/vantage-cli/internal/services/executor"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"os"
)

func doctorCommand(o *options) *cobra.Command {
	c := &cobra.Command{Use: "doctor", Short: "Check configuration, storage, tools, and workflows", Args: cobra.NoArgs}
	c.RunE = o.with(func(a *app.App, c *cobra.Command, _ []string) error {
		checks := a.Doctor(c.Context())
		if o.json {
			return output(c, true, checks)
		}
		for _, check := range checks {
			tag := "[" + check.Status + "]"
			if term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == "" {
				color := lipgloss.Color("#34D399")
				if check.Status == "warning" {
					color = lipgloss.Color("#FBBF24")
				}
				if check.Status == "error" {
					color = lipgloss.Color("#FB7185")
				}
				tag = lipgloss.NewStyle().Foreground(color).Render(tag)
			}
			fmt.Fprintf(c.OutOrStdout(), "%s %-20s %s\n", tag, executor.Clean(check.Name), executor.Clean(check.Detail))
		}
		for _, check := range checks {
			if check.Status == "error" {
				return &ExitError{1, "environment checks found an error"}
			}
		}
		return nil
	})
	return c
}
