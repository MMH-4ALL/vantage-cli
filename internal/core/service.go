// Package core owns project, task, and workflow rules shared by the CLI and TUI.
package core

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/MMH-4ALL/vantage-cli/internal/domain"
	"github.com/MMH-4ALL/vantage-cli/internal/storage"
)

type Service struct{ Store storage.Repository }

func New(s storage.Repository) *Service { return &Service{Store: s} }
func ID() (string, error) {
	var b [6]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	return hex.EncodeToString(b[:]), nil
}
func (s *Service) Snapshot() (storage.Snapshot, error) {
	v, e := s.Store.Read()
	sort.Slice(v.Projects, func(i, j int) bool { return v.Projects[i].AccessedAt.After(v.Projects[j].AccessedAt) })
	return v, e
}

func Directory(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, e := os.UserHomeDir()
		if e != nil {
			return "", e
		}
		path = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~"), string(filepath.Separator)))
	}
	a, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	a, e = filepath.EvalSymlinks(a)
	if e != nil {
		return "", fmt.Errorf("directory is unavailable: %w", e)
	}
	i, e := os.Stat(a)
	if e != nil {
		return "", e
	}
	if !i.IsDir() {
		return "", errors.New("path must be a directory")
	}
	return filepath.Clean(a), nil
}

func samePath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
func FindProject(v storage.Snapshot, ref string) (domain.Project, error) {
	if ref == "" {
		ref = v.State.ActiveProjectID
	}
	var matches []domain.Project
	for _, p := range v.Projects {
		if p.ID == ref || samePath(p.Path, ref) {
			return p, nil
		}
		if strings.EqualFold(p.Name, ref) {
			matches = append(matches, p)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return domain.Project{}, errors.New("project name is ambiguous; use its ID")
	}
	return domain.Project{}, errors.New("project not found; register it with vantage project add or select an active project")
}
func (s *Service) Project(ref string) (domain.Project, error) {
	v, e := s.Snapshot()
	if e != nil {
		return domain.Project{}, e
	}
	return FindProject(v, ref)
}

func (s *Service) AddProject(path, name string) (domain.Project, error) {
	a, e := Directory(path)
	if e != nil {
		return domain.Project{}, e
	}
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(a)
	}
	id, e := ID()
	if e != nil {
		return domain.Project{}, e
	}
	p := domain.Project{ID: id, Name: strings.TrimSpace(name), Path: a, CreatedAt: time.Now(), AccessedAt: time.Now()}
	e = s.Store.Update(func(v *storage.Snapshot) error {
		for _, x := range v.Projects {
			if samePath(x.Path, a) {
				return fmt.Errorf("already registered as %s (%s)", x.Name, x.ID)
			}
		}
		v.Projects = append(v.Projects, p)
		v.State.ActiveProjectID = id
		return nil
	})
	return p, e
}
func (s *Service) Activate(ref string) error {
	return s.Store.Update(func(v *storage.Snapshot) error {
		p, e := FindProject(*v, ref)
		if e != nil {
			return e
		}
		if _, e = Directory(p.Path); e != nil {
			return e
		}
		v.State.ActiveProjectID = p.ID
		for i := range v.Projects {
			if v.Projects[i].ID == p.ID {
				v.Projects[i].AccessedAt = time.Now()
			}
		}
		return nil
	})
}
func (s *Service) RemoveProject(ref string) error {
	return s.Store.Update(func(v *storage.Snapshot) error {
		p, e := FindProject(*v, ref)
		if e != nil {
			return e
		}
		for i, x := range v.Projects {
			if x.ID == p.ID {
				v.Projects = append(v.Projects[:i], v.Projects[i+1:]...)
				break
			}
		}
		if v.State.ActiveProjectID == p.ID {
			v.State.ActiveProjectID = ""
		}
		for i := range v.Tasks {
			if v.Tasks[i].ProjectID == p.ID {
				v.Tasks[i].ProjectID = ""
			}
		}
		for i := range v.Workflows {
			if v.Workflows[i].ProjectID == p.ID {
				v.Workflows[i].ProjectID = ""
			}
		}
		return nil
	})
}

