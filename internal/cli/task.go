package cli

import (
	"fmt"
	"strings"

	"github.com/MMH-4ALL/vantage-cli/internal/app"
	"github.com/MMH-4ALL/vantage-cli/internal/domain"
	"github.com/spf13/cobra"
)

func taskCommand(o *options) *cobra.Command {
	r := &cobra.Command{Use: "task", Short: "Manage local project tasks"}
	var description, filter string
	add := &cobra.Command{Use: "add [title]", Short: "Create a task", Args: cobra.MaximumNArgs(1), RunE: o.with(func(a *app.App, c *cobra.Command, args []string) error {
		title := ""
		if len(args) > 0 {
			title = args[0]
		} else {
			answers, e := guided(c, []string{"Task title", "Description (optional)"})
			if e != nil {
				return e
			}
			title, description = answers[0], answers[1]
		}
		t, e := a.Core.AddTask(title, description, o.project)
		if e != nil {
			return e
		}
		if o.json {
			return output(c, true, t)
		}
		return output(c, false, "Created task "+t.ID+": "+t.Title)
	})}
	add.Flags().StringVarP(&description, "description", "d", "", "Task description")
	list := &cobra.Command{Use: "list", Short: "List tasks; filter pending/completed/all", Args: cobra.NoArgs, RunE: o.with(func(a *app.App, c *cobra.Command, _ []string) error {
		if filter != "pending" && filter != "completed" && filter != "all" {
			return fmt.Errorf("filter must be pending, completed, or all")
		}
		v, e := a.Core.Snapshot()
		if e != nil {
			return e
		}
		pid := ""
		if o.project != "" {
			p, e := a.Core.Project(o.project)
			if e != nil {
				return e
			}
			pid = p.ID
		}
		out := []domain.Task{}
		for _, t := range v.Tasks {
			if pid != "" && t.ProjectID != pid || filter == "pending" && t.Completed || filter == "completed" && !t.Completed {
				continue
			}
			out = append(out, t)
		}
		if o.json {
			return output(c, true, out)
		}
		if len(out) == 0 {
			return output(c, false, "No matching tasks. Create one with vantage task add \"Title\".")
		}
		for _, t := range out {
			mark := "[ ]"
			if t.Completed {
				mark = "[x]"
			}
			if e = output(c, false, fmt.Sprintf("%s %s  %s", mark, t.ID, t.Title)); e != nil {
				return e
			}
		}
		return nil
	})}
	list.Flags().StringVar(&filter, "filter", "all", "pending, completed, or all")
	for _, n := range []string{"done", "undo", "remove"} {
		n := n
		r.AddCommand(&cobra.Command{Use: n + " <id>", Short: map[string]string{"done": "Complete a task", "undo": "Reopen a task", "remove": "Delete a task record"}[n], Args: cobra.ExactArgs(1), RunE: o.with(func(a *app.App, c *cobra.Command, args []string) error {
			var e error
			if n == "remove" {
				if e = confirm(c, o.yes, "Delete task "+args[0]+"?"); e != nil {
					return e
				}
				e = a.Core.RemoveTask(args[0])
			} else {
				e = a.Core.CompleteTask(args[0], n == "done")
			}
			if e != nil {
				return e
			}
			return output(c, false, "Task updated")
		})})
	}
	var editDescription string
	edit := &cobra.Command{Use: "edit <id> <title>", Short: "Edit a task title and optional description", Args: cobra.ExactArgs(2), RunE: o.with(func(a *app.App, c *cobra.Command, args []string) error {
		d := editDescription
		if !c.Flags().Changed("description") {
			v, e := a.Core.Snapshot()
			if e != nil {
				return e
			}
			for _, t := range v.Tasks {
				if t.ID == args[0] {
					d = t.Description
				}
			}
		}
		if e := a.Core.EditTask(args[0], strings.TrimSpace(args[1]), d); e != nil {
			return e
		}
		return output(c, false, "Task updated")
	})}
	edit.Flags().StringVarP(&editDescription, "description", "d", "", "Replace the description")
	r.AddCommand(add, list, edit)
	return r
}
