package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/MMH-4ALL/vantage-cli/internal/config"
	"github.com/MMH-4ALL/vantage-cli/internal/domain"
	"github.com/MMH-4ALL/vantage-cli/internal/services/executor"
	"github.com/MMH-4ALL/vantage-cli/internal/services/system"
	"github.com/MMH-4ALL/vantage-cli/internal/tui/themes"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(1, v.Width)
		m.height = max(1, v.Height)
		sideW := 20
		if m.width < 85 {
			sideW = 16
		}
		mainW, bodyH := m.width-sideW-1, m.height-6
		topH := max(7, min(12, bodyH/2))
		m.output.Width = max(1, mainW-4)
		m.output.Height = max(1, bodyH-topH-3)
		m.search.Width = max(1, m.width-35)
		m.paletteInput.Width = max(1, min(m.width-12, 72))
		return m, nil
	case dataMsg:
		if v.generation != m.generation {
			return m, nil
		}
		m.loading = false
		if v.err != nil {
			m.notify("", v.err)
		} else {
			m.state = v.snapshot
			m.git = v.git
			m.gitError = v.gitError
			m.gitProjectID = v.gitProjectID
			m.info = v.system
			m.checks = v.checks
			m.selection = min(m.selection, max(0, m.itemCount()-1))
		}
		return m, nil
	case operationMsg:
		m.notify(v.note, v.err)
		if v.err == nil {
			return m, m.load()
		}
		return m, nil
	case settingsMsg:
		if v.err != nil {
			m.notify("", v.err)
			return m, nil
		}
		m.a.ApplySettings(v.settings)
		name := v.settings.Theme
		if os.Getenv("NO_COLOR") != "" {
			name = "mono"
		}
		m.theme = themes.Get(name)
		m.notify("Settings saved", nil)
		return m, m.load()
	case formSavedMsg:
		if v.err != nil {
			if m.form != nil {
				m.form.error = v.err.Error()
				m.form.pending = false
			}
			m.notify("", v.err)
			return m, nil
		}
		m.form = nil
		m.notify("Saved successfully", nil)
		return m, m.load()
	case systemMsg:
		m.info = systemInfo(v)
		return m, nil
	case refreshMsg:
		cmd := m.refreshTick()
		if m.screen == "system" || m.screen == "overview" {
			a := m.a
			return m, tea.Batch(cmd, func() tea.Msg { return systemMsg(a.System()) })
		}
		return m, cmd
	case spinner.TickMsg:
		if m.loading || m.running {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
		return m, nil
	case outputTickMsg:
		if !m.running {
			return m, nil
		}
		atBottom := m.output.AtBottom()
		m.output.SetContent(m.log.Text())
		if atBottom {
			m.output.GotoBottom()
		}
		return m, outputTick()
	case workflowDoneMsg:
		m.running = false
		m.outputFocus = false
		m.result = &v.result
		m.output.SetContent(v.result.Output)
		m.output.GotoBottom()
		if v.result.Error != "" {
			m.notify("", fmt.Errorf("%s: %s (exit %d)", m.runName, v.result.Error, v.result.ExitCode))
		} else {
			m.notify(fmt.Sprintf("%s finished in %s", m.runName, v.result.Elapsed.Round(time.Millisecond)), nil)
		}
		return m, m.load()
	case tea.KeyMsg:
		return m, m.key(v)
	}
	return m, nil
}

// systemInfo avoids storing application state in widgets.
func systemInfo(v systemMsg) system.Info { return system.Info(v) }

