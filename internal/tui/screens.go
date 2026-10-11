package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/MMH-4ALL/vantage-cli/internal/services/executor"
	"github.com/MMH-4ALL/vantage-cli/internal/tui/components"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) overviewView(w, h int) string {
	project := "No active project\n\nPress a to register a local directory.\nYour source files stay where they are."
	if p := m.active(); p != nil {
		project = m.theme.Bold(executor.Clean(p.Name)) + "\n" + m.label("PATH", p.Path) + "\n" + m.label("LAST OPENED", age(p.AccessedAt)) + "\n" + m.label("ID", p.ID)
	}
	git := m.theme.Paint("Choose an active project to inspect Git.", m.theme.Muted)
	if m.git.Root != "" {
		tree := "Clean working tree"
		color := m.theme.Success
		if !m.git.Clean {
			tree = fmt.Sprintf("%d changed paths", len(m.git.Changes))
			color = m.theme.Warning
		}
		git = m.label("BRANCH", m.git.Branch) + "\n" + m.theme.Paint(tree, color) + "\n\n" + m.label("RECENT COMMITS", fmt.Sprint(len(m.git.Commits)))
	} else if m.gitError != "" {
		git = m.theme.Paint(executor.Clean(m.gitError), m.theme.Warning)
	}
	topH := min(8, max(5, h/3))
	leftW := (w - 1) / 2
	rightW := w - leftW - 1
	top := lipgloss.JoinHorizontal(lipgloss.Top, components.Box(m.theme, "Active project", project, leftW, topH, false), " ", components.Box(m.theme, "Repository", git, rightW, topH, false))
	pending := 0
	for _, t := range m.state.Tasks {
		if !t.Completed {
			pending++
		}
	}
	summary := fmt.Sprintf("%d projects   /   %d workflows   /   %d pending tasks", len(m.state.Projects), len(m.state.Workflows), pending)
	compact := h-topH-3 < 16
	lines := []string{m.theme.Paint(summary, m.theme.Second)}
	if !compact {
		lines = append(lines, "", m.theme.Paint("QUICK ACTIONS", m.theme.Muted))
	}
	for i, label := range []string{"Projects   Register, search, and switch workspaces", "Git        Inspect changes and recent commits", "Workflows  Run explicit commands with live output", "Tasks      Keep your local to-do list"} {
		s := "  " + label
		if i == m.selection && m.focus == 1 {
			s = m.theme.Selected(label)
		}
		lines = append(lines, s)
	}
	recentLimit := 3
	if compact {
		recentLimit = 1
	} else {
		lines = append(lines, "")
	}
	if !compact {
		lines = append(lines, m.theme.Paint("RECENT PROJECTS", m.theme.Muted))
	}
	for _, p := range m.state.Projects[:min(recentLimit, len(m.state.Projects))] {
		lines = append(lines, components.Clip(executor.Clean(p.Name), max(10, w/3))+"  "+m.theme.Paint(executor.Clean(p.Path), m.theme.Muted))
	}
	if len(m.state.Projects) == 0 {
		lines = append(lines, "Press a to add your first project.")
	}
	if !compact {
		lines = append(lines, "")
	}
	lines = append(lines, m.label("SYSTEM", fmt.Sprintf("%s / %s / %d CPUs / %.1f MiB heap", m.info.OS, m.info.Architecture, m.info.CPUs, float64(m.info.HeapBytes)/1048576)), m.label("CWD", m.info.Directory))
	return top + "\n" + components.Box(m.theme, "Workspace overview", strings.Join(lines, "\n"), w, h-topH-1, m.focus == 1)
}

