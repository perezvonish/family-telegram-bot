package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"perezvonish/health-tracker/internal/shared/config"
	"syscall"
	"time"
)

func main() {
	_, err := config.Init()
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	gracefulShutdown()
}

func gracefulShutdown() {
	fmt.Println("\nShutdown signal received...")

	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Application stopped gracefully")
}
