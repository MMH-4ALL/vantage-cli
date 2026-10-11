package app

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

// Registry supports additional screens, palette actions, and CLI modules without a plugin runtime.
type Destination struct{ ID, Title, Description string }
type Action struct {
	ID, Title, Description, Destination string
	Handler                             func() error
}
type Registry struct {
	Destinations []Destination
	Actions      []Action
	Commands     []func() *cobra.Command
}

func (r *Registry) AddDestination(d Destination) error {
	for _, x := range r.Destinations {
		if x.ID == d.ID {
			return errors.New("duplicate destination")
		}
	}
	r.Destinations = append(r.Destinations, d)
	return nil
}
func (r *Registry) AddAction(a Action) error {
	for _, x := range r.Actions {
		if x.ID == a.ID {
			return errors.New("duplicate action")
		}
	}
	r.Actions = append(r.Actions, a)
	return nil
}
func (r *Registry) AddCommand(factory func() *cobra.Command) {
	r.Commands = append(r.Commands, factory)
}
func (r *Registry) FilterActions(q string) []Action {
	var out []Action
	for _, a := range r.Actions {
		if strings.Contains(strings.ToLower(a.Title+" "+a.Description), strings.ToLower(q)) {
			out = append(out, a)
		}
	}
	return out
}
func DefaultRegistry() *Registry {
	r := &Registry{}
	for _, d := range []Destination{{"overview", "Overview", "Your workspace at a glance"}, {"projects", "Projects", "Register and switch local workspaces"}, {"git", "Git", "Read-only repository insights"}, {"workflows", "Workflows", "Repeatable commands with live output"}, {"tasks", "Tasks", "Local project to-do lists"}, {"system", "System", "Measured application and machine information"}, {"settings", "Settings", "Theme, editor, refresh, and startup"}, {"doctor", "Doctor", "Check your local environment"}} {
		_ = r.AddDestination(d)
		_ = r.AddAction(Action{ID: "go:" + d.ID, Title: d.Title, Description: d.Description, Destination: d.ID})
	}
	return r
}
