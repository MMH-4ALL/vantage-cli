package storage

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/MMH-4ALL/vantage-cli/internal/domain"
)

func TestPersistenceRollbackAndConcurrentUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := s.Update(func(v *Snapshot) error { v.Tasks = append(v.Tasks, domain.Task{Title: "saved"}); return nil }); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	e = s.Update(func(v *Snapshot) error { v.Tasks = nil; return errors.New("rollback") })
	if e == nil {
		t.Fatal("rollback error lost")
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	v, e := s.Read()
	if e != nil || len(v.Tasks) != 20 {
		t.Fatalf("lost persisted data: %d %v", len(v.Tasks), e)
	}
}
func TestTwoHandlesCoexistWithoutLostUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	first, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer first.Close()
	second, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer second.Close()
	var wg sync.WaitGroup
	for _, store := range []*Store{first, second} {
		store := store
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				if e := store.Update(func(v *Snapshot) error { v.Tasks = append(v.Tasks, domain.Task{Title: "concurrent"}); return nil }); e != nil {
					t.Error(e)
				}
			}
		}()
	}
	wg.Wait()
	v, e := first.Read()
	if e != nil || len(v.Tasks) != 20 {
		t.Fatalf("lost updates: %d %v", len(v.Tasks), e)
	}
}
