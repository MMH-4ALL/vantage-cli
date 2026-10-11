package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/MMH-4ALL/vantage-cli/internal/app"
	"github.com/MMH-4ALL/vantage-cli/internal/domain"
	"github.com/MMH-4ALL/vantage-cli/internal/services/executor"
	"github.com/spf13/cobra"
)

func projectCommand(o *options) *cobra.Command {
	r := &cobra.Command{Use: "project", Short: "Register, switch, and inspect local projects"}
	var name string
	add := &cobra.Command{Use: "add <directory>", Short: "Register a directory and make it active", Args: cobra.ExactArgs(1), RunE: o.with(func(a *app.App, c *cobra.Command, args []string) error {
		p, e := a.Core.AddProject(args[0], name)
		if e != nil {
			return e
		}
		if o.json {
			return output(c, true, p)
		}
		return output(c, false, "Registered "+p.Name+" ("+p.ID+") at "+p.Path)
	})}
	add.Flags().StringVar(&name, "name", "", "Display name")
	list := &cobra.Command{Use: "list", Short: "List projects in recently used order", Args: cobra.NoArgs, RunE: o.with(func(a *app.App, c *cobra.Command, _ []string) error {
		v, e := a.Core.Snapshot()
		if e != nil {
			return e
		}
		if o.json {
			return output(c, true, v.Projects)
		}
		if len(v.Projects) == 0 {
			return output(c, false, "No projects. Register one with vantage project add .")
		}
		w := tabwriter.NewWriter(c.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tACTIVE\tPATH")
		for _, p := range v.Projects {
			active := ""
			if p.ID == v.State.ActiveProjectID {
				active = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, executor.Clean(p.Name), active, executor.Clean(p.Path))
		}
		return w.Flush()
	})}
	info := &cobra.Command{Use: "info [project]", Short: "Show project metadata", Args: cobra.MaximumNArgs(1), RunE: o.with(func(a *app.App, c *cobra.Command, args []string) error {
		ref := o.project
		if len(args) > 0 {
			ref = args[0]
		}
		p, e := a.Core.Project(ref)
		if e != nil {
			return e
		}
		if o.json {
			return output(c, true, p)
		}
		return output(c, false, fmt.Sprintf("%s\nID: %s\nPath: %s\nRegistered: %s\nLast used: %s", p.Name, p.ID, p.Path, p.CreatedAt.Format("2006-01-02 15:04"), p.AccessedAt.Format("2006-01-02 15:04")))
	})}
	use := &cobra.Command{Use: "use <project>", Short: "Set the active project", Args: cobra.ExactArgs(1), RunE: o.with(func(a *app.App, c *cobra.Command, args []string) error {
		if e := a.Core.Activate(args[0]); e != nil {
			return e
		}
		return output(c, false, "Active project updated")
	})}
	remove := &cobra.Command{Use: "remove <project>", Short: "Unregister a project; keep its files", Args: cobra.ExactArgs(1), RunE: o.with(func(a *app.App, c *cobra.Command, args []string) error {
		p, e := a.Core.Project(args[0])
		if e != nil {
			return e
		}
		if e = confirm(c, o.yes, "Unregister "+p.Name+"? Its files will remain."); e != nil {
			return e
		}
		if e = a.Core.RemoveProject(p.ID); e != nil {
			return e
		}
		return output(c, false, "Project unregistered; its directory was preserved")
	})}
	open := &cobra.Command{Use: "open [project]", Short: "Open a project in the configured editor", Args: cobra.MaximumNArgs(1), RunE: o.with(func(a *app.App, c *cobra.Command, args []string) error {
		ref := o.project
		if len(args) > 0 {
			ref = args[0]
		}
		p, e := a.Core.Project(ref)
		if e != nil {
			return e
		}
		if a.Settings.ConfirmSafeOperations {
			if e = confirm(c, o.yes, "Open "+p.Name+" in "+a.Settings.Editor+"?"); e != nil {
				return e
			}
		}
		w := domain.Workflow{Name: "Open editor", Executable: a.Settings.Editor, Args: append(append([]string{}, a.Settings.EditorArgs...), p.Path), Directory: p.Path, TimeoutSeconds: 86400}
		result := executor.Run(c.Context(), w, executor.NewLog())
		fmt.Fprint(c.OutOrStdout(), result.Output)
		if result.Error != "" {
			return fmt.Errorf("open editor: %s; configure editor as an executable and editor_args separately", result.Error)
		}
		return nil
	})}
	r.AddCommand(add, list, info, use, remove, open)
	r.Example = strings.TrimSpace("vantage project add . --name my-app\nvantage project use my-app\nvantage project list --json")
	return r
}
