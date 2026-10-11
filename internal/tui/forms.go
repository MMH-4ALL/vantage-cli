package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MMH-4ALL/vantage-cli/internal/config"
	"github.com/MMH-4ALL/vantage-cli/internal/domain"
	"github.com/MMH-4ALL/vantage-cli/internal/services/executor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type formField struct {
	label string
	input textinput.Model
}
type formModel struct {
	kind, id, title string
	fields          []formField
	cursor          int
	error           string
	pending         bool
}

func (m *Model) openForm(kind string) tea.Cmd {
	f := &formModel{kind: kind}
	add := func(label, value string) {
		i := textinput.New()
		i.Prompt = "  "
		i.SetValue(value)
		i.CharLimit = 4096
		i.Width = max(10, min(65, m.width-14))
		f.fields = append(f.fields, formField{label, i})
	}
	projectID, directory := "", m.a.Settings.DefaultProjectDirectory
	if p := m.active(); p != nil {
		projectID, directory = p.ID, p.Path
	}
	if directory == "" {
		directory, _ = os.Getwd()
	}
	switch kind {
	case "project":
		f.title = "Register a project"
		add("Directory", directory)
		add("Name (blank uses directory name)", "")
	case "task":
		f.title = "Create a task"
		add("Title", "")
		add("Description", "")
		add("Project ID (optional)", projectID)
	case "edit-task":
		t := m.selectedTask()
		if t == nil {
			return nil
		}
		f.title = "Edit task"
		f.id = t.ID
		add("Title", t.Title)
		add("Description", t.Description)
	case "workflow":
		f.title = "Create a workflow"
		add("Name", "")
		add("Executable", "")
		add("Arguments as a JSON array", `[]`)
		add("Working directory", directory)
		add("Timeout in seconds", "300")
		add("Project ID (optional)", projectID)
		add("Description", "")
	case "settings":
		f.title = "Edit settings"
		s := m.a.Settings
		add("Theme: midnight / light / mono", s.Theme)
		add("Editor executable", s.Editor)
		b, _ := json.Marshal(s.EditorArgs)
		if string(b) == "null" {
			b = []byte("[]")
		}
		add("Editor arguments as a JSON array", string(b))
		add("Git executable", s.GitExecutable)
		add("Default project directory", s.DefaultProjectDirectory)
		add("Refresh interval in seconds", strconv.Itoa(s.RefreshSeconds))
		add("Startup screen", s.StartupScreen)
		add("Animate loading: true / false", strconv.FormatBool(s.Animations))
		add("Confirm safe operations: true / false", strconv.FormatBool(s.ConfirmSafeOperations))
	}
	m.form = f
	return f.fields[0].input.Focus()
}
func (m *Model) formKey(k tea.KeyMsg) tea.Cmd {
	f := m.form
	if f.pending {
		return nil
	}
	switch k.String() {
	case "esc":
		m.form = nil
		return nil
	case "ctrl+s":
		return m.submitForm()
	case "tab", "enter", "down":
		if k.String() == "enter" && f.cursor == len(f.fields)-1 {
			return m.submitForm()
		}
		f.fields[f.cursor].input.Blur()
		f.cursor = (f.cursor + 1) % len(f.fields)
		return f.fields[f.cursor].input.Focus()
	case "shift+tab", "up":
		f.fields[f.cursor].input.Blur()
		f.cursor = (f.cursor + len(f.fields) - 1) % len(f.fields)
		return f.fields[f.cursor].input.Focus()
	default:
		var cmd tea.Cmd
		f.fields[f.cursor].input, cmd = f.fields[f.cursor].input.Update(k)
		return cmd
	}
}
func (m *Model) submitForm() tea.Cmd {
	f := m.form
	value := func(i int) string { return strings.TrimSpace(f.fields[i].input.Value()) }
	var fn func() error
	switch f.kind {
	case "project":
		path, name := value(0), value(1)
		fn = func() error { _, e := m.a.Core.AddProject(path, name); return e }
	case "task":
		title, description, project := value(0), value(1), value(2)
		fn = func() error { _, e := m.a.Core.AddTask(title, description, project); return e }
	case "edit-task":
		id, title, description := f.id, value(0), value(1)
		fn = func() error { return m.a.Core.EditTask(id, title, description) }
	case "workflow":
		var args []string
		if e := json.Unmarshal([]byte(value(2)), &args); e != nil {
			f.error = "Arguments must be a JSON array, for example [\"test\", \"./...\"]."
			return nil
		}
		timeout, e := strconv.Atoi(value(4))
		if e != nil {
			f.error = "Timeout must be a whole number of seconds."
			return nil
		}
		w := domain.Workflow{Name: value(0), Executable: value(1), Args: args, Directory: value(3), TimeoutSeconds: timeout, ProjectID: value(5), Description: value(6)}
		fn = func() error { _, e := m.a.Core.AddWorkflow(w); return e }
	case "settings":
		s := m.a.Settings
		s.Theme = value(0)
		s.Editor = value(1)
		if e := json.Unmarshal([]byte(value(2)), &s.EditorArgs); e != nil {
			f.error = "Editor arguments must be a JSON array."
			return nil
		}
		s.GitExecutable = value(3)
		s.DefaultProjectDirectory = value(4)
		var e error
		s.RefreshSeconds, e = strconv.Atoi(value(5))
		if e != nil {
			f.error = "Refresh interval must be a whole number."
			return nil
		}
		s.StartupScreen = value(6)
		s.Animations, e = strconv.ParseBool(value(7))
		if e != nil {
			f.error = "Animation setting must be true or false."
			return nil
		}
		s.ConfirmSafeOperations, e = strconv.ParseBool(value(8))
		if e != nil {
			f.error = "Confirmation setting must be true or false."
			return nil
		}
		if e = config.Validate(s); e != nil {
			f.error = e.Error()
			return nil
		}
		m.form = nil
		if m.a.Warning != "" {
			m.confirm = &confirmation{"Replace invalid settings", "The saved configuration could not be loaded.\nSave your validated settings over it?", func() tea.Cmd { return m.saveSettings(s) }}
			return nil
		}
		return m.saveSettings(s)
	}
	// Validation and disk work run outside the event loop. Keep form contents until success.
	f.pending = true
	return func() tea.Msg { e := fn(); return formSavedMsg{f.kind, e} }
}

type formSavedMsg struct {
	kind string
	err  error
}

func (f *formModel) content(m *Model, w, h int) string {
	capacity := max(1, (h-6)/3)
	start := max(0, f.cursor-capacity+1)
	end := min(len(f.fields), start+capacity)
	var lines []string
	for i := start; i < end; i++ {
		field := f.fields[i]
		label := field.label
		if i == f.cursor {
			label = m.theme.Paint(label, m.theme.Accent)
		} else {
			label = m.theme.Paint(label, m.theme.Muted)
		}
		field.input.Width = max(1, w-10)
		lines = append(lines, label, field.input.View(), "")
	}
	if end < len(f.fields) {
		lines = append(lines, m.theme.Paint(fmt.Sprintf("%d more fields below; Tab to continue", len(f.fields)-end), m.theme.Muted))
	}
	if f.error != "" {
		lines = append(lines, m.theme.Paint(executor.Clean(f.error), m.theme.Error))
	}
	lines = append(lines, "", m.theme.Paint("Tab next   Shift+Tab previous   Ctrl+S save   Esc cancel", m.theme.Muted))
	return strings.Join(lines, "\n")
}
