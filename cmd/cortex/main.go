package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/cortex/cli/internal/app"
	"github.com/cortex/cli/internal/ux"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	rootCmd := app.NewRootCommand()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		ux.PrintError(err)
		os.Exit(1)
	}
}