func (s *Service) AddTask(title, description, projectRef string) (domain.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Task{}, errors.New("task title cannot be empty")
	}
	id, e := ID()
	if e != nil {
		return domain.Task{}, e
	}
	t := domain.Task{ID: id, Title: title, Description: description, CreatedAt: time.Now()}
	e = s.Store.Update(func(v *storage.Snapshot) error {
		if projectRef != "" {
			p, e := FindProject(*v, projectRef)
			if e != nil {
				return e
			}
			t.ProjectID = p.ID
		}
		v.Tasks = append(v.Tasks, t)
		return nil
	})
	return t, e
}
func (s *Service) EditTask(id, title, description string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("task title cannot be empty")
	}
	return s.mutateTask(id, func(t *domain.Task) { t.Title = strings.TrimSpace(title); t.Description = description })
}
func (s *Service) CompleteTask(id string, done bool) error {
	return s.mutateTask(id, func(t *domain.Task) { t.Completed = done })
}
func (s *Service) mutateTask(id string, fn func(*domain.Task)) error {
	return s.Store.Update(func(v *storage.Snapshot) error {
		for i := range v.Tasks {
			if v.Tasks[i].ID == id {
				fn(&v.Tasks[i])
				return nil
			}
		}
		return errors.New("task not found")
	})
}
func (s *Service) RemoveTask(id string) error {
	return s.Store.Update(func(v *storage.Snapshot) error {
		for i, t := range v.Tasks {
			if t.ID == id {
				v.Tasks = append(v.Tasks[:i], v.Tasks[i+1:]...)
				return nil
			}
		}
		return errors.New("task not found")
	})
}

func SensitiveKey(k string) bool {
	k = strings.ToUpper(k)
	for _, s := range []string{"TOKEN", "SECRET", "PASSWORD", "PASSWD", "API_KEY", "PRIVATE_KEY", "CREDENTIAL"} {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}
func ValidateWorkflow(w domain.Workflow) error {
	if strings.TrimSpace(w.Name) == "" {
		return errors.New("workflow name cannot be empty")
	}
	if strings.TrimSpace(w.Executable) == "" || strings.IndexByte(w.Executable, 0) >= 0 {
		return errors.New("provide an executable path or name")
	}
	for _, a := range w.Args {
		if strings.IndexByte(a, 0) >= 0 {
			return errors.New("arguments cannot contain NUL characters")
		}
	}
	if w.TimeoutSeconds < 1 || w.TimeoutSeconds > 86400 {
		return errors.New("timeout must be between 1 second and 24 hours")
	}
	if _, e := Directory(w.Directory); e != nil {
		return fmt.Errorf("workflow working directory: %w", e)
	}
	for k, v := range w.Env {
		if k == "" || strings.ContainsAny(k, "=\x00") || strings.ContainsRune(v, 0) {
			return errors.New("invalid environment variable")
		}
		if SensitiveKey(k) {
			return fmt.Errorf("%s looks secret; use --inherit-env %s instead of storing a value", k, k)
		}
	}
	for _, k := range w.InheritEnv {
		if k == "" || strings.ContainsAny(k, "=\x00") {
			return errors.New("invalid inherited environment name")
		}
	}
	return nil
}
func (s *Service) AddWorkflow(w domain.Workflow) (domain.Workflow, error) {
	if e := ValidateWorkflow(w); e != nil {
		return w, e
	}
	a, e := Directory(w.Directory)
	if e != nil {
		return w, e
	}
	w.Directory = a
	id, e := ID()
	if e != nil {
		return w, e
	}
	w.ID = id
	w.Approved = false
	w.CreatedAt = time.Now()
	e = s.Store.Update(func(v *storage.Snapshot) error {
		for _, x := range v.Workflows {
			if strings.EqualFold(x.Name, w.Name) {
				return errors.New("workflow name already exists")
			}
		}
		if w.ProjectID != "" {
			p, e := FindProject(*v, w.ProjectID)
			if e != nil {
				return e
			}
			w.ProjectID = p.ID
		}
		v.Workflows = append(v.Workflows, w)
		return nil
	})
	return w, e
}
func (s *Service) Workflow(ref string) (domain.Workflow, error) {
	v, e := s.Snapshot()
	if e != nil {
		return domain.Workflow{}, e
	}
	for _, w := range v.Workflows {
		if w.ID == ref || strings.EqualFold(w.Name, ref) {
			return w, nil
		}
	}
	return domain.Workflow{}, errors.New("workflow not found")
}
func (s *Service) ApproveWorkflow(id string) error {
	return s.Store.Update(func(v *storage.Snapshot) error {
		for i := range v.Workflows {
			if v.Workflows[i].ID == id {
				v.Workflows[i].Approved = true
				return nil
			}
		}
		return errors.New("workflow not found")
	})
}
func (s *Service) RemoveWorkflow(ref string) error {
	return s.Store.Update(func(v *storage.Snapshot) error {
		for i, w := range v.Workflows {
			if w.ID == ref || strings.EqualFold(w.Name, ref) {
				v.Workflows = append(v.Workflows[:i], v.Workflows[i+1:]...)
				return nil
			}
		}
		return errors.New("workflow not found")
	})
}
