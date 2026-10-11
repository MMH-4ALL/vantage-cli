package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func command(t *testing.T, path string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", path}, args...)...)
	b, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %s %v", args, b, e)
	}
	return string(b)
}
func TestRealGitEmptyChangesRenamesAndDetachedHead(t *testing.T) {
	if _, e := exec.LookPath("git"); e != nil {
		t.Skip("Git not installed")
	}
	path := filepath.Join(t.TempDir(), "repo with spaces")
	os.Mkdir(path, 0700)
	command(t, path, "init", "-b", "main")
	s := Service{Executable: "git"}
	v, e := s.Inspect(context.Background(), path)
	if e != nil || !v.Clean || len(v.Commits) != 0 {
		t.Fatalf("empty repo: %+v %v", v, e)
	}
	command(t, path, "config", "user.name", "Test Developer")
	command(t, path, "config", "user.email", "test@example.invalid")
	os.WriteFile(filepath.Join(path, "old name.txt"), []byte("initial\n"), 0600)
	command(t, path, "add", "--", "old name.txt")
	command(t, path, "commit", "-m", "Initial test commit")
	command(t, path, "mv", "old name.txt", "new name.txt")
	name := "untracked odd name.txt"
	if runtime.GOOS != "windows" {
		name = "untracked\n\"odd\".txt"
	}
	os.WriteFile(filepath.Join(path, name), []byte("new"), 0600)
	v, e = s.Inspect(context.Background(), path)
	if e != nil {
		t.Fatal(e)
	}
	if len(v.Commits) != 1 || len(v.Changes) != 2 || !strings.Contains(v.Branch, "main") {
		t.Fatalf("bad git data: %+v", v)
	}
	foundRename, foundName := false, false
	for _, c := range v.Changes {
		if c.Path == "new name.txt" && c.Original == "old name.txt" {
			foundRename = true
		}
		if c.Path == name {
			foundName = true
		}
	}
	if !foundRename || !foundName {
		t.Fatalf("filenames or rename lost: %+v", v.Changes)
	}
	command(t, path, "checkout", "--detach")
	v, e = s.Inspect(context.Background(), path)
	if e != nil || !strings.Contains(v.Branch, "HEAD") {
		t.Fatalf("detached head: %+v %v", v, e)
	}
}
func TestParseFailuresAndMissingGit(t *testing.T) {
	if _, e := ParseStatus("M\x00"); e == nil {
		t.Fatal("accepted invalid porcelain")
	}
	if _, e := ParseStatus("R  new\x00"); e == nil {
		t.Fatal("accepted incomplete rename")
	}
	if _, e := (Service{Executable: "missing-git-vantage"}).Inspect(context.Background(), t.TempDir()); e == nil {
		t.Fatal("missing Git accepted")
	}
	remote := redactRemote("origin\thttps://user:secret@example.invalid/repo?token=secret (fetch)")
	if strings.Contains(remote, "secret") || strings.Contains(remote, "user:") {
		t.Fatal("remote credential exposed")
	}
}

func TestInspectionDoesNotRunRepositoryPrograms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the executable sentinel uses a POSIX shell")
	}
	if _, e := exec.LookPath("git"); e != nil {
		t.Skip("Git not installed")
	}
	path := t.TempDir()
	command(t, path, "init", "-b", "main")
	command(t, path, "config", "user.name", "Test Developer")
	command(t, path, "config", "user.email", "test@example.invalid")
	file := filepath.Join(path, "file.txt")
	if e := os.WriteFile(file, []byte("initial\n"), 0600); e != nil {
		t.Fatal(e)
	}
	command(t, path, "add", "file.txt")
	command(t, path, "commit", "-m", "Initial")
	marker := filepath.Join(path, "unexpected-execution")
	script := filepath.Join(path, ".git", "sentinel.sh")
	if e := os.WriteFile(script, []byte("#!/bin/sh\n: > \""+marker+"\"\ncat\n"), 0700); e != nil {
		t.Fatal(e)
	}
	command(t, path, "config", "core.fsmonitor", script)
	command(t, path, "config", "filter.sentinel.clean", script)
	command(t, path, "config", "filter.sentinel.process", script)
	command(t, path, "config", "filter.sentinel.required", "true")
	command(t, path, "config", "log.showSignature", "true")
	command(t, path, "config", "core.pager", script)
	if e := os.WriteFile(filepath.Join(path, ".gitattributes"), []byte("*.txt filter=sentinel\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(file, []byte("changed content\n"), 0600); e != nil {
		t.Fatal(e)
	}
	status, e := (Service{Executable: "git"}).Inspect(context.Background(), path)
	if e != nil || status.Clean {
		t.Fatalf("inspect changed repository: %+v %v", status, e)
	}
	if _, e = os.Stat(marker); !os.IsNotExist(e) {
		t.Fatalf("repository program executed or marker unreadable: %v", e)
	}
}
