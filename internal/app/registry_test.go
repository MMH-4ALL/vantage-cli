package app

import (
	"github.com/spf13/cobra"
	"testing"
)

func TestExtensionRegistration(t *testing.T) {
	r := DefaultRegistry()
	ran := false
	if e := r.AddAction(Action{ID: "example:hello", Title: "Say hello", Description: "An independently registered extension", Handler: func() error { ran = true; return nil }}); e != nil {
		t.Fatal(e)
	}
	items := r.FilterActions("hello")
	if len(items) != 1 {
		t.Fatal("extension missing from palette")
	}
	if e := items[0].Handler(); e != nil || !ran {
		t.Fatal("extension did not execute")
	}
	if r.AddAction(items[0]) == nil {
		t.Fatal("duplicate action accepted")
	}
	r.AddCommand(func() *cobra.Command {
		return &cobra.Command{Use: "hello", Run: func(*cobra.Command, []string) { ran = true }}
	})
	ran = false
	root := &cobra.Command{Use: "root"}
	for _, factory := range r.Commands {
		root.AddCommand(factory())
	}
	root.SetArgs([]string{"hello"})
	if e := root.Execute(); e != nil || !ran {
		t.Fatal("extension command failed")
	}
}
