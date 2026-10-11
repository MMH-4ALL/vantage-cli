// Package git reads machine-readable Git data without mutating a repository.
package git

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"sort"
	"strings"

	"github.com/MMH-4ALL/vantage-cli/internal/services/executor"
)

type Change struct {
	Index    string `json:"index"`
	Worktree string `json:"worktree"`
	Path     string `json:"path"`
	Original string `json:"original,omitempty"`
}
type Commit struct {
	Hash    string `json:"hash"`
	Short   string `json:"short"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}
type Status struct {
	Root    string   `json:"root"`
	Branch  string   `json:"branch"`
	Clean   bool     `json:"clean"`
	Changes []Change `json:"changes"`
	Commits []Commit `json:"commits"`
	Remotes []string `json:"remotes"`
}
type Service struct {
	Executable string
	overrides  []string
}

func (s Service) run(ctx context.Context, path string, args ...string) (string, error) {
	if _, e := exec.LookPath(s.Executable); e != nil {
		return "", errors.New("Git was not found. Install Git or set git_executable in Settings")
	}
	fixed := []string{"--no-pager", "--no-optional-locks", "-c", "core.quotepath=false", "-c", "color.ui=false", "-c", "core.fsmonitor=false", "-c", "log.showSignature=false"}
	for _, override := range s.overrides {
		fixed = append(fixed, "-c", override)
	}
	fixed = append(fixed, "-C", path)
	return executor.Capture(ctx, "", s.Executable, append(fixed, args...)...)
}
func (s Service) Inspect(ctx context.Context, path string) (Status, error) {
	root, e := s.run(ctx, path, "rev-parse", "--show-toplevel")
	if e != nil {
		return Status{}, fmt.Errorf("cannot read repository: %w", e)
	}
	// Reading key names does not execute filter commands. Disable every configured
	// driver before status can convert working-tree content through external code.
	keys, e := s.run(ctx, path, "config", "--list", "--name-only", "--null")
	if e != nil {
		return Status{}, fmt.Errorf("read Git configuration: %w", e)
	}
	drivers := map[string]bool{}
	for _, key := range strings.Split(keys, "\x00") {
		if !strings.HasPrefix(strings.ToLower(key), "filter.") {
			continue
		}
		if dot := strings.LastIndexByte(key, '.'); dot > len("filter.") {
			drivers[key[:dot]] = true
		}
	}
	for driver := range drivers {
		s.overrides = append(s.overrides, driver+".clean=", driver+".smudge=", driver+".process=", driver+".required=false")
	}
	sort.Strings(s.overrides)
	raw, e := s.run(ctx, path, "status", "--porcelain=v1", "--branch", "-z", "--untracked-files=normal", "--ignore-submodules=all")
	if e != nil {
		return Status{}, e
	}
	out, e := ParseStatus(raw)
	if e != nil {
		return out, e
	}
	out.Root = strings.TrimSpace(root)
	log, e := s.run(ctx, path, "log", "-15", "--format=%H%x00%h%x00%s%x00%an%x00%aI%x00")
	if e == nil {
		out.Commits = ParseLog(log)
	} else if !strings.HasPrefix(out.Branch, "No commits yet") {
		return out, fmt.Errorf("read commit history: %w", e)
	}
	remotes, e := s.run(ctx, path, "remote", "-v")
	if e != nil {
		return out, e
	}
	for _, r := range strings.Split(strings.TrimSpace(remotes), "\n") {
		if r != "" {
			out.Remotes = append(out.Remotes, redactRemote(r))
		}
	}
	return out, nil
}
func ParseStatus(raw string) (Status, error) {
	out := Status{Changes: []Change{}, Commits: []Commit{}, Remotes: []string{}}
	parts := strings.Split(raw, "\x00")
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "## ") {
			out.Branch = strings.TrimPrefix(p, "## ")
			continue
		}
		if len(p) < 4 || p[2] != ' ' {
			return out, errors.New("unexpected Git status format")
		}
		c := Change{Index: string(p[0]), Worktree: string(p[1]), Path: p[3:]}
		if c.Index == "R" || c.Index == "C" || c.Worktree == "R" || c.Worktree == "C" {
			i++
			if i >= len(parts) || parts[i] == "" {
				return out, errors.New("incomplete Git rename record")
			}
			c.Original = parts[i]
		}
		out.Changes = append(out.Changes, c)
	}
	out.Clean = len(out.Changes) == 0
	return out, nil
}
func ParseLog(raw string) []Commit {
	result := []Commit{}
	p := strings.Split(raw, "\x00")
	for len(p) >= 5 {
		result = append(result, Commit{Hash: strings.TrimSpace(p[0]), Short: p[1], Subject: p[2], Author: p[3], Date: p[4]})
		p = p[5:]
	}
	return result
}
func redactRemote(raw string) string {
	fields := strings.Fields(raw)
	for i, f := range fields {
		if strings.Contains(f, "://") {
			u, e := url.Parse(f)
			if e == nil {
				u.User = nil
				u.RawQuery = ""
				u.Fragment = ""
				fields[i] = u.String()
			}
		}
	}
	return executor.Clean(strings.Join(fields, " "))
}
