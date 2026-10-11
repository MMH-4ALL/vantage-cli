// Package cli exposes shared services through scriptable Cobra commands.
package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MMH-4ALL/vantage-cli/internal/app"
	"github.com/MMH-4ALL/vantage-cli/internal/services/executor"
	"github.com/MMH-4ALL/vantage-cli/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string { return e.Message }

type options struct {
	home, project, version, commit, date string
	json, yes                            bool
}
type handler func(*app.App, *cobra.Command, []string) error

func (o *options) with(fn handler) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		a, e := app.Open(o.home, o.version)
		if e != nil {
			return e
		}
		defer a.Close()
		if a.Warning != "" {
			fmt.Fprintln(c.ErrOrStderr(), "warning:", executor.Clean(a.Warning))
		}
		return fn(a, c, args)
	}
}

func New(version, commit, date string) *cobra.Command {
	o := &options{version: version, commit: commit, date: date}
	r := &cobra.Command{Use: "vantage", Short: "Your entire development workspace. One terminal.", Long: "VANTAGE is a local-first developer command center. Run without a subcommand to open the dashboard.", SilenceErrors: true, SilenceUsage: true, RunE: o.with(func(a *app.App, c *cobra.Command, _ []string) error {
		if o.json {
			return errors.New("choose a read command such as project list --json")
		}
		if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) || os.Getenv("TERM") == "dumb" {
			return errors.New("the dashboard requires an interactive terminal; use vantage --help for CLI commands")
		}
		if o.project != "" {
			if e := a.Core.Activate(o.project); e != nil {
				return e
			}
		}
		return tui.Run(c.Context(), a)
	})}
	r.PersistentFlags().StringVar(&o.home, "home", "", "Isolate configuration and data in this directory (or VANTAGE_HOME)")
	r.PersistentFlags().StringVarP(&o.project, "project", "p", "", "Project ID or name; defaults to the active project")
	r.PersistentFlags().BoolVar(&o.json, "json", false, "Machine-readable output for read commands")
	r.PersistentFlags().BoolVarP(&o.yes, "yes", "y", false, "Confirm the requested operation without prompting")
	registry := app.DefaultRegistry()
	for _, factory := range []func() *cobra.Command{func() *cobra.Command { return projectCommand(o) }, func() *cobra.Command { return gitCommand(o) }, func() *cobra.Command { return workflowCommand(o) }, func() *cobra.Command { return taskCommand(o) }, func() *cobra.Command { return configCommand(o) }, func() *cobra.Command { return systemCommand(o) }, func() *cobra.Command { return doctorCommand(o) }} {
		registry.AddCommand(factory)
	}
	for _, factory := range registry.Commands {
		r.AddCommand(factory())
	}
	r.AddCommand(&cobra.Command{Use: "version", Short: "Show version and build information", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		v := map[string]string{"version": version, "commit": commit, "built": date}
		if o.json {
			return output(c, true, v)
		}
		_, e := fmt.Fprintf(c.OutOrStdout(), "VANTAGE %s\ncommit: %s\nbuilt: %s\n", version, commit, date)
		return e
	}})
	return r
}

func output(c *cobra.Command, jsonMode bool, v any) error {
	if jsonMode {
		e := json.NewEncoder(c.OutOrStdout())
		e.SetIndent("", "  ")
		return e.Encode(v)
	}
	_, e := fmt.Fprintln(c.OutOrStdout(), executor.Clean(fmt.Sprint(v)))
	return e
}
func confirm(c *cobra.Command, yes bool, prompt string) error {
	if yes {
		return nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("confirmation required; repeat the command with --yes")
	}
	fmt.Fprint(c.ErrOrStderr(), executor.Clean(prompt)+" [y/N] ")
	s, e := bufio.NewReader(c.InOrStdin()).ReadString('\n')
	if e != nil && e != io.EOF {
		return e
	}
	if strings.EqualFold(strings.TrimSpace(s), "y") || strings.EqualFold(strings.TrimSpace(s), "yes") {
		return nil
	}
	return errors.New("operation canceled")
}
func guided(c *cobra.Command, questions []string) ([]string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errors.New("guided creation requires a terminal; provide the documented flags")
	}
	reader := bufio.NewReader(c.InOrStdin())
	var values []string
	for _, q := range questions {
		fmt.Fprint(c.ErrOrStderr(), q+": ")
		line, e := reader.ReadString('\n')
		if e != nil {
			return nil, e
		}
		values = append(values, strings.TrimSpace(line))
	}
	return values, nil
}
