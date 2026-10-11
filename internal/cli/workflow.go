package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/MMH-4ALL/vantage-cli/internal/app"
	"github.com/MMH-4ALL/vantage-cli/internal/domain"
	"github.com/MMH-4ALL/vantage-cli/internal/services/executor"
	"github.com/spf13/cobra"
)

func workflowCommand(o *options) *cobra.Command {
	r := &cobra.Command{Use: "workflow", Short: "Save and execute explicit, repeatable commands"}
	var name, executable, directory, description string
	var args, env, inherited []string
	var timeout int
	add := &cobra.Command{Use: "add", Short: "Create a workflow with flags or guided prompts", Args: cobra.NoArgs, RunE: o.with(func(a *app.App, c *cobra.Command, _ []string) error {
		if name == "" || executable == "" {
			answers, e := guided(c, []string{"Workflow name", "Executable", "Working directory (blank uses active project or current directory)"})
			if e != nil {
				return e
			}
			name, executable, directory = answers[0], answers[1], answers[2]
		}
		if directory == "" {
			if p, e := a.Core.Project(o.project); e == nil {
				directory = p.Path
			} else {
				directory, _ = os.Getwd()
			}
		}
		m := map[string]string{}
		for _, v := range env {
			k, value, ok := strings.Cut(v, "=")
			if !ok {
				return fmt.Errorf("--env expects KEY=VALUE")
			}
			m[k] = value
		}
		w, e := a.Core.AddWorkflow(domain.Workflow{Name: name, Description: description, ProjectID: o.project, Executable: executable, Args: args, Directory: directory, TimeoutSeconds: timeout, Env: m, InheritEnv: inherited})
		if e != nil {
			return e
		}
		if o.json {
			return output(c, true, w)
		}
		return output(c, false, "Saved workflow "+w.Name+" ("+w.ID+"); approval is required before its first run")
	})}
	add.Flags().StringVar(&name, "name", "", "Unique workflow name")
	add.Flags().StringVar(&executable, "exec", "", "Executable path or name; no implicit shell")
	add.Flags().StringArrayVar(&args, "arg", nil, "One literal argument (repeat for each argument)")
	add.Flags().StringVar(&directory, "dir", "", "Working directory")
	add.Flags().StringVar(&description, "description", "", "Description")
	add.Flags().IntVar(&timeout, "timeout", 300, "Timeout in seconds, up to 86400")
	add.Flags().StringArrayVar(&env, "env", nil, "Nonsecret KEY=VALUE override (repeatable)")
	add.Flags().StringArrayVar(&inherited, "inherit-env", nil, "Require a live environment variable; store its name only")
	list := &cobra.Command{Use: "list", Short: "List saved workflows", Args: cobra.NoArgs, RunE: o.with(func(a *app.App, c *cobra.Command, _ []string) error {
		v, e := a.Core.Snapshot()
		if e != nil {
			return e
		}
		out := []domain.Workflow{}
		pid := ""
		if o.project != "" {
			p, e := a.Core.Project(o.project)
			if e != nil {
				return e
			}
			pid = p.ID
		}
		for _, w := range v.Workflows {
			if pid == "" || pid == w.ProjectID {
				out = append(out, w)
			}
		}
		if o.json {
			return output(c, true, out)
		}
		if len(out) == 0 {
			return output(c, false, "No workflows. Use vantage workflow add --help.")
		}
		for _, w := range out {
			if e = output(c, false, fmt.Sprintf("%s  %-18s  %s  [%ds]", w.ID, w.Name, w.Executable, w.TimeoutSeconds)); e != nil {
				return e
			}
		}
		return nil
	})}
	run := &cobra.Command{Use: "run <workflow>", Short: "Run a workflow; Ctrl+C cancels it", Args: cobra.ExactArgs(1), RunE: o.with(func(a *app.App, c *cobra.Command, args []string) error {
		w, e := a.Core.Workflow(args[0])
		if e != nil {
			return e
		}
		if !w.Approved {
			if e = confirm(c, o.yes, fmt.Sprintf("Run %s (%s with %d arguments) in %s with your permissions?", w.Name, w.Executable, len(w.Args), w.Directory)); e != nil {
				return e
			}
			if e = a.Core.ApproveWorkflow(w.ID); e != nil {
				return e
			}
		}
		result := executor.Run(c.Context(), w, executor.NewLog())
		if o.json {
			if e = output(c, true, result); e != nil {
				return e
			}
		} else {
			fmt.Fprint(c.OutOrStdout(), result.Output)
			fmt.Fprintf(c.ErrOrStderr(), "\n%s: exit %d, elapsed %s\n", executor.Clean(w.Name), result.ExitCode, result.Elapsed.Round(1e6))
		}
		if result.Error != "" {
			code := result.ExitCode
			if code < 1 || code > 125 {
				code = 1
			}
			if result.Canceled {
				code = 130
			}
			return &ExitError{code, result.Error}
		}
		return nil
	})}
	remove := &cobra.Command{Use: "remove <workflow>", Short: "Delete a saved workflow", Args: cobra.ExactArgs(1), RunE: o.with(func(a *app.App, c *cobra.Command, args []string) error {
		if e := confirm(c, o.yes, "Delete the saved workflow "+args[0]+"?"); e != nil {
			return e
		}
		if e := a.Core.RemoveWorkflow(args[0]); e != nil {
			return e
		}
		return output(c, false, "Workflow removed")
	})}
	r.AddCommand(add, list, run, remove)
	return r
}
