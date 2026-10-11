package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MMH-4ALL/vantage-cli/internal/app"
	"github.com/MMH-4ALL/vantage-cli/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func testModel(t *testing.T) *Model {
	t.Helper()
	a, e := app.Open(t.TempDir(), "test")
	if e != nil {
		t.Fatal(e)
	}
	a.Settings.Animations = false
	m := New(context.Background(), a)
	t.Cleanup(func() { m.Close(); a.Close() })
	applyCommand(m, m.load())
	return m
}
func applyCommand(m *Model, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			applyCommand(m, c)
		}
		return
	}
	switch msg.(type) {
	case dataMsg, operationMsg, settingsMsg, workflowDoneMsg, formSavedMsg:
		_, next := m.Update(msg)
		applyCommand(m, next)
	}
}
func key(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }
func assertDimensions(t *testing.T, m *Model) {
	t.Helper()
	v := m.View()
	lines := strings.Split(v, "\n")
	if len(lines) != m.height {
		t.Fatalf("%s at %dx%d: got %d lines\n%s", m.screen, m.width, m.height, len(lines), v)
	}
	for i, line := range lines {
		if lipgloss.Width(line) > m.width {
			t.Fatalf("%s: line %d exceeds width: %d > %d\n%s", m.screen, i, lipgloss.Width(line), m.width, line)
		}
	}
}
func TestAllScreensAndOverlaysAtRequiredDimensions(t *testing.T) {
	m := testModel(t)
	p, e := m.a.Core.AddProject(t.TempDir(), strings.Repeat("Long project 世界 ", 10))
	if e != nil {
		t.Fatal(e)
	}
	m.a.Core.AddTask(strings.Repeat("long task ", 30), "description", p.ID)
	applyCommand(m, m.load())
	for _, size := range [][2]int{{80, 24}, {100, 30}, {120, 35}, {160, 45}, {55, 16}, {40, 8}, {1, 1}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, d := range m.a.Registry.Destinations {
			applyCommand(m, m.navigate(d.ID))
			assertDimensions(t, m)
		}
		m.palette = true
		assertDimensions(t, m)
		m.palette = false
		m.help = true
		assertDimensions(t, m)
		m.help = false
		m.openForm("workflow")
		assertDimensions(t, m)
		m.form = nil
		m.confirm = &confirmation{"Confirm", "Some real operation", func() tea.Cmd { return nil }}
		assertDimensions(t, m)
		m.confirm = nil
	}
}
func TestPaletteFocusSearchAndTextEntry(t *testing.T) {
	m := testModel(t)
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focus != 0 {
		t.Fatal("Tab did not move focus")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	if !m.palette {
		t.Fatal("palette did not open")
	}
	m.Update(key("Tasks"))
	applyCommand(m, m.key(tea.KeyMsg{Type: tea.KeyEnter}))
	if m.screen != "tasks" || m.palette {
		t.Fatal("palette destination failed")
	}
	m.key(key("/"))
	m.Update(key("q"))
	if m.ctx.Err() != nil || m.search.Value() != "q" {
		t.Fatal("typing q triggered global shortcut")
	}
	m.key(tea.KeyMsg{Type: tea.KeyEsc})
	m.openForm("task")
	m.Update(key("q"))
	if m.ctx.Err() != nil || m.form.fields[0].input.Value() != "q" {
		t.Fatal("form shortcut intercepted text")
	}
}
func TestFormSuccessAndFailureRetainInputs(t *testing.T) {
	m := testModel(t)
	m.openForm("project")
	m.form.fields[0].input.SetValue(filepath.Join(t.TempDir(), "missing"))
	applyCommand(m, m.submitForm())
	if m.form == nil || m.form.error == "" || m.form.pending {
		t.Fatal("failed form did not retain editable input")
	}
	path := t.TempDir()
	m.form.fields[0].input.SetValue(path)
	m.form.fields[1].input.SetValue("My project")
	applyCommand(m, m.submitForm())
	if m.form != nil || len(m.state.Projects) != 1 {
		t.Fatal("successful form did not register project")
	}
	m.openForm("task")
	m.form.fields[0].input.SetValue("Fix it")
	applyCommand(m, m.submitForm())
	if len(m.state.Tasks) != 1 {
		t.Fatal("task form failed")
	}
	applyCommand(m, m.navigate("tasks"))
	applyCommand(m, m.key(tea.KeyMsg{Type: tea.KeyEnter}))
	if !m.state.Tasks[0].Completed {
		t.Fatal("task toggle failed")
	}
	m.key(key("d"))
	if m.confirm == nil {
		t.Fatal("task deletion lacked confirmation")
	}
	m.key(tea.KeyMsg{Type: tea.KeyEsc})
	if len(m.state.Tasks) != 1 {
		t.Fatal("cancel deleted task")
	}
}
func TestUIWorkflowHelper(t *testing.T) {
	if os.Getenv("VANTAGE_UI_HELPER") != "1" {
		return
	}
	fmt.Println("real workflow output")
	os.Exit(0)
}
func TestWorkflowApprovalAndOutput(t *testing.T) {
	m := testModel(t)
	binary, _ := os.Executable()
	w, e := m.a.Core.AddWorkflow(domain.Workflow{Name: "Run UI helper", Executable: binary, Args: []string{"-test.run=TestUIWorkflowHelper"}, Directory: t.TempDir(), TimeoutSeconds: 10, Env: map[string]string{"VANTAGE_UI_HELPER": "1"}})
	if e != nil {
		t.Fatal(e)
	}
	applyCommand(m, m.navigate("workflows"))
	m.requestRun(w)
	if m.confirm == nil || m.running {
		t.Fatal("workflow executed before approval")
	}
	applyCommand(m, m.key(tea.KeyMsg{Type: tea.KeyEnter}))
	if m.running || m.result == nil || m.result.ExitCode != 0 || !strings.Contains(m.result.Output, "real workflow output") {
		t.Fatalf("workflow did not finish: %+v", m.result)
	}
	saved, _ := m.a.Core.Workflow(w.ID)
	if !saved.Approved {
		t.Fatal("approval was not persisted")
	}
}
func TestStaleResultsAndFailureFeedback(t *testing.T) {
	m := testModel(t)
	generation := m.generation
	m.generation++
	m.Update(dataMsg{generation: generation, err: fmt.Errorf("stale")})
	if strings.Contains(m.notice, "stale") {
		t.Fatal("stale data changed screen")
	}
	m.Update(operationMsg{err: fmt.Errorf("disk unavailable")})
	if !m.noticeError || !strings.Contains(m.View(), "disk unavailable") {
		t.Fatal("failure not visible")
	}
}