func (m *Model) projectsView(w, h int) string {
	projects := m.projects()
	lines := []string{m.searchLine(), "", m.theme.Paint(fmt.Sprintf("%-14s %-22s %s", "ID", "NAME", "PATH"), m.theme.Muted)}
	listH := h
	if w < 95 {
		listH = max(6, h/2)
	}
	start, end := listRange(len(projects), m.selection, listH-6)
	for i := start; i < end; i++ {
		p := projects[i]
		marker := " "
		if p.ID == m.state.State.ActiveProjectID {
			marker = "*"
		}
		row := fmt.Sprintf("%s %-12s %-22s %s", marker, p.ID, components.Clip(executor.Clean(p.Name), 22), executor.Clean(p.Path))
		if i == m.selection {
			row = m.theme.Selected(strings.TrimSpace(row))
		}
		lines = append(lines, row)
	}
	if len(projects) == 0 {
		lines = append(lines, "No matching projects.", "Press a to register a directory.")
	}
	detail := "Select a project to view its details."
	if p := m.selectedProject(); p != nil {
		detail = m.theme.Bold(executor.Clean(p.Name)) + "\n\n" + m.label("Path", p.Path) + "\n" + m.label("Registered", p.CreatedAt.Format("2006-01-02")) + "\n" + m.label("Last used", age(p.AccessedAt)) + "\n\n"
		if m.gitProjectID != p.ID {
			detail += m.theme.Paint("Git: press r to load this project", m.theme.Muted)
		} else if m.git.Root != "" {
			state := "clean"
			if !m.git.Clean {
				state = fmt.Sprintf("%d changes", len(m.git.Changes))
			}
			detail += m.label("Git", m.git.Branch+" / "+state)
		} else {
			detail += m.theme.Paint(executor.Clean(m.gitError), m.theme.Muted)
		}
		detail += "\n\nEnter  Make active\no      Open editor\nd      Unregister (keep files)"
	}
	if w >= 95 {
		listW := max(48, w*3/5)
		return lipgloss.JoinHorizontal(lipgloss.Top, components.Box(m.theme, fmt.Sprintf("Projects (%d)", len(projects)), strings.Join(lines, "\n"), listW, h, m.focus == 1), " ", components.Box(m.theme, "Project details", detail, w-listW-1, h, false))
	}
	return components.Box(m.theme, fmt.Sprintf("Projects (%d)", len(projects)), strings.Join(lines, "\n"), w, listH, m.focus == 1) + "\n" + components.Box(m.theme, "Project details", detail, w, h-listH-1, false)
}

func (m *Model) gitView(w, h int) string {
	if m.active() == nil {
		return components.Box(m.theme, "Git command center", "Choose an active project in Projects.\nPress 2 to open Projects; Enter activates a selection.", w, h, m.focus == 1)
	}
	if m.gitError != "" {
		return components.Box(m.theme, "Git unavailable", m.theme.Paint(executor.Clean(m.gitError), m.theme.Warning)+"\n\nPress r to retry, or 7 to check the Git executable.", w, h, true)
	}
	header := m.label("BRANCH", m.git.Branch) + "\n" + m.label("ROOT", m.git.Root)
	if m.git.Clean {
		header += "\n" + m.theme.Paint("Working tree is clean", m.theme.Success)
	} else {
		header += "\n" + m.theme.Paint(fmt.Sprintf("%d changed paths", len(m.git.Changes)), m.theme.Warning)
	}
	headerH := 6
	listH := max(4, (h-headerH-2)/2)
	lines := []string{m.theme.Paint("INDEX / WORKTREE    PATH", m.theme.Muted)}
	start, end := listRange(len(m.git.Changes), m.selection, listH-3)
	for i := start; i < end; i++ {
		c := m.git.Changes[i]
		row := fmt.Sprintf("%s%s   %s", c.Index, c.Worktree, executor.Clean(c.Path))
		if c.Original != "" {
			row += " <- " + executor.Clean(c.Original)
		}
		if i == m.selection {
			row = m.theme.Selected(row)
		}
		lines = append(lines, row)
	}
	if len(m.git.Changes) == 0 {
		lines = append(lines, "No staged, unstaged, or untracked changes.")
	}
	commits := []string{}
	start, end = listRange(len(m.git.Commits), m.selection, h-headerH-listH-5)
	for i := start; i < end; i++ {
		c := m.git.Commits[i]
		commits = append(commits, m.theme.Paint(c.Short, m.theme.Second)+"  "+executor.Clean(c.Subject)+m.theme.Paint("  / "+executor.Clean(c.Author), m.theme.Muted))
	}
	if len(commits) == 0 {
		commits = append(commits, "This repository has no commits yet.")
	}
	if len(m.git.Remotes) > 0 && len(commits) < h-headerH-listH-7 {
		commits = append(commits, "", m.theme.Paint("REMOTES", m.theme.Muted))
		commits = append(commits, m.git.Remotes...)
	}
	return components.Box(m.theme, "Repository / read only", header, w, headerH, false) + "\n" + components.Box(m.theme, "Changes", strings.Join(lines, "\n"), w, listH, m.focus == 1) + "\n" + components.Box(m.theme, "Recent commits", strings.Join(commits, "\n"), w, h-headerH-listH-2, false)
}