func (m *Model) key(k tea.KeyMsg) tea.Cmd {
	key := k.String()
	if key == "ctrl+c" {
		m.cancel()
		if m.runCancel != nil {
			m.runCancel()
		}
		return tea.Quit
	}
	if m.confirm != nil {
		switch key {
		case "esc", "n":
			m.confirm = nil
		case "y", "enter":
			run := m.confirm.run
			m.confirm = nil
			return run()
		}
		return nil
	}
	if m.form != nil {
		return m.formKey(k)
	}
	if m.help {
		if key == "up" || key == "k" {
			m.helpOffset = max(0, m.helpOffset-1)
		}
		if key == "down" || key == "j" {
			m.helpOffset = min(max(0, len(strings.Split(m.helpText(), "\n"))-max(1, m.height-9)), m.helpOffset+1)
		}
		if key == "pgdown" {
			m.helpOffset = min(max(0, len(strings.Split(m.helpText(), "\n"))-max(1, m.height-9)), m.helpOffset+8)
		}
		if key == "pgup" {
			m.helpOffset = max(0, m.helpOffset-8)
		}
		if key == "esc" || key == "?" || key == "q" {
			m.help = false
		}
		return nil
	}
	if m.palette {
		switch key {
		case "esc":
			m.palette = false
			m.paletteInput.Blur()
			return nil
		case "up", "ctrl+p":
			m.paletteCursor = max(0, m.paletteCursor-1)
			return nil
		case "down", "ctrl+n":
			m.paletteCursor = min(max(0, len(m.a.Registry.FilterActions(m.paletteInput.Value()))-1), m.paletteCursor+1)
			return nil
		case "enter":
			items := m.a.Registry.FilterActions(m.paletteInput.Value())
			if len(items) > 0 {
				return m.dispatch(items[min(m.paletteCursor, len(items)-1)])
			}
			return nil
		default:
			var cmd tea.Cmd
			m.paletteInput, cmd = m.paletteInput.Update(k)
			m.paletteCursor = 0
			return cmd
		}
	}
	if m.searching {
		switch key {
		case "esc", "enter":
			m.searching = false
			m.search.Blur()
			return nil
		default:
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(k)
			m.selection = 0
			if m.screen == "projects" {
				return tea.Batch(cmd, m.load())
			}
			return cmd
		}
	}
	switch key {
	case "q":
		m.cancel()
		if m.runCancel != nil {
			m.runCancel()
		}
		return tea.Quit
	case "ctrl+k":
		m.palette = true
		m.paletteCursor = 0
		m.paletteInput.SetValue("")
		return m.paletteInput.Focus()
	case "?":
		m.help = true
		m.helpOffset = 0
		return nil
	case "tab", "shift+tab":
		m.focus = 1 - m.focus
		return nil
	case "left", "h":
		if m.outputFocus && m.screen == "workflows" {
			m.output.ScrollLeft(8)
			return nil
		}
		m.focus = 0
		return nil
	case "right", "l":
		if m.outputFocus && m.screen == "workflows" {
			m.output.ScrollRight(8)
			return nil
		}
		m.focus = 1
		return nil
	case "esc":
		m.search.SetValue("")
		m.outputFocus = false
		m.focus = 0
		return nil
	case "/":
		m.searching = true
		m.focus = 1
		return m.search.Focus()
	case "r":
		return m.load()
	case "1", "2", "3", "4", "5", "6", "7", "8":
		i := int(key[0] - '1')
		if i < len(m.a.Registry.Destinations) {
			return m.navigate(m.a.Registry.Destinations[i].ID)
		}
	case "x":
		if m.running && m.runCancel != nil {
			m.runCancel()
			m.notify("Canceling workflow...", nil)
		}
		return nil
	case "v":
		if m.screen == "workflows" {
			m.outputFocus = !m.outputFocus
		}
		return nil
	}
	if m.outputFocus && m.screen == "workflows" {
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(k)
		return cmd
	}
	if key == "up" || key == "k" || key == "down" || key == "j" {
		delta := 1
		if key == "up" || key == "k" {
			delta = -1
		}
		if m.focus == 0 {
			m.navCursor = min(max(m.navCursor+delta, 0), len(m.a.Registry.Destinations)-1)
			return nil
		}
		m.selection = min(max(m.selection+delta, 0), max(0, m.itemCount()-1))
		if m.screen == "projects" {
			return m.load()
		}
		return nil
	}
	if key == "enter" && m.focus == 0 {
		return m.navigate(m.a.Registry.Destinations[m.navCursor].ID)
	}
	if key == "enter" {
		switch m.screen {
		case "overview":
			return m.navigate([]string{"projects", "git", "workflows", "tasks"}[m.selection%4])
		case "projects":
			if p := m.selectedProject(); p != nil {
				id := p.ID
				return m.mutate(func() error { return m.a.Core.Activate(id) }, "Active project updated")
			}
		case "tasks":
			if t := m.selectedTask(); t != nil {
				id, done := t.ID, !t.Completed
				return m.mutate(func() error { return m.a.Core.CompleteTask(id, done) }, "Task updated")
			}
		case "workflows":
			if w := m.selectedWorkflow(); w != nil {
				return m.requestRun(*w)
			}
		case "settings":
			return m.openForm("settings")
		}
	}
	if key == "a" {
		switch m.screen {
		case "overview", "projects":
			return m.openForm("project")
		case "tasks":
			return m.openForm("task")
		case "workflows":
			return m.openForm("workflow")
		}
	}
	if key == "e" {
		if m.screen == "tasks" && m.selectedTask() != nil {
			return m.openForm("edit-task")
		}
		if m.screen == "settings" {
			return m.openForm("settings")
		}
	}
	if key == "d" && m.screen != "settings" {
		switch m.screen {
		case "projects":
			if p := m.selectedProject(); p != nil {
				id := p.ID
				m.confirm = &confirmation{"Unregister project", p.Name + "\nOnly the registration will be removed. Your files stay in place.", func() tea.Cmd {
					return m.mutate(func() error { return m.a.Core.RemoveProject(id) }, "Project unregistered")
				}}
			}
		case "tasks":
			if t := m.selectedTask(); t != nil {
				id := t.ID
				m.confirm = &confirmation{"Delete task", t.Title + "\nThis removes its saved task record.", func() tea.Cmd { return m.mutate(func() error { return m.a.Core.RemoveTask(id) }, "Task removed") }}
			}
		case "workflows":
			if w := m.selectedWorkflow(); w != nil {
				id := w.ID
				m.confirm = &confirmation{"Delete workflow", w.Name + "\nThis removes the saved command.", func() tea.Cmd {
					return m.mutate(func() error { return m.a.Core.RemoveWorkflow(id) }, "Workflow removed")
				}}
			}
		}
		return nil
	}
	if key == "o" && m.screen == "projects" {
		if p := m.selectedProject(); p != nil {
			args := append(append([]string{}, m.a.Settings.EditorArgs...), p.Path)
			cmd := exec.CommandContext(m.ctx, m.a.Settings.Editor, args...)
			cmd.Dir = p.Path
			open := func() tea.Cmd {
				return tea.ExecProcess(cmd, func(e error) tea.Msg { return operationMsg{"Editor closed", e} })
			}
			if m.a.Settings.ConfirmSafeOperations {
				m.confirm = &confirmation{"Open editor", m.a.Settings.Editor + "\n" + p.Path, open}
				return nil
			}
			return open()
		}
	}
	if key == "f" && m.screen == "tasks" {
		switch m.taskFilter {
		case "all":
			m.taskFilter = "pending"
		case "pending":
			m.taskFilter = "completed"
		default:
			m.taskFilter = "all"
		}
		m.selection = 0
	}
	if key == "p" && m.screen == "tasks" {
		m.projectTasks = !m.projectTasks
		m.selection = 0
	}
	if key == "t" && m.screen == "settings" {
		s := m.a.Settings
		switch s.Theme {
		case "midnight":
			s.Theme = "light"
		case "light":
			s.Theme = "mono"
		default:
			s.Theme = "midnight"
		}
		if m.a.Warning != "" {
			m.confirm = &confirmation{"Replace invalid settings", "Save validated settings over the invalid configuration?", func() tea.Cmd { return m.saveSettings(s) }}
			return nil
		}
		return m.saveSettings(s)
	}
	if key == "d" && m.screen == "settings" {
		m.confirm = &confirmation{"Reset settings", "Replace your settings with defaults?\nProjects, tasks, and workflows will be preserved.", func() tea.Cmd { return m.saveSettings(config.Defaults()) }}
	}
	return nil
}

