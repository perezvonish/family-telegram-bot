package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"perezvonish/health-tracker/internal/entry-points/telegram_bot"
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
	defer mongoDB.Close(ctx)

	telegramChatBot := telegram_bot.NewChatBot(ctx, cfg.Telegram)
	telegramChatBot.Start()

	<-ctx.Done()
	gracefulShutdown(mongoDB)
}

func gracefulShutdown(mongoDB *database.MongoDB) {
	fmt.Println("\nShutdown signal received...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := mongoDB.Close(ctx); err != nil {
		log.Printf("Error closing MongoDB connection: %v", err)
	}

	fmt.Println("Application stopped gracefully")
}
