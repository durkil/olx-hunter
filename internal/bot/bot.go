package bot

import (
	"fmt"
	"log"

	"olx-hunter/internal/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api *tgbotapi.BotAPI
	db *database.DB
}

func NewBot(token string, db *database.DB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	api.Debug = false

	log.Printf("Bot is authorized as: @%s", api.Self.UserName)

	return &Bot{
		api: api,
		db: db,
	}, nil
}

func (b *Bot) Start() {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := b.api.GetUpdatesChan(updateConfig)

	log.Println("Bot is started! Waitning for message...")

	for update := range updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
		}
	}
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	user, err := b.db.CreateOrUpdateUser(
		message.From.ID,
		message.From.UserName,
		message.From.FirstName,
	)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		b.sendMessage(message.Chat.ID, "Server error. Try later")
		return
	}

	log.Printf("Message from: %s (@%s) - %s", user.FirstName, user.Username, message.Text)

	if message.IsCommand() {
		switch message.Command() {
		case "start":
			b.handleStart(message)
		case "help":
			b.handleHelp(message)
		case "list":
			b.handleList(message)
		default:
			b.handleUnknown(message)
		}
	} else {
		b.handleText(message)
	}
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func (b *Bot) handleStart(message *tgbotapi.Message) {
	welcomeText := `👋 Привіт! Я бот для моніторингу оголошень на OLX!

🔍 Що я вмію:
• Створювати фільтри для пошуку
• Автоматично перевіряти нові оголошення
• Надсилати сповіщення про цікаві знахідки

📝 Команди:
/help - показати всі команди
/list - мої фільтри

Почнемо! 🚀`

	b.sendMessage(message.Chat.ID, welcomeText)
}

func (b *Bot) handleHelp(message *tgbotapi.Message) {
	helpText := `📚 Доступні команди:

🏠 Основні:
/start - почати роботу з ботом
/help - показати цю довідку

🔍 Фільтри:
/list - показати мої фільтри
/add - додати новий фільтр

Приклад:
/add "iPhone 15" iphone-15 25000 35000 київ

💡 Підказка: після створення фільтру я буду автоматично шукати нові оголошення і надсилати тобі!`

	b.sendMessage(message.Chat.ID, helpText)
}

func (b *Bot) handleUnknown(message *tgbotapi.Message) {
	text := `❓ Невідома команда: ` + message.Command() + `

Використай /help щоб побачити всі доступні команди.`
	
	b.sendMessage(message.Chat.ID, text)
}

func (b *Bot) handleText(message *tgbotapi.Message) {
	text := `💬 Я отримав твоє повідомлення: "` + message.Text + `"

Але я поки що працюю тільки з командами. Спробуй /help щоб побачити що я вмію! 🤖`

	b.sendMessage(message.Chat.ID, text)
}

func (b *Bot) handleList(message *tgbotapi.Message) {
	user, err := b.db.GetUserByTelegramID(message.From.ID)
	if err != nil || user == nil {
		b.sendMessage(message.Chat.ID, "❌ Помилка отримання даних користувача")
		return
	}

	filters, err := b.db.GetUserFilters(user.ID)
	if err != nil {
		log.Printf("Error getting user filters", err)
		b.sendMessage(message.Chat.ID, "❌ Помилка отримання фільтрів пошуку")
		return
	}

	if len(filters) == 0 {
		text := `📝 У тебе поки що немає фільтрів.

Створи перший фільтр командою:
/add "iPhone 15" iphone-15 25000 35000 київ

Формат: назва, запит, мін_ціна, макс_ціна, місто`

		b.sendMessage(message.Chat.ID, text)
		return
	}

	text := fmt.Sprintf("📋 Твої фільтри (%d):\n\n", len(filters))

	for i, filter := range filters {
		status := "🟢"
		if !filter.IsActive {
			status = "🔴"
		}

		text += fmt.Sprintf("%s **%d.** %s\n", status, i+1, filter.Name)
		text += fmt.Sprintf("   🔍 Запит: `%s`\n", filter.Query)

		if filter.MinPrice > 0 || filter.MaxPrice > 0 {
			priceRange := ""
			if filter.MinPrice > 0 && filter.MaxPrice > 0 {
				priceRange = fmt.Sprintf("%d - %d грн", filter.MinPrice, filter.MaxPrice)
			} else if filter.MinPrice > 0 {
				priceRange = fmt.Sprintf("від %d грн", filter.MinPrice)
			} else {
				priceRange = fmt.Sprintf("до %d грн", filter.MaxPrice)
			}
			text += fmt.Sprintf("   💰 Ціна: %s\n", priceRange)
		}

		if filter.City != "" {
			text += fmt.Sprintf("   🏙 Місто: %s\n", filter.City)
		}

		text += "\n"
	}

	text += "🟢 активний | 🔴 неактивний"

	b.sendMessage(message.Chat.ID, text)
}