func (m *Model) saveSettings(s config.Settings) tea.Cmd {
	path := m.a.Paths.Config
	return func() tea.Msg { return settingsMsg{s, config.Save(path, s)} }
}
func (m *Model) requestRun(w domain.Workflow) tea.Cmd {
	if m.running {
		m.notify("A workflow is already running. Press x to cancel it.", nil)
		return nil
	}
	if !w.Approved {
		m.confirm = &confirmation{"Approve first execution", fmt.Sprintf("%s\nExecutable: %s\nArguments: %q\nDirectory: %s\nTimeout: %ds\nRuns with your normal permissions.", w.Name, w.Executable, w.Args, w.Directory, w.TimeoutSeconds), func() tea.Cmd { return m.startWorkflow(w) }}
		return nil
	}
	return m.startWorkflow(w)
}
func (m *Model) startWorkflow(w domain.Workflow) tea.Cmd {
	ctx, cancel := context.WithCancel(m.ctx)
	m.runCancel = cancel
	m.running = true
	m.runName = w.Name
	m.runStarted = time.Now()
	m.result = nil
	m.log = executor.NewLog()
	m.output.SetContent("")
	m.outputFocus = true
	m.screen = "workflows"
	log := m.log
	a := m.a
	done := make(chan workflowDoneMsg, 1)
	m.workers.Add(1)
	go func() {
		defer m.workers.Done()
		if e := a.Core.ApproveWorkflow(w.ID); e != nil {
			done <- workflowDoneMsg{executor.Result{ExitCode: -1, Error: e.Error()}}
			return
		}
		done <- workflowDoneMsg{executor.Run(ctx, w, log)}
	}()
	run := func() tea.Msg { return <-done }
	if m.a.Settings.Animations {
		return tea.Batch(run, outputTick(), m.spin.Tick)
	}
	return tea.Batch(run, outputTick())
}
