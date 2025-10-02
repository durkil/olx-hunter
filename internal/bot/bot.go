package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"olx-hunter/internal/cache"
	"olx-hunter/internal/database"
	"olx-hunter/internal/scraper"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api   *tgbotapi.BotAPI
	db    *database.DB
	cache *cache.RedisCache
}

type FilterCreationState struct {
	Step int
	Data map[string]string
}

var creationStates = make(map[int64]*FilterCreationState)

func NewBot(token string, db *database.DB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	api.Debug = false

	redisCache := cache.NewRedisCache()

	if err := redisCache.Ping(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		log.Printf("Bot will work without caching!")
	} else {
		log.Printf("Redis connected successfully")
	}

	log.Printf("Bot is authorized as: @%s", api.Self.UserName)

	return &Bot{
		api:   api,
		db:    db,
		cache: redisCache,
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
		case "create":
			b.handleCreate(message)
		case "find":
			b.handleFind(message)
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
/create - створити новий фільтр (покроково)
/find - знайти оголошення по конкретному фільтру
/find [номер] - знайти по фільтру з номером

💡 Підказка: введи "-" щоб пропустити необов'язкові поля (ціна, місто)`

	b.sendMessage(message.Chat.ID, helpText)
}

func (b *Bot) handleUnknown(message *tgbotapi.Message) {
	text := `❓ Невідома команда: ` + message.Command() + `

Використай /help щоб побачити всі доступні команди.`

	b.sendMessage(message.Chat.ID, text)
}

func (b *Bot) handleText(message *tgbotapi.Message) {
	state, exists := creationStates[message.From.ID]
	if !exists {
		text := `💬 Я отримав твоє повідомлення: "` + message.Text + `"

Але я поки що працюю тільки з командами. Спробуй /help щоб побачити що я вмію! 🤖`

		b.sendMessage(message.Chat.ID, text)
		return
	}

	switch state.Step {
	case 1:
		state.Data["name"] = message.Text
		state.Step++
		b.sendMessage(message.Chat.ID, "🔍 Введи пошуковий запит (наприклад, iphone-15):")
	case 2:
		state.Data["query"] = message.Text
		state.Step++
		b.sendMessage(message.Chat.ID, "💰 Мінімальна ціна (або 0):")
	case 3:
		minPriceStr := strings.TrimSpace(message.Text)
		if minPriceStr == "-" {
			minPriceStr = "0"
		}
		state.Data["min_price"] = minPriceStr
		state.Step++
		b.sendMessage(message.Chat.ID, "💰 Максимальна ціна (або 0):")
	case 4:
		maxPriceStr := strings.TrimSpace(message.Text)
		if maxPriceStr == "-" {
			maxPriceStr = "0"
		}
		state.Data["max_price"] = maxPriceStr
		state.Step++
		b.sendMessage(message.Chat.ID, "🏙 Місто (або залиш порожнім, або введи -):")
	case 5:
		city := strings.TrimSpace(message.Text)
		if city == "-" {
			city = ""
		}
		state.Data["city"] = city

		name := strings.TrimSpace(state.Data["name"])
		query := strings.TrimSpace(state.Data["query"])
		minPriceStr := strings.TrimSpace(state.Data["min_price"])
		maxPriceStr := strings.TrimSpace(state.Data["max_price"])

		minPrice := 0
		if minPriceStr != "0" && minPriceStr != "" {
			var err error
			minPrice, err = strconv.Atoi(minPriceStr)
			if err != nil {
				b.sendMessage(message.Chat.ID, "❌ Мінімальна ціна має бути числом!")
				delete(creationStates, message.From.ID)
				return
			}
		}

		maxPrice := 0
		if maxPriceStr != "0" && minPriceStr != "" {
			var err error
			maxPrice, err = strconv.Atoi(maxPriceStr)
			if err != nil {
				b.sendMessage(message.Chat.ID, "❌ Максимальна ціна має бути числом!")
				delete(creationStates, message.From.ID)
				return
			}
		}

		if minPrice < 0 || maxPrice < 0 {
			b.sendMessage(message.Chat.ID, "❌ Ціни не можуть бути від'ємними!")
			delete(creationStates, message.From.ID)
			return
		}

		if minPrice > maxPrice && maxPrice > 0 {
			b.sendMessage(message.Chat.ID, "❌ Мінімальна ціна не може бути більшою за максимальну!")
			delete(creationStates, message.From.ID)
			return
		}

		user, err := b.db.GetUserByTelegramID(message.From.ID)
		if err != nil || user == nil {
			b.sendMessage(message.Chat.ID, "❌ Помилка отримання даних користувача")
			delete(creationStates, message.From.ID)
			return
		}

		createdFilter, err := b.db.CreateFilter(user.ID, name, query, minPrice, maxPrice, city)
		if err != nil {
			log.Printf("Error creating filter: %v", err)
			b.sendMessage(message.Chat.ID, "❌ Помилка створення фільтру. Спробуй ще раз.")
			delete(creationStates, message.From.ID)
			return
		}

		successText := fmt.Sprintf(`✅ Фільтр створено успішно!

📋 **%s**
🔍 Запит: %s`, createdFilter.Name, createdFilter.Query)

		if createdFilter.MinPrice > 0 || createdFilter.MaxPrice > 0 {
			successText += "\n💰 Ціна: "
			if createdFilter.MinPrice > 0 && createdFilter.MaxPrice > 0 {
				successText += fmt.Sprintf("%d - %d грн", createdFilter.MinPrice, createdFilter.MaxPrice)
			} else if createdFilter.MinPrice > 0 {
				successText += fmt.Sprintf("від %d грн", createdFilter.MinPrice)
			} else {
				successText += fmt.Sprintf("до %d грн", createdFilter.MaxPrice)
			}
		} else {
			successText += "\n💰 Ціна: без обмежень"
		}

		if createdFilter.City != "" {
			successText += fmt.Sprintf("\n🏙 Місто: %s", createdFilter.City)
		}

		successText += "\n\n🟢 Фільтр активний і готовий до роботи!"

		b.sendMessage(message.Chat.ID, successText)
		delete(creationStates, message.From.ID)
	}
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
		text := `📝 У тебе поки що немає фільтрів.`

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

func (b *Bot) handleCreate(message *tgbotapi.Message) {

	creationStates[message.From.ID] = &FilterCreationState{
		Step: 1,
		Data: make(map[string]string),
	}
	b.sendMessage(message.Chat.ID, "📝 Введи назву фільтра:")
}

func (b *Bot) handleFind(message *tgbotapi.Message) {
	args := strings.Fields(message.CommandArguments())

	user, err := b.db.GetUserByTelegramID(message.From.ID)
	if err != nil || user == nil {
		b.sendMessage(message.Chat.ID, "❌ Помилка отримання даних користувача")
		return
	}

	filters, err := b.db.GetUserFilters(user.ID)
	if err != nil {
		b.sendMessage(message.Chat.ID, "❌ Помилка отримання фільтрів")
		return
	}

	if len(filters) == 0 {
		b.sendMessage(message.Chat.ID, "❌ У тебе немає активних фільтрів. Створи через /create")
		return
	}

	if len(args) == 0 {
		text := "🔍 Вкажи номер фільтра для пошуку:\n\n"
		for i, filter := range filters {
			status := "🟢"
			if !filter.IsActive {
				status = "🔴"
			}
			text += fmt.Sprintf("%s **%d.** %s - `%s`\n", status, i+1, filter.Name, filter.Query)
		}
		text += "\n📝 Використання: `/find 1` (для пошуку по першому фільтру)"
		b.sendMessage(message.Chat.ID, text)
		return
	}

	filterNum, err := strconv.Atoi(args[0])
	if err != nil || filterNum < 1 || filterNum > len(filters) {
		b.sendMessage(message.Chat.ID, fmt.Sprintf("❌ Невірний номер фільтра. Використай номер від 1 до %d", len(filters)))
		return
	}

	selectedFilter := filters[filterNum-1]

	if !selectedFilter.IsActive {
		b.sendMessage(message.Chat.ID, "❌ Цей фільтр неактивний")
		return
	}

	cacheKey := fmt.Sprintf("%s:%d:%d:%s", selectedFilter.Query, selectedFilter.MinPrice, selectedFilter.MaxPrice, selectedFilter.City)

	if cached, found := b.cache.GetCachedResults(cacheKey); found {
		b.sendMessage(message.Chat.ID, "⚡ Результати з кешу (швидко!):")
		b.sendSearchResults(message.Chat.ID, selectedFilter.Name, cached)
		return
	}

	if !b.cache.CanScrapeQuery(selectedFilter.Query) {
		b.sendMessage(message.Chat.ID, "⏰ Зачекай трохи перед наступним запитом (захист від бану)")
		return
	}

	b.sendMessage(message.Chat.ID, "🔍 Шукаю оголошення по твоїх фільтрах...")

	olxScraper := scraper.NewOLXScraper()
	searchFilters := scraper.SearchFilters{
		Query:    selectedFilter.Query,
		MinPrice: selectedFilter.MinPrice,
		MaxPrice: selectedFilter.MaxPrice,
		City:     selectedFilter.City,
	}

	listings, err := olxScraper.SearchListings(searchFilters)
	if err != nil {
		log.Printf("Error scraping for filter %d: %v", selectedFilter.ID, err)
		b.sendMessage(message.Chat.ID, "❌ Помилка пошуку на OLX")
		return
	}

	b.cache.CacheSearchResults(cacheKey, listings)

	b.sendSearchResults(message.Chat.ID, selectedFilter.Name, listings)
}

func (b *Bot) sendSearchResults(chatID int64, filterName string, listings []scraper.Listing) {
	if len(listings) == 0 {
		b.sendMessage(chatID, "😔 Оголошень не знайдено")
		return
	}

	text := fmt.Sprintf("📋 **%s** - знайдено %d:\n\n", filterName, len(listings))

	for i, listing := range listings {
		if i >= 5 {
			break
		}
		text += fmt.Sprintf("%d. %s\n💰 %s\n📍 %s\n🔗 %s\n\n",
			i+1, listing.Title, listing.Price, listing.Location, listing.URL)
	}

	if len(listings) > 5 {
		text += fmt.Sprintf("... і ще %d оголошень\n\n", len(listings)-5)
	}

	b.sendMessage(chatID, text)
}
