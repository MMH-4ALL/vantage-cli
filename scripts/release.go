//go:build ignore

// Run with: go run scripts/release.go --version 1.0.0
// This packager needs only Go and works on Windows, Linux, and macOS.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	version := flag.String("version", "", "Release version, for example 1.0.0")
	out := flag.String("out", "dist", "Output directory")
	flag.Parse()
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?$`).MatchString(*version) {
		return fmt.Errorf("provide a valid --version, for example 1.0.0")
	}
	if e := os.MkdirAll(*out, 0755); e != nil {
		return e
	}
	commit := "unknown"
	if b, e := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); e == nil {
		commit = strings.TrimSpace(string(b))
	}
	date := time.Now().UTC().Format(time.RFC3339)
	if s := os.Getenv("SOURCE_DATE_EPOCH"); s != "" {
		var seconds int64
		if _, e := fmt.Sscan(s, &seconds); e != nil {
			return e
		}
		date = time.Unix(seconds, 0).UTC().Format(time.RFC3339)
	}
	var paths []string
	for _, target := range [][2]string{{"windows", "amd64"}, {"windows", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}} {
		goos, arch := target[0], target[1]
		name := "vantage"
		if goos == "windows" {
			name += ".exe"
		}
		dir, e := os.MkdirTemp("", "vantage-release-*")
		if e != nil {
			return e
		}
		binary := filepath.Join(dir, name)
		goExecutable := filepath.Join(runtime.GOROOT(), "bin", "go")
		if runtime.GOOS == "windows" {
			goExecutable += ".exe"
		}
		cmd := exec.Command(goExecutable, "build", "-trimpath", "-buildvcs=false", "-ldflags", "-s -w -X main.version="+*version+" -X main.commit="+commit+" -X main.date="+date, "-o", binary, "./cmd/vantage")
		cmd.Env = environment(goos, arch)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if e = cmd.Run(); e != nil {
			os.RemoveAll(dir)
			return fmt.Errorf("build %s/%s: %w", goos, arch, e)
		}
		archive := filepath.Join(*out, fmt.Sprintf("vantage_%s_%s_%s.zip", *version, goos, arch))
		files := map[string]string{name: binary}
		for _, doc := range []string{"LICENSE", "README.md", "THIRD_PARTY_NOTICES.md", "SECURITY.md", "CHANGELOG.md", "CONTRIBUTING.md", "docs/ENGINEERING_REPORT.md", "docs/ARCHITECTURE.md", "docs/DEVELOPMENT.md", "docs/RELEASE_NOTES.md"} {
			files[doc] = doc
		}
		screenshots, err := filepath.Glob("docs/screenshots/*.png")
		if err != nil {
			os.RemoveAll(dir)
			return err
		}
		for _, shot := range screenshots {
			files[filepath.ToSlash(shot)] = shot
		}
		e = pack(archive, files, date)
		os.RemoveAll(dir)
		if e != nil {
			return e
		}
		paths = append(paths, archive)
		fmt.Printf("Built %s/%s -> %s\n", goos, arch, archive)
	}
	var sums strings.Builder
	sort.Strings(paths)
	for _, path := range paths {
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		hash := sha256.New()
		_, e = io.Copy(hash, f)
		f.Close()
		if e != nil {
			return e
		}
		fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(hash.Sum(nil)), filepath.Base(path))
	}
	return os.WriteFile(filepath.Join(*out, "checksums.txt"), []byte(sums.String()), 0644)
}

func environment(goos, arch string) []string {
	var env []string
	for _, e := range os.Environ() {
		k, _, _ := strings.Cut(e, "=")
		if k != "GOOS" && k != "GOARCH" && k != "CGO_ENABLED" {
			env = append(env, e)
		}
	}
	return append(env, "GOOS="+goos, "GOARCH="+arch, "CGO_ENABLED=0")
}
func pack(path string, files map[string]string, date string) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	z := zip.NewWriter(f)
	var keys []string
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	stamp, _ := time.Parse(time.RFC3339, date)
	for _, name := range keys {
		b, e := os.ReadFile(files[name])
		if e != nil {
			z.Close()
			f.Close()
			return e
		}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: stamp}
		mode := os.FileMode(0644)
		if name == "vantage" {
			mode = 0755
		}
		header.SetMode(mode)
		w, e := z.CreateHeader(header)
		if e != nil {
			z.Close()
			f.Close()
			return e
		}
		if _, e = w.Write(b); e != nil {
			z.Close()
			f.Close()
			return e
		}
	}
	if e = z.Close(); e != nil {
		f.Close()
		return e
	}
	return f.Close()
}
