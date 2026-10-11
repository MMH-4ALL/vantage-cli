package cli

import (
	"fmt"
	"github.com/MMH-4ALL/vantage-cli/internal/app"
	"github.com/spf13/cobra"
)

func gitCommand(o *options) *cobra.Command {
	root := &cobra.Command{Use: "git", Short: "Read-only Git status, history, and branch"}
	for _, name := range []string{"status", "log", "branch"} {
		name := name
		command := &cobra.Command{Use: name, Short: "Show repository " + name, Args: cobra.NoArgs}
		command.RunE = o.with(func(a *app.App, c *cobra.Command, _ []string) error {
			p, err := a.Core.Project(o.project)
			if err != nil {
				return err
			}
			v, err := a.Git.Inspect(c.Context(), p.Path)
			if err != nil {
				return err
			}
			switch name {
			case "branch":
				return output(c, o.json, v.Branch)
			case "log":
				if o.json {
					return output(c, true, v.Commits)
				}
				if len(v.Commits) == 0 {
					return output(c, false, "No commits yet")
				}
				for _, commit := range v.Commits {
					if err = output(c, false, fmt.Sprintf("%s  %s  (%s)", commit.Short, commit.Subject, commit.Author)); err != nil {
						return err
					}
				}
			default:
				if o.json {
					return output(c, true, v)
				}
				if err = output(c, false, "Branch: "+v.Branch+"\nRoot: "+v.Root); err != nil {
					return err
				}
				if v.Clean {
					return output(c, false, "Working tree is clean")
				}
				for _, change := range v.Changes {
					if err = output(c, false, fmt.Sprintf("%s%s  %s", change.Index, change.Worktree, change.Path)); err != nil {
						return err
					}
				}
			}
			return nil
		})
		root.AddCommand(command)
	}
	return root
}
