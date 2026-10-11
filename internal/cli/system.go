package cli

import (
	"fmt"
	"github.com/MMH-4ALL/vantage-cli/internal/app"
	"github.com/spf13/cobra"
)

func systemCommand(o *options) *cobra.Command {
	r := &cobra.Command{Use: "system", Short: "Measured machine and application information"}
	r.AddCommand(&cobra.Command{Use: "info", Short: "Show system and process information", Args: cobra.NoArgs, RunE: o.with(func(a *app.App, c *cobra.Command, _ []string) error {
		v := a.System()
		if o.json {
			return output(c, true, v)
		}
		s := fmt.Sprintf("OS: %s\nArchitecture: %s\nLogical CPUs: %d\nHostname: %s\nGo heap: %.2f MiB\nGo runtime reserved: %.2f MiB\nUptime: %.2fs\nRuntime: %s\nDirectory: %s", v.OS, v.Architecture, v.CPUs, v.Hostname, float64(v.HeapBytes)/1048576, float64(v.RuntimeReservedBytes)/1048576, v.UptimeSeconds, v.GoVersion, v.Directory)
		if v.ResidentBytes != nil {
			s += fmt.Sprintf("\nResident memory: %.2f MiB", float64(*v.ResidentBytes)/1048576)
		}
		return output(c, false, s)
	})})
	return r
}
