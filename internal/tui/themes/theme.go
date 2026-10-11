package themes

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name, BG, Panel, Text, Muted, Accent, Second, Success, Warning, Error, Border string
	ASCII                                                                         bool
}

func Get(name string) Theme {
	switch name {
	case "light":
		return Theme{"light", "#F5F7FB", "#FFFFFF", "#162033", "#59687D", "#0369A1", "#5B4FC7", "#087F5B", "#9A6700", "#B4233C", "#B7C5D8", false}
	case "mono":
		return Theme{Name: "mono", ASCII: true}
	default:
		return Theme{"midnight", "#0B1020", "#111827", "#E8EDF5", "#94A3B8", "#38BDF8", "#818CF8", "#34D399", "#FBBF24", "#FB7185", "#334155", false}
	}
}
func (t Theme) Paint(s, color string) string {
	if color == "" {
		return s
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(s)
}
func (t Theme) Bold(s string) string {
	if t.ASCII {
		return s
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.Text)).Render(s)
}
func (t Theme) Selected(s string) string {
	if t.ASCII {
		return "> " + s
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)).Bold(true).Render("> " + s)
}