func (m *Model) tasksView(w, h int) string {
	tasks := m.tasks()
	scope := "all projects"
	if m.projectTasks {
		scope = "active project"
	}
	topH := max(6, h*2/3)
	lines := []string{m.searchLine(), m.theme.Paint("Filter: "+m.taskFilter+" / "+scope+"  (f / p)", m.theme.Muted), ""}
	start, end := listRange(len(tasks), m.selection, topH-6)
	for i := start; i < end; i++ {
		t := tasks[i]
		mark := "[ ]"
		if t.Completed {
			mark = "[x]"
		}
		row := mark + " " + t.Title + "  " + m.theme.Paint(t.ID, m.theme.Muted)
		if i == m.selection {
			row = m.theme.Selected(mark + " " + executor.Clean(t.Title) + "  " + t.ID)
		} else {
			row = executor.Clean(row)
		}
		lines = append(lines, row)
	}
	if len(tasks) == 0 {
		lines = append(lines, "No tasks match this filter.", "Press a to create a task, or f to change the filter.")
	}
	detail := "Select a task to see its description."
	if t := m.selectedTask(); t != nil {
		detail = m.theme.Bold(executor.Clean(t.Title)) + "\n" + executor.Clean(t.Description) + "\n" + m.label("Project", t.ProjectID) + "\n" + m.label("Created", t.CreatedAt.Format("2006-01-02"))
	}
	return components.Box(m.theme, fmt.Sprintf("Tasks (%d)", len(tasks)), strings.Join(lines, "\n"), w, topH, m.focus == 1) + "\n" + components.Box(m.theme, "Task details", detail, w, h-topH-1, false)
}

func (m *Model) workflowsView(w, h int) string {
	workflows := m.workflows()
	topH := max(7, min(12, h/2))
	listW := w
	if w >= 94 {
		listW = w / 2
	}
	lines := []string{m.searchLine(), ""}
	start, end := listRange(len(workflows), m.selection, topH-5)
	for i := start; i < end; i++ {
		wf := workflows[i]
		label := wf.Name
		approval := "review"
		if wf.Approved {
			approval = "approved"
		}
		row := "  " + executor.Clean(label) + m.theme.Paint("  ["+approval+"]", m.theme.Muted)
		if i == m.selection {
			row = m.theme.Selected(executor.Clean(label) + "  [" + approval + "]")
		}
		lines = append(lines, row)
	}
	if len(workflows) == 0 {
		lines = append(lines, "No saved workflows. Press a to create one.", "Commands run only when you request them.")
	}
	detail := ""
	if wf := m.selectedWorkflow(); wf != nil {
		detail = m.label("Executable", wf.Executable) + "\n" + m.label("Arguments", fmt.Sprintf("%q", wf.Args)) + "\n" + m.label("Directory", wf.Directory) + "\n" + m.label("Timeout", fmt.Sprintf("%ds", wf.TimeoutSeconds)) + "\n" + executor.Clean(wf.Description)
	}
	var top string
	if w >= 94 {
		top = lipgloss.JoinHorizontal(lipgloss.Top, components.Box(m.theme, "Saved workflows", strings.Join(lines, "\n"), listW, topH, m.focus == 1 && !m.outputFocus), " ", components.Box(m.theme, "Command details", detail, w-listW-1, topH, false))
	} else {
		if wf := m.selectedWorkflow(); wf != nil {
			lines = append(lines, "", m.label("Command", wf.Executable+" "+strings.Join(wf.Args, " ")))
		}
		top = components.Box(m.theme, "Saved workflows", strings.Join(lines, "\n"), w, topH, m.focus == 1 && !m.outputFocus)
	}
	state := "Output / v to focus"
	content := m.theme.Paint("Select a workflow and press Enter to run it.\nThe first run requires your approval.\n\nLarge logs retain the latest 2 MiB.\nUse PgUp / PgDown after pressing v to scroll.", m.theme.Muted)
	if m.running {
		state = "Running / " + executor.Clean(m.runName) + " / x cancel"
		content = m.output.View()
	} else if m.result != nil {
		status := "success"
		if m.result.Error != "" {
			status = "failed"
		}
		state = fmt.Sprintf("%s / exit %d / %s", status, m.result.ExitCode, m.result.Elapsed.Round(time.Millisecond))
		content = m.output.View()
	}
	return top + "\n" + components.Box(m.theme, state, content, w, h-topH-1, m.outputFocus)
}

