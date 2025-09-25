package main

import (
	"log"
    "os"

    "olx-hunter/internal/bot"
    "olx-hunter/internal/database"
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

	telegramBot.Start()
}