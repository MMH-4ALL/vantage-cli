// Package config owns validated TOML settings and platform-standard paths.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Settings struct {
	Theme                   string   `toml:"theme" json:"theme"`
	DefaultProjectDirectory string   `toml:"default_project_directory" json:"default_project_directory"`
	Editor                  string   `toml:"editor" json:"editor"`
	EditorArgs              []string `toml:"editor_args" json:"editor_args"`
	GitExecutable           string   `toml:"git_executable" json:"git_executable"`
	Animations              bool     `toml:"animations" json:"animations"`
	RefreshSeconds          int      `toml:"refresh_seconds" json:"refresh_seconds"`
	StartupScreen           string   `toml:"startup_screen" json:"startup_screen"`
	ConfirmSafeOperations   bool     `toml:"confirm_safe_operations" json:"confirm_safe_operations"`
}

type Paths struct{ Config, Data string }

func Defaults() Settings {
	home, _ := os.UserHomeDir()
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "code"
	}
	return Settings{Theme: "midnight", DefaultProjectDirectory: home, Editor: editor, GitExecutable: "git", Animations: true, RefreshSeconds: 5, StartupScreen: "overview", ConfirmSafeOperations: true}
}

// Resolve lets tests and portable installs isolate all data using --home or VANTAGE_HOME.
func Resolve(home string) (Paths, error) {
	if home == "" {
		home = os.Getenv("VANTAGE_HOME")
	}
	if home != "" {
		a, e := filepath.Abs(home)
		if e != nil {
			return Paths{}, e
		}
		return Paths{filepath.Join(a, "config.toml"), filepath.Join(a, "data.db")}, nil
	}
	c, e := os.UserConfigDir()
	if e != nil {
		return Paths{}, e
	}
	var d string
	switch runtime.GOOS {
	case "windows":
		d = os.Getenv("LOCALAPPDATA")
		if d == "" {
			d = c
		}
	case "darwin":
		h, e := os.UserHomeDir()
		if e != nil {
			return Paths{}, e
		}
		d = filepath.Join(h, "Library", "Application Support")
	default:
		d = os.Getenv("XDG_DATA_HOME")
		if d == "" {
			h, e := os.UserHomeDir()
			if e != nil {
				return Paths{}, e
			}
			d = filepath.Join(h, ".local", "share")
		}
	}
	return Paths{filepath.Join(c, "vantage", "config.toml"), filepath.Join(d, "vantage", "data.db")}, nil
}

func Validate(s Settings) error {
	switch s.Theme {
	case "midnight", "light", "mono":
	default:
		return fmt.Errorf("theme must be midnight, light, or mono")
	}
	if s.RefreshSeconds < 1 || s.RefreshSeconds > 3600 {
		return errors.New("refresh_seconds must be between 1 and 3600")
	}
	switch s.StartupScreen {
	case "overview", "projects", "git", "workflows", "tasks", "system", "settings", "doctor":
	default:
		return errors.New("unknown startup_screen")
	}
	if strings.TrimSpace(s.GitExecutable) == "" {
		return errors.New("git_executable cannot be empty")
	}
	if strings.TrimSpace(s.Editor) == "" {
		return errors.New("editor cannot be empty")
	}
	if strings.IndexByte(s.Editor, 0) >= 0 || strings.IndexByte(s.GitExecutable, 0) >= 0 {
		return errors.New("executable paths cannot contain NUL characters")
	}
	return nil
}

// Load preserves invalid files and returns defaults with a warning.
func Load(path string) (Settings, error) {
	s := Defaults()
	b, e := os.ReadFile(path)
	if errors.Is(e, os.ErrNotExist) {
		return s, nil
	}
	if e != nil {
		return s, fmt.Errorf("read configuration: %w", e)
	}
	if e = toml.Unmarshal(b, &s); e != nil {
		return Defaults(), fmt.Errorf("configuration is invalid; using defaults without overwriting %s: %w", path, e)
	}
	if e = Validate(s); e != nil {
		return Defaults(), fmt.Errorf("configuration is invalid; using defaults without overwriting %s: %w", path, e)
	}
	return s, nil
}

func Encode(s Settings) ([]byte, error) { return toml.Marshal(s) }

// Save syncs a private temporary file before replacing the settings file.
func Save(path string, s Settings) error {
	if e := Validate(s); e != nil {
		return e
	}
	b, e := Encode(s)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".config-*")
	if e != nil {
		return e
	}
	n := f.Name()
	defer os.Remove(n)
	if e = f.Chmod(0600); e != nil {
		f.Close()
		return e
	}
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return replaceFile(n, path)
}