func (m *Model) systemView(w, h int) string {
	v := m.info
	lines := []string{m.theme.Bold("Measured information. No synthetic metrics."), "", m.label("Operating system", v.OS), m.label("Architecture", v.Architecture), m.label("Logical CPUs", fmt.Sprint(v.CPUs)), m.label("Hostname", v.Hostname), "", m.label("Go heap allocated", fmt.Sprintf("%.2f MiB", float64(v.HeapBytes)/1048576)), m.label("Go runtime reserved", fmt.Sprintf("%.2f MiB", float64(v.RuntimeReservedBytes)/1048576))}
	if v.ResidentBytes != nil {
		lines = append(lines, m.label("Process resident memory", fmt.Sprintf("%.2f MiB", float64(*v.ResidentBytes)/1048576)))
	} else {
		lines = append(lines, m.theme.Paint("Process resident memory: unavailable on this platform", m.theme.Muted))
	}
	lines = append(lines, m.label("Application uptime", time.Duration(v.UptimeSeconds*float64(time.Second)).Round(time.Second).String()), m.label("Runtime", v.GoVersion), "", m.label("Current directory", v.Directory), "", m.theme.Paint(fmt.Sprintf("Refresh every %ds while this screen is visible", m.a.Settings.RefreshSeconds), m.theme.Muted))
	return components.Box(m.theme, "System monitor", strings.Join(lines, "\n"), w, h, m.focus == 1)
}
func (m *Model) settingsView(w, h int) string {
	s := m.a.Settings
	lines := []string{m.theme.Bold("Make this workspace yours."), "", m.label("Theme", s.Theme), m.label("Editor", s.Editor), m.label("Editor arguments", fmt.Sprintf("%q", s.EditorArgs)), m.label("Git executable", s.GitExecutable), m.label("Default project directory", s.DefaultProjectDirectory), m.label("Refresh interval", fmt.Sprintf("%ds", s.RefreshSeconds)), m.label("Startup screen", s.StartupScreen), m.label("Loading animation", fmt.Sprint(s.Animations)), m.label("Safe-operation confirmations", fmt.Sprint(s.ConfirmSafeOperations)), "", m.label("Configuration file", m.a.Paths.Config), m.label("Data file", m.a.Paths.Data), "", m.theme.Paint("e Edit settings   t Cycle theme   d Reset settings", m.theme.Accent), "", m.theme.Paint("Git is read only. Project removal preserves your files.", m.theme.Muted)}
	return components.Box(m.theme, "Settings", strings.Join(lines, "\n"), w, h, m.focus == 1)
}
func (m *Model) doctorView(w, h int) string {
	lines := []string{m.theme.Bold("Local environment health"), m.theme.Paint("Warnings include a suggestion; r runs the checks again.", m.theme.Muted), ""}
	start, end := listRange(len(m.checks), m.selection, h-6)
	for i := start; i < end; i++ {
		c := m.checks[i]
		color := m.theme.Success
		if c.Status == "warning" {
			color = m.theme.Warning
		}
		if c.Status == "error" {
			color = m.theme.Error
		}
		row := m.theme.Paint("["+c.Status+"]", color) + " " + executor.Clean(c.Name) + "  " + m.theme.Paint(executor.Clean(c.Detail), m.theme.Muted)
		if i == m.selection {
			row = "> " + row
		}
		lines = append(lines, row)
	}
	return components.Box(m.theme, "Doctor", strings.Join(lines, "\n"), w, h, m.focus == 1)
}
