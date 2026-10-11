// Package tui implements a full-screen Bubble Tea client of shared application services.
package tui

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/MMH-4ALL/vantage-cli/internal/app"
	"github.com/MMH-4ALL/vantage-cli/internal/config"
	"github.com/MMH-4ALL/vantage-cli/internal/domain"
	"github.com/MMH-4ALL/vantage-cli/internal/services/executor"
	gitservice "github.com/MMH-4ALL/vantage-cli/internal/services/git"
	"github.com/MMH-4ALL/vantage-cli/internal/services/system"
	"github.com/MMH-4ALL/vantage-cli/internal/storage"
	"github.com/MMH-4ALL/vantage-cli/internal/tui/themes"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type dataMsg struct {
	generation   int
	snapshot     storage.Snapshot
	git          gitservice.Status
	gitError     string
	gitProjectID string
	system       system.Info
	checks       []app.Check
	err          error
}
type operationMsg struct {
	note string
	err  error
}
type settingsMsg struct {
	settings config.Settings
	err      error
}
type systemMsg system.Info
type refreshMsg time.Time
type outputTickMsg time.Time
type workflowDoneMsg struct{ result executor.Result }
type confirmation struct {
	title, detail string
	run           func() tea.Cmd
}

type Model struct {
	a                           *app.App
	ctx                         context.Context
	cancel                      context.CancelFunc
	loadCancel                  context.CancelFunc
	runCancel                   context.CancelFunc
	workers                     sync.WaitGroup
	width, height               int
	screen                      string
	navCursor, selection, focus int
	generation                  int
	state                       storage.Snapshot
	git                         gitservice.Status
	gitError                    string
	gitProjectID                string
	info                        system.Info
	checks                      []app.Check
	loading                     bool
	notice                      string
	noticeError                 bool
	search                      textinput.Model
	searching                   bool
	palette                     bool
	paletteInput                textinput.Model
	paletteCursor               int
	help                        bool
	helpOffset                  int
	form                        *formModel
	confirm                     *confirmation
	spin                        spinner.Model
	theme                       themes.Theme
	taskFilter                  string
	projectTasks                bool
	running                     bool
	runName                     string
	runStarted                  time.Time
	log                         *executor.Log
	output                      viewport.Model
	outputFocus                 bool
	result                      *executor.Result
	actionHandlers              map[string]func() tea.Cmd
}

func New(ctx context.Context, a *app.App) *Model {
	ctx, cancel := context.WithCancel(ctx)
	search := textinput.New()
	search.Prompt = "/ "
	search.CharLimit = 200
	palette := textinput.New()
	palette.Prompt = "> "
	palette.CharLimit = 200
	themeName := a.Settings.Theme
	if os.Getenv("NO_COLOR") != "" {
		themeName = "mono"
	}
	spin := spinner.New()
	spin.Spinner = spinner.Line
	m := &Model{a: a, ctx: ctx, cancel: cancel, width: 100, height: 30, screen: a.Settings.StartupScreen, focus: 1, search: search, paletteInput: palette, spin: spin, theme: themes.Get(themeName), taskFilter: "all", output: viewport.New(70, 10), notice: "Welcome. Press Ctrl+K to find an action."}
	if a.Warning != "" {
		m.notice = a.Warning
		m.noticeError = true
	}
	for i, d := range a.Registry.Destinations {
		if d.ID == m.screen {
			m.navCursor = i
		}
	}
	m.output.SetHorizontalStep(8)
	m.registerActions()
	return m
}

func Run(ctx context.Context, a *app.App) error {
	m := New(ctx, a)
	defer m.Close()
	_, e := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx)).Run()
	if errors.Is(e, tea.ErrProgramKilled) && ctx.Err() != nil {
		return nil
	}
	return e
}
func (m *Model) Close() {
	m.cancel()
	if m.loadCancel != nil {
		m.loadCancel()
	}
	if m.runCancel != nil {
		m.runCancel()
	}
	m.workers.Wait()
}
func (m *Model) Init() tea.Cmd { return tea.Batch(m.load(), m.refreshTick()) }

