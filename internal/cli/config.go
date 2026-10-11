package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/MMH-4ALL/vantage-cli/internal/app"
	"github.com/MMH-4ALL/vantage-cli/internal/config"
	"github.com/spf13/cobra"
)

func configCommand(o *options) *cobra.Command {
	r := &cobra.Command{Use: "config", Short: "Inspect and change validated TOML settings"}
	show := &cobra.Command{Use: "show", Short: "Show effective settings", Args: cobra.NoArgs, RunE: o.with(func(a *app.App, c *cobra.Command, _ []string) error {
		if o.json {
			return output(c, true, a.Settings)
		}
		b, e := config.Encode(a.Settings)
		if e != nil {
			return e
		}
		_, e = c.OutOrStdout().Write(b)
		return e
	})}
	path := &cobra.Command{Use: "path", Short: "Show configuration and data file locations", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		p, e := config.Resolve(o.home)
		if e != nil {
			return e
		}
		if o.json {
			return output(c, true, p)
		}
		return output(c, false, "Config: "+p.Config+"\nData: "+p.Data)
	}}
	reset := &cobra.Command{Use: "reset", Short: "Reset settings; preserve projects, tasks, and workflows", Args: cobra.NoArgs, RunE: o.with(func(a *app.App, c *cobra.Command, _ []string) error {
		if e := confirm(c, o.yes, "Overwrite your settings with defaults?"); e != nil {
			return e
		}
		if e := a.SaveSettings(config.Defaults()); e != nil {
			return e
		}
		return output(c, false, "Settings reset")
	})}
	set := &cobra.Command{Use: "set <key> <value>", Short: "Set a validated setting; editor_args is a JSON array", Args: cobra.ExactArgs(2), RunE: o.with(func(a *app.App, c *cobra.Command, args []string) error {
		s := a.Settings
		value := args[1]
		var e error
		switch args[0] {
		case "theme":
			s.Theme = value
		case "editor":
			s.Editor = value
		case "editor_args":
			e = json.Unmarshal([]byte(value), &s.EditorArgs)
		case "git_executable":
			s.GitExecutable = value
		case "default_project_directory":
			s.DefaultProjectDirectory = value
		case "refresh_seconds":
			s.RefreshSeconds, e = strconv.Atoi(value)
		case "startup_screen":
			s.StartupScreen = value
		case "animations":
			s.Animations, e = strconv.ParseBool(value)
		case "confirm_safe_operations":
			s.ConfirmSafeOperations, e = strconv.ParseBool(value)
		default:
			return fmt.Errorf("unknown setting %q", args[0])
		}
		if e != nil {
			return e
		}
		if a.Warning != "" {
			if e = confirm(c, o.yes, "Replace invalid configuration with these validated settings?"); e != nil {
				return e
			}
		}
		if e = a.SaveSettings(s); e != nil {
			return e
		}
		return output(c, false, "Setting saved")
	})}
	r.AddCommand(show, path, reset, set)
	return r
}
