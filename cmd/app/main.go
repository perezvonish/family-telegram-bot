package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"perezvonish/health-tracker/internal/app"
	"perezvonish/health-tracker/internal/infrastructure/database"
	"perezvonish/health-tracker/internal/shared/config"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.Init()
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mongoDB, err := database.NewMongoDB(ctx, cfg.Mongo)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	container := app.NewContainer(ctx, cfg, mongoDB)
	container.TelegramBot.Start()

	<-ctx.Done()
	gracefulShutdown(container)
}

func gracefulShutdown(container *app.Container) {
	fmt.Println("\nShutdown signal received...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := container.Close(ctx); err != nil {
		log.Printf("Error closing container: %v", err)
	}

	fmt.Println("Application stopped gracefully")
}
