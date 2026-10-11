// Package components contains dimension-safe terminal primitives shared by screens.
package components

import (
	"github.com/MMH-4ALL/vantage-cli/internal/tui/themes"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

func Fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	tail := ""
	if w >= 4 {
		tail = "..."
	}
	s = ansi.Truncate(s, w, tail)
	return s + strings.Repeat(" ", max(0, w-lipgloss.Width(s)))
}
func Clip(s string, w int) string {
	if w < 1 {
		return ""
	}
	return ansi.Truncate(s, w, "...")
}
func Box(t themes.Theme, title, content string, w, h int, focused bool) string {
	if w < 5 || h < 3 {
		return Block(content, w, h)
	}
	b := lipgloss.RoundedBorder()
	if t.ASCII {
		b = lipgloss.ASCIIBorder()
	}
	color := t.Border
	if focused {
		color = t.Accent
	}
	border := func(s string) string { return t.Paint(s, color) }
	label := " " + strings.ToUpper(title) + " "
	label = ansi.Truncate(label, max(0, w-4), "")
	top := border(b.TopLeft+b.Top) + t.Paint(label, color) + border(strings.Repeat(b.Top, max(0, w-3-lipgloss.Width(label)))+b.TopRight)
	lines := strings.Split(content, "\n")
	out := []string{top}
	for i := 0; i < h-2; i++ {
		s := ""
		if i < len(lines) {
			s = lines[i]
		}
		out = append(out, border(b.Left)+" "+Fit(s, w-4)+" "+border(b.Right))
	}
	out = append(out, border(b.BottomLeft+strings.Repeat(b.Bottom, w-2)+b.BottomRight))
	return strings.Join(out, "\n")
}
func Block(s string, w, h int) string {
	if h <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	out := make([]string, h)
	for i := range out {
		if i < len(lines) {
			out[i] = Fit(lines[i], w)
		} else {
			out[i] = Fit("", w)
		}
	}
	return strings.Join(out, "\n")
}
