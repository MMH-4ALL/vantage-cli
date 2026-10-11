package core

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/MMH-4ALL/vantage-cli/internal/domain"
	"github.com/MMH-4ALL/vantage-cli/internal/storage"
)

func BenchmarkSnapshot500Projects(b *testing.B) {
	store, e := storage.Open(filepath.Join(b.TempDir(), "data.db"))
	if e != nil {
		b.Fatal(e)
	}
	defer store.Close()
	service := New(store)
	e = store.Update(func(v *storage.Snapshot) error {
		for i := 0; i < 500; i++ {
			v.Projects = append(v.Projects, domain.Project{ID: fmt.Sprint(i), Name: fmt.Sprintf("Project %d", i), Path: fmt.Sprintf("/bench/project-%d", i)})
		}
		return nil
	})
	if e != nil {
		b.Fatal(e)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, e = service.Snapshot(); e != nil {
			b.Fatal(e)
		}
	}
}
