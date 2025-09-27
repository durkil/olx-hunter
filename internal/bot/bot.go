package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"olx-hunter/internal/database"
	"olx-hunter/internal/scraper"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api *tgbotapi.BotAPI
	db  *database.DB
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
		db:  db,
	}, nil
}

func (b *Bot) Start() {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := b.api.GetUpdatesChan(updateConfig)

	log.Println("Bot is started! Waiting for message...")

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
		case "add":
			b.handleAdd(message)
		case "search":
			b.handleSearch(message)
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
/add iPhone15;iphone-15;25000;35000;київ

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
		log.Printf("Error getting user filters %v", err)
		b.sendMessage(message.Chat.ID, "❌ Помилка отримання фільтрів пошуку")
		return
	}

	if len(filters) == 0 {
		text := `📝 У тебе поки що немає фільтрів.

Створи перший фільтр командою:
/add iPhone15;iphone-15;25000;35000;київ

Формат: назва;запит;мін_ціна;макс_ціна;місто`

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

func (b *Bot) handleAdd(message *tgbotapi.Message) {
	args := strings.Split(message.CommandArguments(), ";")

	if len(args) != 5 {
		text := `❌ Неправильний формат команди!

📝 Правильний формат:
/add назва;запит;мін_ціна;макс_ціна;місто

📋 Приклад:
/add iPhone15;iphone-15;25000;35000;київ`

		b.sendMessage(message.Chat.ID, text)
		return
	}

	name := strings.TrimSpace(args[0])
	query := strings.TrimSpace(args[1])
	minPriceStr := strings.TrimSpace(args[2])
	maxPriceStr := strings.TrimSpace(args[3])
	city := strings.TrimSpace(args[4])

	if name == "" {
		b.sendMessage(message.Chat.ID, "❌ Назва фільтру не може бути пустою!")
		return
	}
	if query == "" {
		b.sendMessage(message.Chat.ID, "❌ Пошуковий запит не може бути пустим!")
		return
	}

	var minPrice, maxPrice int
	var err error

	if minPriceStr != "" {
		minPrice, err = strconv.Atoi(minPriceStr)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Мінімальна ціна має бути числом!")
			return
		}
	}

	if maxPriceStr != "" {
		maxPrice, err = strconv.Atoi(maxPriceStr)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Максимальна ціна має бути числом!")
			return
		}
	}

	if minPrice < 0 || maxPrice < 0 {
		b.sendMessage(message.Chat.ID, "❌ Ціни не можуть бути від'ємними!")
		return
	}

	if minPrice > maxPrice && maxPrice > 0 {
		b.sendMessage(message.Chat.ID, "❌ Мінімальна ціна не може бути більшою за максимальну!")
		return
	}

	user, err := b.db.GetUserByTelegramID(message.From.ID)
	if err != nil || user == nil {
		b.sendMessage(message.Chat.ID, "❌ Помилка отримання даних користувача")
		return
	}

	createdFilter, err := b.db.CreateFilter(user.ID, name, query, minPrice, maxPrice, city)
	if err != nil {
		log.Printf("Error creating filter: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Помилка створення фільтру. Спробуй ще раз.")
		return
	}

	successText := fmt.Sprintf(`✅ Фільтр створено успішно!

📋 **%s**
🔍 Запит: %s
💰 Ціна: `, createdFilter.Name, createdFilter.Query)

	if createdFilter.MinPrice > 0 && createdFilter.MaxPrice > 0 {
		successText += fmt.Sprintf("%d - %d грн", createdFilter.MinPrice, createdFilter.MaxPrice)
	} else if createdFilter.MinPrice > 0 {
		successText += fmt.Sprintf("від %d грн", createdFilter.MinPrice)
	} else if createdFilter.MaxPrice > 0 {
		successText += fmt.Sprintf("до %d грн", createdFilter.MaxPrice)
	} else {
		successText += "без обмежень ціни"
	}

	if createdFilter.City != "" {
		successText += fmt.Sprintf("\n🏙 Місто: %s", createdFilter.City)
	}

	successText += "\n\n🟢 Фільтр активний і готовий до роботи!"

	b.sendMessage(message.Chat.ID, successText)
}

func (b *Bot) handleSearch(message *tgbotapi.Message) {
	args := strings.Fields(message.CommandArguments())

	if len(args) == 0 {
		b.sendMessage(message.Chat.ID, `🔍 Тестовий пошук на OLX:

Використання: /search запит [мін_ціна] [макс_ціна] [місто]
Приклади:
/search iphone-15
/search iphone-15 25000 35000
/search iphone-15 0 0 київ`)
		return
	}
	query := args[0]
	minPrice, maxPrice := 0, 0
	city := ""

	if len(args) > 1 {
		minPrice, _ = strconv.Atoi(args[1])
	}
	if len(args) > 2 {
		maxPrice, _ = strconv.Atoi(args[2])
	}
	if len(args) > 3 {
		city = strings.Join(args[3:], " ")
	}

	olxScraper := scraper.NewOLXScraper()

	filters := scraper.SearchFilters{
		Query:    query,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		City:     city,
	}

	listings, err := olxScraper.SearchListings(filters)
	if err != nil {
		log.Printf("Error scraping OLX: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Помилка пошуку на OLX: "+err.Error())
		return
	}
	response := fmt.Sprintf("🔍 Знайдено %d оголошень:\n\n", len(listings))
	for i, listing := range listings {
		if i >= 5 {
			response += fmt.Sprintf("... та ще %d оголошень", len(listings)-5)
			break
		}
		response += fmt.Sprintf("%d. %s\n%s\n%s\n🔗 %s\n\n", i+1, listing.Title, listing.Price, listing.Location, listing.URL)
	}
	b.sendMessage(message.Chat.ID, response)
}
