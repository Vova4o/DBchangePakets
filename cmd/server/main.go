package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"dbchangepakets/internal/app"
)

func main() {
	// Initialize default slog logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Listen for interruption and termination OS signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Hand over execution to the application package
	if err := app.Run(ctx, logger); err != nil {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}