func (m *Model) load() tea.Cmd {
	if m.loadCancel != nil {
		m.loadCancel()
	}
	ctx, cancel := context.WithCancel(m.ctx)
	m.loadCancel = cancel
	m.generation++
	generation := m.generation
	m.loading = true
	a := m.a
	screen := m.screen
	gitReader := a.Git
	projectID := ""
	if screen == "projects" {
		p := m.selectedProject()
		if p != nil {
			projectID = p.ID
		}
	}
	command := func() tea.Msg {
		v, e := a.Core.Snapshot()
		out := dataMsg{generation: generation, snapshot: v, err: e, system: a.System()}
		if e != nil {
			return out
		}
		if screen == "doctor" {
			out.checks = a.Doctor(ctx)
		}
		if screen == "git" || screen == "overview" || screen == "projects" {
			ref := projectID
			if ref == "" {
				ref = v.State.ActiveProjectID
			}
			if p, e := a.Core.Project(ref); e == nil {
				out.gitProjectID = p.ID
				out.git, e = gitReader.Inspect(ctx, p.Path)
				if e != nil {
					out.gitError = e.Error()
				}
			}
		}
		return out
	}
	if m.a.Settings.Animations {
		return tea.Batch(command, m.spin.Tick)
	}
	return command
}
func (m *Model) refreshTick() tea.Cmd {
	return tea.Tick(time.Duration(m.a.Settings.RefreshSeconds)*time.Second, func(t time.Time) tea.Msg { return refreshMsg(t) })
}
func outputTick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg { return outputTickMsg(t) })
}
func (m *Model) notify(note string, err error) {
	m.notice = executor.Clean(note)
	m.noticeError = err != nil
	if err != nil {
		m.notice = executor.Clean(err.Error())
	}
}
func (m *Model) mutate(fn func() error, note string) tea.Cmd {
	return func() tea.Msg { return operationMsg{note, fn()} }
}
func (m *Model) navigate(id string) tea.Cmd {
	m.screen = id
	m.selection = 0
	m.search.SetValue("")
	m.searching = false
	m.focus = 1
	m.outputFocus = false
	for i, d := range m.a.Registry.Destinations {
		if d.ID == id {
			m.navCursor = i
		}
	}
	return m.load()
}

func (m *Model) projects() []domain.Project {
	q := strings.ToLower(m.search.Value())
	var out []domain.Project
	for _, p := range m.state.Projects {
		if strings.Contains(strings.ToLower(p.Name+" "+p.Path), q) {
			out = append(out, p)
		}
	}
	return out
}
func (m *Model) tasks() []domain.Task {
	q := strings.ToLower(m.search.Value())
	var out []domain.Task
	for _, t := range m.state.Tasks {
		if m.projectTasks && t.ProjectID != m.state.State.ActiveProjectID || m.taskFilter == "pending" && t.Completed || m.taskFilter == "completed" && !t.Completed {
			continue
		}
		if strings.Contains(strings.ToLower(t.Title+" "+t.Description), q) {
			out = append(out, t)
		}
	}
	return out
}
func (m *Model) workflows() []domain.Workflow {
	q := strings.ToLower(m.search.Value())
	var out []domain.Workflow
	for _, w := range m.state.Workflows {
		if strings.Contains(strings.ToLower(w.Name+" "+w.Description), q) {
			out = append(out, w)
		}
	}
	return out
}
func (m *Model) selectedProject() *domain.Project {
	p := m.projects()
	if len(p) == 0 {
		return nil
	}
	return &p[min(max(m.selection, 0), len(p)-1)]
}
func (m *Model) selectedTask() *domain.Task {
	v := m.tasks()
	if len(v) == 0 {
		return nil
	}
	return &v[min(max(m.selection, 0), len(v)-1)]
}
func (m *Model) selectedWorkflow() *domain.Workflow {
	v := m.workflows()
	if len(v) == 0 {
		return nil
	}
	return &v[min(max(m.selection, 0), len(v)-1)]
}
func (m *Model) itemCount() int {
	switch m.screen {
	case "projects":
		return len(m.projects())
	case "tasks":
		return len(m.tasks())
	case "workflows":
		return len(m.workflows())
	case "git":
		return max(len(m.git.Changes), len(m.git.Commits))
	case "doctor":
		return len(m.checks)
	case "overview":
		return 4
	}
	return 0
}
func (m *Model) active() *domain.Project {
	for _, p := range m.state.Projects {
		if p.ID == m.state.State.ActiveProjectID {
			return &p
		}
	}
	return nil
}

func (m *Model) registerActions() {
	m.actionHandlers = map[string]func() tea.Cmd{
		"add:project": func() tea.Cmd { return m.openForm("project") }, "add:task": func() tea.Cmd { return m.openForm("task") }, "add:workflow": func() tea.Cmd { return m.openForm("workflow") }, "edit:settings": func() tea.Cmd { return m.openForm("settings") }, "refresh": m.load, "help": func() tea.Cmd { m.help = true; return nil },
	}
	for _, a := range []app.Action{{ID: "add:project", Title: "Add a project", Description: "Register a local directory"}, {ID: "add:task", Title: "Create a task", Description: "Save a local task"}, {ID: "add:workflow", Title: "Create a workflow", Description: "Save an explicit command and arguments"}, {ID: "edit:settings", Title: "Edit settings", Description: "Change theme, editor, startup, and refresh"}, {ID: "refresh", Title: "Refresh current screen", Description: "Reload actual application state"}, {ID: "help", Title: "Keyboard help", Description: "Discover navigation and screen actions"}} {
		_ = m.a.Registry.AddAction(a)
	}
}

func (m *Model) dispatch(a app.Action) tea.Cmd {
	m.palette = false
	m.paletteInput.Blur()
	if a.Destination != "" {
		return m.navigate(a.Destination)
	}
	if fn, ok := m.actionHandlers[a.ID]; ok {
		return fn()
	}
	if a.Handler != nil {
		return m.mutate(a.Handler, a.Title+" completed")
	}
	m.notify("", errors.New("action has no registered handler"))
	return nil
}
