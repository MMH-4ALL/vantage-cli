// Package app wires dependencies and lightweight extension registries.
package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/MMH-4ALL/vantage-cli/internal/config"
	"github.com/MMH-4ALL/vantage-cli/internal/core"
	"github.com/MMH-4ALL/vantage-cli/internal/services/git"
	"github.com/MMH-4ALL/vantage-cli/internal/services/system"
	"github.com/MMH-4ALL/vantage-cli/internal/storage"
)

type App struct {
	mu       sync.RWMutex
	Core     *core.Service
	Settings config.Settings
	Paths    config.Paths
	Git      git.Service
	Registry *Registry
	Started  time.Time
	Warning  string
	Version  string
}

func Open(home, version string) (*App, error) {
	paths, e := config.Resolve(home)
	if e != nil {
		return nil, e
	}
	settings, warn := config.Load(paths.Config)
	st, e := storage.Open(paths.Data)
	if e != nil {
		return nil, e
	}
	a := &App{Core: core.New(st), Settings: settings, Paths: paths, Git: git.Service{Executable: settings.GitExecutable}, Registry: DefaultRegistry(), Started: time.Now(), Version: version}
	if warn != nil {
		a.Warning = warn.Error()
	}
	return a, nil
}
func (a *App) Close() error { return a.Core.Store.Close() }
func (a *App) SaveSettings(s config.Settings) error {
	if e := config.Save(a.Paths.Config, s); e != nil {
		return e
	}
	a.ApplySettings(s)
	return nil
}
func (a *App) ApplySettings(s config.Settings) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Settings = s
	a.Git.Executable = s.GitExecutable
	a.Warning = ""
}
func (a *App) settingsSnapshot() (config.Settings, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Settings, a.Warning
}
func (a *App) System() system.Info { return system.Collect(a.Started) }

type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func (a *App) Doctor(ctx context.Context) []Check {
	settings, warning := a.settingsSnapshot()
	c := []Check{{"Version", "ok", a.Version}}
	if warning == "" {
		c = append(c, Check{"Configuration", "ok", a.Paths.Config})
	} else {
		c = append(c, Check{"Configuration", "warning", warning})
	}
	if _, e := a.Core.Snapshot(); e != nil {
		c = append(c, Check{"Database", "error", e.Error()})
	} else {
		c = append(c, Check{"Database", "ok", a.Paths.Data})
	}
	for _, p := range []struct{ name, command string }{{"Git", settings.GitExecutable}, {"Editor", settings.Editor}} {
		if resolved, e := exec.LookPath(p.command); e != nil {
			c = append(c, Check{p.name, "warning", "Executable not found; configure its path in Settings"})
		} else {
			c = append(c, Check{p.name, "ok", resolved})
		}
	}
	f, e := os.CreateTemp(filepath.Dir(a.Paths.Data), ".doctor-*")
	if e == nil {
		n := f.Name()
		f.Close()
		os.Remove(n)
		c = append(c, Check{"Data permissions", "ok", "Data directory is writable"})
	} else {
		c = append(c, Check{"Data permissions", "error", e.Error()})
	}
	term := os.Getenv("TERM")
	if term == "dumb" {
		c = append(c, Check{"Terminal", "warning", "TERM=dumb; use CLI commands or a capable terminal"})
	} else {
		c = append(c, Check{"Terminal", "ok", fmt.Sprintf("TERM=%q; colors are detected at launch", term)})
	}
	v, e := a.Core.Snapshot()
	if e == nil {
		for _, w := range v.Workflows {
			select {
			case <-ctx.Done():
				return c
			default:
			}
			status, detail := "ok", "Validated"
			if e = core.ValidateWorkflow(w); e != nil {
				status, detail = "warning", e.Error()
			}
			if _, err := exec.LookPath(w.Executable); err != nil {
				status, detail = "warning", "Executable not found in PATH"
			}
			c = append(c, Check{"Workflow: " + w.Name, status, detail})
		}
	}
	return c
}
