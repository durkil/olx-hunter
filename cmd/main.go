package main

import (
	"log"
    "os"

    "olx-hunter/internal/bot"
    "olx-hunter/internal/database"
	"olx-hunter/internal/scraper"
	"context"
	"os/signal"
	"syscall"
	"time"
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
	telegramBot.Start()

	log.Println("Starting OLX Hunter Scraper Service...")

	scraperService := scraper.NewScraperService(db)
	if err := scraperService.LoadExistingFilters(); err != nil {
		log.Fatalf("Failed to load existing filters: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go scraperService.StartPeriodicScraping(ctx)

	go func() {
		c := make(chan os.Signal, 1)
        signal.Notify(c, os.Interrupt, syscall.SIGTERM)
        <-c
        log.Println("Shutdown signal received...")
        cancel()
        time.Sleep(2 * time.Second)
        os.Exit(0)
	}()

	log.Println("OLX Hunter is running!")
    telegramBot.Start()
}