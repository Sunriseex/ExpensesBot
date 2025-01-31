package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sunriseex/tgboy-money/cache"
	"github.com/sunriseex/tgboy-money/db"
	"github.com/sunriseex/tgboy-money/models"
)

// StartBot запускает цикл обработки обновлений
func StartBot() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := Bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		go handleMessage(update.Message)
	}
}

func handleMessage(msg *tgbotapi.Message) {
	userID := msg.From.ID
	chatID := msg.Chat.ID
	response := tgbotapi.NewMessage(chatID, "")

	switch msg.Command() {
	case "start":
		response.Text = "📊 Бот для учета расходов\n\n" +
			"Команды:\n" +
			"/add <сумма> <категория> - добавить расход\n" +
			"/list - последние 10 записей\n" +
			// можно добавить остальные команды
			"/stats_category <категория> [week/month] - статистика по категории"
	case "add":
		handleAddCommand(msg.CommandArguments(), userID, &response)
	case "list":
		handleListCommand(userID, &response)
	case "stats_category":
		handleStatsCategoryCommand(msg.CommandArguments(), userID, &response)
	default:
		response.Text = "Неизвестная команда. Используйте /start"
	}

	if _, err := Bot.Send(response); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

// Пример функции обработки команды /add
func handleAddCommand(args string, userID int64, response *tgbotapi.MessageConfig) {
	parts := strings.Fields(args)
	if len(parts) < 1 {
		response.Text = "Формат: /add 500 еда"
		return
	}

	amount, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		response.Text = "Ошибка формата суммы"
		return
	}

	category := "другое"
	if len(parts) > 1 {
		category = strings.Join(parts[1:], " ")
	}

	expense := models.Expense{
		UserID:    userID,
		Amount:    amount,
		Category:  category,
		CreatedAt: time.Now(),
	}

	if err := db.DB.Create(&expense).Error; err != nil {
		response.Text = "Ошибка сохранения расхода"
		log.Printf("DB Create error: %v", err)
		return
	}

	// Сброс кеша для пользователя
	cacheKey := fmt.Sprintf("user:%d:expenses", userID)
	if err := cache.Client.Del(cache.Ctx, cacheKey).Err(); err != nil {
		log.Printf("Ошибка удаления кеша: %v", err)
	}

	response.Text = fmt.Sprintf("✅ Добавлено: %.2f руб. в категорию «%s»", amount, category)
}

// Пример функции обработки команды /list
func handleListCommand(userID int64, response *tgbotapi.MessageConfig) {
	cacheKey := fmt.Sprintf("user:%d:expenses", userID)
	cached, err := cache.Client.Get(cache.Ctx, cacheKey).Result()
	if err == nil {
		response.Text = cached
		return
	}

	var expenses []models.Expense
	err = db.DB.Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(10).
		Find(&expenses).Error
	if err != nil {
		response.Text = "Ошибка получения данных"
		log.Printf("DB error: %v", err)
		return
	}

	if len(expenses) == 0 {
		response.Text = "🗒 Список расходов пуст"
		return
	}

	var builder strings.Builder
	builder.WriteString("📝 Последние расходы:\n\n")
	for _, e := range expenses {
		builder.WriteString(fmt.Sprintf(
			"[%d] %s - %.2f руб. (%s)\n",
			e.ID,
			e.CreatedAt.Format("02.01 15:04"),
			e.Amount,
			e.Category,
		))
	}

	response.Text = builder.String()
	if err := cache.Client.Set(cache.Ctx, cacheKey, response.Text, 10*time.Minute).Err(); err != nil {
		log.Printf("Ошибка установки кеша: %v", err)
	}
}

// Пример функции обработки команды /stats_category
func handleStatsCategoryCommand(args string, userID int64, response *tgbotapi.MessageConfig) {
	parts := strings.Fields(args)
	if len(parts) < 1 {
		response.Text = "Формат: /stats_category <категория> [week/month]"
		return
	}

	category := parts[0]
	period := "week"
	if len(parts) >= 2 {
		period = parts[1]
	}

	now := time.Now()
	start := now.AddDate(0, 0, -7)
	if period == "month" {
		start = now.AddDate(0, -1, 0)
	}

	var total float64
	err := db.DB.Model(&models.Expense{}).
		Where("user_id = ? AND category = ? AND created_at >= ?", userID, category, start).
		Select("sum(amount)").
		Row().Scan(&total)
	if err != nil {
		response.Text = "Ошибка получения статистики по категории"
		log.Printf("DB error in stats_category: %v", err)
		return
	}

	if total == 0 {
		response.Text = fmt.Sprintf("📊 Нет данных по категории «%s» за выбранный период", category)
		return
	}

	response.Text = fmt.Sprintf("📊 Статистика по категории «%s» за %s:\n%.2f руб.", category, period, total)
}
