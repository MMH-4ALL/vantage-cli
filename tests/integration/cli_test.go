package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MMH-4ALL/vantage-cli/internal/cli"
	"github.com/MMH-4ALL/vantage-cli/internal/domain"
)

func run(t *testing.T, home string, args ...string) (string, error) {
	t.Helper()
	c := cli.New("test", "test-commit", "today")
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	c.SetOut(out)
	c.SetErr(errOut)
	c.SetArgs(append([]string{"--home", home}, args...))
	e := c.Execute()
	return out.String(), e
}
func TestCLIPersistenceJSONHelpAndSafeRemoval(t *testing.T) {
	home, path := t.TempDir(), t.TempDir()
	if _, e := run(t, home, "project", "add", path, "--name", "Example"); e != nil {
		t.Fatal(e)
	}
	out, e := run(t, home, "project", "list", "--json")
	if e != nil {
		t.Fatal(e)
	}
	var projects []domain.Project
	if e = json.Unmarshal([]byte(out), &projects); e != nil || len(projects) != 1 {
		t.Fatalf("invalid JSON: %s %v", out, e)
	}
	if _, e = run(t, home, "task", "add", "Ship project", "--project", projects[0].ID); e != nil {
		t.Fatal(e)
	}
	out, e = run(t, home, "task", "list", "--json")
	if e != nil {
		t.Fatal(e)
	}
	var tasks []domain.Task
	json.Unmarshal([]byte(out), &tasks)
	if len(tasks) != 1 {
		t.Fatal("task not persisted")
	}
	if _, e = run(t, home, "task", "done", tasks[0].ID); e != nil {
		t.Fatal(e)
	}
	out, _ = run(t, home, "task", "list", "--filter", "completed", "--json")
	if !strings.Contains(out, "Ship project") {
		t.Fatal("completed task missing")
	}
	if _, e = run(t, home, "project", "remove", projects[0].ID); e == nil {
		t.Fatal("noninteractive removal lacked --yes")
	}
	if _, e = run(t, home, "project", "remove", projects[0].ID, "--yes"); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(path); e != nil {
		t.Fatal("project folder removed")
	}
	out, e = run(t, home, "--help")
	if e != nil || !strings.Contains(out, "workflow") {
		t.Fatal("help is incomplete")
	}
	out, e = run(t, home, "version", "--json")
	if e != nil || !strings.Contains(out, "test-commit") {
		t.Fatal("version metadata missing")
	}
}
func TestInvalidConfigPreservedAndMissingGit(t *testing.T) {
	home := t.TempDir()
	file := filepath.Join(home, "config.toml")
	os.WriteFile(file, []byte("theme=[oops"), 0600)
	if _, e := run(t, home, "config", "show", "--json"); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(file)
	if string(b) != "theme=[oops" {
		t.Fatal("invalid configuration changed")
	}
	if _, e := run(t, home, "config", "reset", "--yes"); e != nil {
		t.Fatal(e)
	}
	if _, e := run(t, home, "config", "set", "git_executable", "missing-git-command"); e != nil {
		t.Fatal(e)
	}
	if _, e := run(t, home, "project", "add", t.TempDir()); e != nil {
		t.Fatal(e)
	}
	if _, e := run(t, home, "git", "status"); e == nil || !strings.Contains(e.Error(), "Git was not found") {
		t.Fatalf("missing Git error lost: %v", e)
	}
}
func TestHelperCLI(t *testing.T) {
	if os.Getenv("VANTAGE_CLI_HELPER") != "1" {
		return
	}
	if os.Getenv("VANTAGE_FAIL") != "" {
		os.Exit(9)
	}
	os.Stdout.WriteString("actual CLI workflow output\n")
	os.Exit(0)
}
func TestCLIWorkflowConsentSuccessAndExitCodes(t *testing.T) {
	home := t.TempDir()
	binary, _ := os.Executable()
	args := []string{"workflow", "add", "--name", "Tests", "--exec", binary, "--arg=-test.run=TestHelperCLI", "--dir", t.TempDir(), "--env", "VANTAGE_CLI_HELPER=1"}
	if _, e := run(t, home, args...); e != nil {
		t.Fatal(e)
	}
	if _, e := run(t, home, "workflow", "run", "Tests"); e == nil {
		t.Fatal("workflow ran without first approval")
	}
	out, e := run(t, home, "workflow", "run", "Tests", "--yes", "--json")
	if e != nil || !strings.Contains(out, "actual CLI workflow output") {
		t.Fatalf("workflow failed: %s %v", out, e)
	}
	t.Setenv("VANTAGE_FAIL", "1")
	_, e = run(t, home, "workflow", "run", "Tests", "--json")
	exit, ok := e.(*cli.ExitError)
	if !ok || exit.Code != 9 {
		t.Fatalf("workflow exit code not preserved: %v", e)
	}
}
