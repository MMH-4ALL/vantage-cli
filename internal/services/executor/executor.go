// Package executor is the single boundary for local subprocess execution.
package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/MMH-4ALL/vantage-cli/internal/core"
	"github.com/MMH-4ALL/vantage-cli/internal/domain"
)

const MaxOutput = 2 * 1024 * 1024

type Result struct {
	Output   string        `json:"output"`
	ExitCode int           `json:"exit_code"`
	Elapsed  time.Duration `json:"elapsed_ns"`
	Canceled bool          `json:"canceled"`
	Error    string        `json:"error,omitempty"`
}

// Log is bounded and safe to poll while a process is running. Raw output is private.
type Log struct {
	mu        sync.Mutex
	data      []byte
	truncated bool
	secrets   []string
	dirty     bool
	cache     string
}

func (l *Log) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(p)
	l.data = append(l.data, p...)
	l.dirty = true
	if len(l.data) > MaxOutput {
		l.data = append([]byte(nil), l.data[len(l.data)-MaxOutput:]...)
		l.truncated = true
	}
	return n, nil
}
func (l *Log) Text() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.dirty {
		return l.cache
	}
	s := string(l.data)
	for _, v := range l.secrets {
		if len(v) >= 4 {
			s = strings.ReplaceAll(s, v, "[REDACTED]")
			// Mask an incomplete secret at the live log boundary as well.
			for n := min(len(v)-1, len(s)); n >= 4; n-- {
				if strings.HasSuffix(s, v[:n]) {
					s = s[:len(s)-n] + "[REDACTED]"
					break
				}
			}
		}
	}
	s = Clean(s)
	if l.truncated {
		s = "[earlier output omitted: 2 MiB limit]\n" + s
	}
	l.cache = s
	l.dirty = false
	return s
}

var escapes = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[@-_])`)

// Clean prevents child commands and untrusted repository names from controlling the terminal.
func Clean(s string) string {
	s = escapes.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) || r == 0x202e || r == 0x202d || r >= 0x2066 && r <= 0x2069 {
			return -1
		}
		return r
	}, s)
}

func environment(extra map[string]string) ([]string, []string) {
	m := map[string]string{}
	for _, e := range os.Environ() {
		k, v, ok := strings.Cut(e, "=")
		if ok {
			m[k] = v
		}
	}
	for k, v := range extra {
		m[k] = v
	}
	var keys, secrets []string
	for k, v := range m {
		keys = append(keys, k)
		if core.SensitiveKey(k) && v != "" {
			secrets = append(secrets, v)
		}
	}
	sort.Strings(keys)
	var env []string
	for _, k := range keys {
		env = append(env, k+"="+m[k])
	}
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	return env, secrets
}

func NewLog() *Log { _, s := environment(nil); return &Log{secrets: s} }

// Run does not interpret shell syntax. The caller owns first-run consent.
func Run(ctx context.Context, w domain.Workflow, log *Log) Result {
	start := time.Now()
	result := Result{ExitCode: -1}
	if e := core.ValidateWorkflow(w); e != nil {
		result.Error = e.Error()
		return result
	}
	for _, k := range w.InheritEnv {
		if _, ok := os.LookupEnv(k); !ok {
			result.Error = fmt.Sprintf("environment variable %s is not set", k)
			return result
		}
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(w.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, w.Executable, w.Args...)
	cmd.Dir = w.Directory
	var secrets []string
	cmd.Env, secrets = environment(w.Env)
	log.mu.Lock()
	log.secrets = secrets
	log.dirty = true
	log.mu.Unlock()
	cmd.Stdout = log
	cmd.Stderr = log
	cmd.WaitDelay = 2 * time.Second
	prepare(cmd)
	e := cmd.Run()
	result.Elapsed = time.Since(start)
	result.Output = log.Text()
	result.Canceled = ctx.Err() != nil
	if e == nil {
		result.ExitCode = 0
	} else {
		var exit *exec.ExitError
		if errors.As(e, &exit) {
			result.ExitCode = exit.ExitCode()
		}
		result.Error = Clean(e.Error())
	}
	if result.Canceled {
		result.Error = ctx.Err().Error()
	}
	return result
}

// Capture also bounds read-only commands, including Git on slow filesystems.
func Capture(ctx context.Context, directory, executable string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = directory
	cmd.WaitDelay = 2 * time.Second
	prepare(cmd)
	out := &boundedBuffer{}
	errOut := NewLog()
	cmd.Stdout = out
	cmd.Stderr = errOut
	e := cmd.Run()
	if e != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("%s: %s", e, strings.TrimSpace(errOut.Text()))
	}
	if out.overflow {
		return "", errors.New("command output exceeded the 2 MiB safety limit")
	}
	return string(out.data), nil
}

type boundedBuffer struct {
	data     []byte
	overflow bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := MaxOutput - len(b.data)
	if len(p) > remaining {
		b.overflow = true
		p = p[:remaining]
	}
	b.data = append(b.data, p...)
	return n, nil
}
