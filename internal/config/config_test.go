package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigurationRoundTripAndValidation(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	s := Defaults()
	s.Theme = "light"
	s.Editor = "editor with spaces"
	s.EditorArgs = []string{"--wait", "a b"}
	if e := Save(p, s); e != nil {
		t.Fatal(e)
	}
	got, e := Load(p)
	if e != nil {
		t.Fatal(e)
	}
	if got.Theme != "light" || got.EditorArgs[1] != "a b" {
		t.Fatalf("lost settings: %+v", got)
	}
	s.RefreshSeconds = 0
	if Save(p, s) == nil {
		t.Fatal("accepted zero refresh interval")
	}
	got, e = Load(p)
	if e != nil || got.RefreshSeconds == 0 {
		t.Fatal("invalid save changed file")
	}
}
func TestCorruptSettingsArePreserved(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	raw := []byte("theme = [broken")
	os.WriteFile(p, raw, 0600)
	s, e := Load(p)
	if e == nil || s.Theme != "midnight" {
		t.Fatal("expected safe defaults and warning")
	}
	b, _ := os.ReadFile(p)
	if string(b) != string(raw) {
		t.Fatal("corrupt settings were overwritten")
	}
}
func TestPortablePaths(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "with spaces")
	t.Setenv("VANTAGE_HOME", dir)
	p, e := Resolve("")
	if e != nil || p.Config != filepath.Join(dir, "config.toml") {
		t.Fatalf("wrong paths: %+v %v", p, e)
	}
}
