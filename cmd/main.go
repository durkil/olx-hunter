package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"olx-hunter/internal/bot"
	"olx-hunter/internal/config"
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

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Config error:", err)
	}

	db, err := database.Connect(cfg.DatabaseDSN)
	if err != nil {
		log.Fatal("Error connecting to db:", err)
	}

	log.Println("Starting OLX Hunter Scraper Service...")

	notifyChan := make(chan models.Notification, 100)

	scraperService := scraper.NewScraperService(db, notifyChan, cfg.WorkerCount, cfg.ScrapeInterval)
	if err := scraperService.LoadExistingFilters(); err != nil {
		log.Fatalf("Failed to load existing filters: %v", err)
	}

	log.Println("🤖 Starting Telegram Bot...")

	telegramBot, err := bot.NewBot(cfg.BotToken, db, cfg.RedisAddr, scraperService)
	if err != nil {
		log.Fatal("Error creating bot:", err)
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