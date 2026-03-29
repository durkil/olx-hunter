package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"olx-hunter/internal/bot"
	"olx-hunter/internal/database"
	"olx-hunter/internal/models"
	"olx-hunter/internal/scraper"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Env file is not found")
	}

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN is not set")
	}

	dsn := "host=localhost user=postgres password=password dbname=olx_hunter port=5432 sslmode=disable"
	db, err := database.Connect(dsn)
	if err != nil {
		log.Fatal("Error connecting to db:", err)
	}

	telegramBot, err := bot.NewBot(botToken, db)
	if err != nil {
		log.Fatal("Error creating bot:", err)
	}

	log.Println("🤖 Starting Telegram Bot...")

	log.Println("Starting OLX Hunter Scraper Service...")

	notifyChan := make(chan models.Notification, 100)

	scraperService := scraper.NewScraperService(db, notifyChan, 5)
	if err := scraperService.LoadExistingFilters(); err != nil {
		log.Fatalf("Failed to load existing filters: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go scraperService.StartPeriodicScraping(ctx)
	go telegramBot.Start()
	go telegramBot.ListenNotifications(notifyChan)

	log.Println("OLX Hunter is running!")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutdown signal received...")
	cancel()
	time.Sleep(2 * time.Second)
	log.Println("Goodbye!")
}