package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MMH-4ALL/vantage-cli/internal/domain"
	"github.com/MMH-4ALL/vantage-cli/internal/storage"
)

func fixture(t *testing.T) *Service {
	t.Helper()
	s, e := storage.Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return New(s)
}
func TestProjectsSurviveUnregisterAndResolveDuplicates(t *testing.T) {
	s := fixture(t)
	path := filepath.Join(t.TempDir(), "project with spaces")
	os.Mkdir(path, 0700)
	file := filepath.Join(path, "precious.txt")
	os.WriteFile(file, []byte("keep"), 0600)
	p, e := s.AddProject(path, "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.AddProject(filepath.Join(path, "."), "other"); e == nil {
		t.Fatal("accepted duplicate directory")
	}
	if _, e = s.AddProject(filepath.Join(path, "missing"), ""); e == nil {
		t.Fatal("accepted missing directory")
	}
	if _, e = s.AddProject(file, ""); e == nil {
		t.Fatal("accepted file as project")
	}
	task, e := s.AddTask("Keep task", "details", p.ID)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.CompleteTask(task.ID, true); e != nil {
		t.Fatal(e)
	}
	if e = s.EditTask(task.ID, "Edited", "new details"); e != nil {
		t.Fatal(e)
	}
	if e = s.RemoveProject(p.ID); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(file)
	if e != nil || string(b) != "keep" {
		t.Fatal("unregister deleted project content")
	}
	v, _ := s.Snapshot()
	if len(v.Projects) != 0 || v.State.ActiveProjectID != "" || v.Tasks[0].ProjectID != "" || !v.Tasks[0].Completed || v.Tasks[0].Title != "Edited" {
		t.Fatalf("bad state: %+v", v)
	}
	if e = s.RemoveTask(task.ID); e != nil {
		t.Fatal(e)
	}
}
func TestSymlinkRegistrationAndAmbiguousNames(t *testing.T) {
	s := fixture(t)
	a, b := t.TempDir(), t.TempDir()
	p, e := s.AddProject(a, "same")
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.AddProject(b, "same")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Project("same"); e == nil {
		t.Fatal("guessed ambiguous project")
	}
	if _, e = s.Project(p.ID); e != nil {
		t.Fatal(e)
	}
	link := filepath.Join(t.TempDir(), "alias")
	if e = os.Symlink(a, link); e == nil {
		if _, e = s.AddProject(link, "alias"); e == nil {
			t.Fatal("symlink duplicate accepted")
		}
	}
}
func TestWorkflowValidationAndApproval(t *testing.T) {
	s := fixture(t)
	w := domain.Workflow{Name: "Tests", Executable: "go", Args: []string{"test", "./..."}, Directory: t.TempDir(), TimeoutSeconds: 30}
	saved, e := s.AddWorkflow(w)
	if e != nil {
		t.Fatal(e)
	}
	if saved.Approved {
		t.Fatal("new workflow preapproved")
	}
	if e = s.ApproveWorkflow(saved.ID); e != nil {
		t.Fatal(e)
	}
	got, _ := s.Workflow("Tests")
	if !got.Approved {
		t.Fatal("approval not saved")
	}
	if _, e = s.AddWorkflow(w); e == nil {
		t.Fatal("duplicate workflow name accepted")
	}
	w.TimeoutSeconds = 0
	if ValidateWorkflow(w) == nil {
		t.Fatal("unbounded workflow accepted")
	}
	w.TimeoutSeconds = 30
	w.Env = map[string]string{"API_TOKEN": "secret"}
	if ValidateWorkflow(w) == nil {
		t.Fatal("plaintext secret accepted")
	}
	w.Env = nil
	w.InheritEnv = []string{"API_TOKEN"}
	if e = ValidateWorkflow(w); e != nil {
		t.Fatal(e)
	}
	if _, e = s.AddTask(" ", "", ""); e == nil {
		t.Fatal("blank task accepted")
	}
}
