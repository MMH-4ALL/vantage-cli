package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/MMH-4ALL/vantage-cli/internal/cli"
)

var version = "dev"
var commit = "unknown"
var date = "unknown"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	root := cli.New(version, commit, date)
	if e := root.ExecuteContext(ctx); e != nil {
		fmt.Fprintln(os.Stderr, "vantage:", e)
		code := 1
		if v, ok := e.(*cli.ExitError); ok {
			code = v.Code
		}
		os.Exit(code)
	}
}
