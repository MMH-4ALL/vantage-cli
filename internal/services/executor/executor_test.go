package executor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MMH-4ALL/vantage-cli/internal/domain"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("VANTAGE_TEST_HELPER") != "1" {
		return
	}
	args := os.Args
	mode := ""
	for i, a := range args {
		if a == "--" && i+1 < len(args) {
			mode = args[i+1]
			break
		}
	}
	switch mode {
	case "ok":
		fmt.Print("output\n")
		for _, a := range args {
			fmt.Println(a)
		}
	case "fail":
		fmt.Fprintln(os.Stderr, "intentional error")
		os.Exit(7)
	case "wait":
		time.Sleep(10 * time.Second)
	case "secret":
		fmt.Println(os.Getenv("VANTAGE_TEST_TOKEN"))
		fmt.Print("\x1b[2Jnormal\x1b[31m text\x1b[0m")
	case "large":
		fmt.Print(strings.Repeat("x", MaxOutput+500))
	}
	os.Exit(0)
}
func workflow(t *testing.T, mode string) domain.Workflow {
	t.Helper()
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	return domain.Workflow{Name: "Helper", Executable: binary, Args: []string{"-test.run=TestHelperProcess", "--", mode}, Directory: t.TempDir(), TimeoutSeconds: 5, Env: map[string]string{"VANTAGE_TEST_HELPER": "1"}}
}
func TestRealExecutionFailureAndLiteralArguments(t *testing.T) {
	w := workflow(t, "ok")
	w.Args = append(w.Args, "literal; rm -rf imaginary", "path with spaces")
	r := Run(context.Background(), w, NewLog())
	if r.ExitCode != 0 || !strings.Contains(r.Output, "literal; rm -rf imaginary") || !strings.Contains(r.Output, "path with spaces") {
		t.Fatalf("literal execution failed: %+v", r)
	}
	r = Run(context.Background(), workflow(t, "fail"), NewLog())
	if r.ExitCode != 7 || r.Error == "" || !strings.Contains(r.Output, "intentional error") {
		t.Fatalf("failure lost: %+v", r)
	}
}
func TestTimeoutAndCancellation(t *testing.T) {
	w := workflow(t, "wait")
	w.TimeoutSeconds = 1
	start := time.Now()
	r := Run(context.Background(), w, NewLog())
	if !r.Canceled || time.Since(start) > 4*time.Second {
		t.Fatalf("timeout failed: %+v", r)
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	r = Run(ctx, w, NewLog())
	if !r.Canceled {
		t.Fatal("cancellation not detected")
	}
}
func TestSecretRedactionOutputLimitAndConcurrentPolling(t *testing.T) {
	t.Setenv("VANTAGE_TEST_TOKEN", "very-secret-value")
	w := workflow(t, "secret")
	log := NewLog()
	var wg sync.WaitGroup
	done := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				_ = log.Text()
			}
		}
	}()
	r := Run(context.Background(), w, log)
	close(done)
	wg.Wait()
	if strings.Contains(r.Output, "very-secret-value") || strings.Contains(r.Output, "\x1b") || !strings.Contains(r.Output, "[REDACTED]") {
		t.Fatalf("unsafe output: %q", r.Output)
	}
	r = Run(context.Background(), workflow(t, "large"), NewLog())
	if len(r.Output) > MaxOutput+100 || !strings.Contains(r.Output, "omitted") {
		t.Fatal("output limit failed")
	}
}
func TestInvalidAndMissingExecutable(t *testing.T) {
	w := workflow(t, "ok")
	w.Executable = "vantage-missing-executable-xyz"
	r := Run(context.Background(), w, NewLog())
	if r.Error == "" || r.ExitCode != -1 {
		t.Fatal("missing executable accepted")
	}
	w.TimeoutSeconds = 0
	r = Run(context.Background(), w, NewLog())
	if r.Error == "" {
		t.Fatal("invalid workflow executed")
	}
}
