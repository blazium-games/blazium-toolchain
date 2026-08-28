package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/blazium-games/blazium-toolchain/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(app.Run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
