package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/MMH-4ALL/vantage-cli/internal/services/executor"
	"github.com/MMH-4ALL/vantage-cli/internal/tui/components"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	w, h := m.width, m.height
	if w < 55 || h < 16 {
		return components.Block("VANTAGE\n\nTerminal is too small for the dashboard.\nUse at least 55 columns and 16 rows.\nResize the window, or press q to exit.\nCLI commands work at any size.", w, h)
	}
	if m.form != nil {
		return m.modal(m.form.title, m.form.content(m, min(82, w-8), h-6), min(82, w-8), h-6)
	}
	if m.confirm != nil {
		return m.modal(m.confirm.title, executor.Clean(m.confirm.detail)+"\n\n"+m.theme.Paint("Y / Enter approve     N / Esc cancel", m.theme.Warning), min(86, w-8), min(16, h-6))
	}
	if m.palette {
		return m.paletteView()
	}
	if m.help {
		lines := strings.Split(m.helpText(), "\n")
		capacity := max(1, h-9)
		start := min(m.helpOffset, max(0, len(lines)-capacity))
		end := min(len(lines), start+capacity)
		content := m.theme.Paint("Up/Down scroll   PgUp/PgDown page   Esc close", m.theme.Accent) + "\n\n" + strings.Join(lines[start:end], "\n")
		return m.modal("Keyboard & screen actions", content, min(100, w-6), h-4)
	}
	active := "No active project"
	if p := m.active(); p != nil {
		active = executor.Clean(p.Name)
	}
	brand := m.theme.Paint(" VANTAGE", m.theme.Accent) + m.theme.Paint("  /  DEVELOPER COMMAND CENTER", m.theme.Muted)
	rule := "─"
	if m.theme.ASCII {
		rule = "-"
	}
	header := components.Fit(brand, w) + "\n" + components.Fit(m.theme.Paint(" Workspace: ", m.theme.Muted)+m.theme.Bold(active)+m.theme.Paint("  |  "+m.a.Version+"  |  LOCAL", m.theme.Muted), w) + "\n" + m.theme.Paint(strings.Repeat(rule, w), m.theme.Border)
	bodyH := h - 6
	sideW := 20
	if w < 85 {
		sideW = 16
	}
	mainW := w - sideW - 1
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.sidebar(sideW, bodyH), " ", m.screenView(mainW, bodyH))
	status := m.notice
	if m.loading {
		state := "Loading"
		if m.a.Settings.Animations {
			state = m.spin.View() + " Loading"
		}
		status = state + "  |  " + status
	}
	if m.running {
		status = "Running " + m.runName + "  |  x cancel  |  " + status
	}
	statusColor := m.theme.Muted
	if m.noticeError {
		statusColor = m.theme.Error
	}
	footer := components.Fit(m.theme.Paint(" "+executor.Clean(status), statusColor), w) + "\n" + components.Fit(m.theme.Paint(" "+m.contextHelp(), m.theme.Muted), w) + "\n" + components.Fit(m.theme.Paint(" Tab panels  |  / search  |  Ctrl+K commands  |  ? help  |  q exit", m.theme.Muted), w)
	view := header + "\n" + body + "\n" + footer
	if m.theme.BG != "" {
		view = lipgloss.NewStyle().Background(lipgloss.Color(m.theme.BG)).Foreground(lipgloss.Color(m.theme.Text)).Render(view)
	}
	return view
}

func (m *Model) sidebar(w, h int) string {
	lines := []string{m.theme.Paint("WORKSPACE", m.theme.Muted), ""}
	for i, d := range m.a.Registry.Destinations {
		title := fmt.Sprintf("%d %s", i+1, d.Title)
		if w < 20 {
			if i == m.navCursor && m.focus == 0 {
				title = m.theme.Selected(d.Title)
			} else if d.ID == m.screen {
				title = m.theme.Paint("*"+title, m.theme.Accent)
			} else {
				title = " " + title
			}
		} else if i == m.navCursor && m.focus == 0 {
			title = m.theme.Selected(title)
		} else if d.ID == m.screen {
			title = m.theme.Paint("* "+title, m.theme.Accent)
		} else {
			title = "  " + title
		}
		lines = append(lines, title)
		if h >= 23 {
			lines = append(lines, "")
		}
	}
	for len(lines) < h-7 {
		lines = append(lines, "")
	}
	lines = append(lines, m.theme.Paint("LOCAL DATA", m.theme.Muted), m.theme.Paint("OFFLINE", m.theme.Muted))
	return components.Box(m.theme, "Navigation", strings.Join(lines, "\n"), w, h, m.focus == 0)
}

