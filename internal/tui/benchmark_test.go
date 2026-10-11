package tui

import (
	"context"
	"testing"

	"github.com/MMH-4ALL/vantage-cli/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

func BenchmarkOverview120x35(b *testing.B) {
	a, e := app.Open(b.TempDir(), "benchmark")
	if e != nil {
		b.Fatal(e)
	}
	a.Settings.Animations = false
	m := New(context.Background(), a)
	defer a.Close()
	defer m.Close()
	applyCommand(m, m.load())
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}
