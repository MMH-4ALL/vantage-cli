//go:build windows

package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// Windows' built-in taskkill terminates the descendant tree when available.
// Resolve under SystemRoot instead of trusting a project-controlled PATH entry.
func prepare(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		root := os.Getenv("SystemRoot")
		if root != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			kill := exec.CommandContext(ctx, filepath.Join(root, "System32", "taskkill.exe"), "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
			kill.WaitDelay = 2 * time.Second
			if kill.Run() == nil {
				return nil
			}
		}
		return cmd.Process.Kill()
	}
}