func (m *Model) screenView(w, h int) string {
	switch m.screen {
	case "overview":
		return m.overviewView(w, h)
	case "projects":
		return m.projectsView(w, h)
	case "git":
		return m.gitView(w, h)
	case "workflows":
		return m.workflowsView(w, h)
	case "tasks":
		return m.tasksView(w, h)
	case "system":
		return m.systemView(w, h)
	case "settings":
		return m.settingsView(w, h)
	case "doctor":
		return m.doctorView(w, h)
	}
	return components.Box(m.theme, "Unknown screen", "Use Ctrl+K to choose a registered destination.", w, h, true)
}

func (m *Model) modal(title, content string, w, h int) string {
	w = min(w, m.width-2)
	h = min(h, m.height-2)
	box := components.Box(m.theme, title, content, w, h, true)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
func (m *Model) paletteView() string {
	w := min(88, m.width-6)
	h := min(23, m.height-4)
	items := m.a.Registry.FilterActions(m.paletteInput.Value())
	lines := []string{m.paletteInput.View(), m.theme.Paint("Find screens and actions", m.theme.Muted), ""}
	capacity := max(1, (h-7)/2)
	start := max(0, m.paletteCursor-capacity+1)
	for i := start; i < min(len(items), start+capacity); i++ {
		a := items[i]
		title := "  " + a.Title
		if i == m.paletteCursor {
			title = m.theme.Selected(a.Title)
		}
		lines = append(lines, title, m.theme.Paint("    "+a.Description, m.theme.Muted))
	}
	if len(items) == 0 {
		lines = append(lines, "No matching actions. Try a different search.")
	}
	lines = append(lines, "", m.theme.Paint("Up / Down select   Enter run   Esc close", m.theme.Muted))
	return m.modal("Command palette", strings.Join(lines, "\n"), w, h)
}
func (m *Model) helpText() string {
	return `NAVIGATION
  1-8              Jump to a screen
  Up/Down or j/k   Move through a list
  Tab / Shift+Tab  Switch sidebar and content focus
  Enter            Activate selection
  /                Filter the current list; Enter finishes typing
  Escape           Close overlay or clear search
  Ctrl+K           Search commands and destinations
  r                Refresh from disk and Git
  q / Ctrl+C       Exit; cancel running operations

SCREEN ACTIONS
  Projects         a add | Enter activate | o editor | d unregister
  Tasks            a add | e edit | Enter complete/reopen | d delete
                   f pending/completed/all | p active project/all
  Workflows        a add | Enter run | x cancel | v output focus
                   PgUp/PgDown scroll output | d delete
  Settings         e edit | t cycle themes | d reset

FORMS
  Tab / Shift+Tab  Next / previous field
  Ctrl+S           Save; Enter on last field also saves
  Escape           Cancel without saving

Workflows never run on launch. Git screens never change your repository.`
}
func (m *Model) contextHelp() string {
	switch m.screen {
	case "projects":
		return "a add  |  Enter activate  |  o editor  |  d unregister"
	case "tasks":
		return "a add  |  e edit  |  Enter toggle done  |  f filter  |  p project  |  d delete"
	case "workflows":
		return "a add  |  Enter run  |  x cancel  |  v output focus  |  PgUp/PgDn scroll"
	case "settings":
		return "e edit  |  t theme  |  d reset"
	case "overview":
		return "Enter open selected quick action  |  a add project"
	default:
		return "Up/Down navigate  |  r refresh"
	}
}
func (m *Model) label(label, value string) string {
	return m.theme.Paint(label+"  ", m.theme.Muted) + executor.Clean(value)
}
func (m *Model) searchLine() string {
	if m.searching {
		return m.search.View()
	}
	if m.search.Value() != "" {
		return m.theme.Paint("Filter: "+executor.Clean(m.search.Value()), m.theme.Accent)
	}
	return m.theme.Paint("Press / to search", m.theme.Muted)
}
func listRange(total, selected, capacity int) (int, int) {
	capacity = max(1, capacity)
	start := max(0, min(selected-capacity+1, max(0, total-capacity)))
	return start, min(total, start+capacity)
}
func age(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return t.Format("Jan 02")
}
