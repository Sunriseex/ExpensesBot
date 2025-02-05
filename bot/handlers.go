package bot

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"github.com/sunriseex/tgbot-money/cache"
	"github.com/sunriseex/tgbot-money/db"
	"github.com/sunriseex/tgbot-money/models"
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
	case "start", "help":
		response.Text = getHelpMessage()
	case "add":
		handleAddCommand(msg.CommandArguments(), userID, &response)
	case "list":
		handleListCommand(userID, &response)
	case "stats_category":
		handleStatsCategoryCommand(msg.CommandArguments(), userID, &response)
	case "top_categories":
		handleTopCategoriesCommand(msg.CommandArguments(), userID, &response)
	case "export":
		handleExportCommand(userID, &response)
	case "clear_cache":
		handleClearCacheCommand(userID, &response)
	case "edit":
		handleEditCommand(msg.CommandArguments(), userID, &response)
	case "delete":
		handleDeleteCommand(msg.CommandArguments(), &response)
	default:
		response.Text = "Неизвестная команда. Используйте /help для списка команд."
	}

	if _, err := Bot.Send(response); err != nil {
		logrus.WithError(err).Error("Ошибка отправки сообщения")
	}
}

func getHelpMessage() string {
	return "📊 Бот для учёта расходов\n\n" +
		"Команды:\n" +
		"/add <сумма> <категория> - добавить расход\n" +
		"/list - последние 10 записей\n" +
		"/stats_category <категория> [week/month] - статистика по категории\n" +
		"/top_categories [week/month] - топ категорий по расходам\n" +
		"/export - экспорт расходов в CSV\n" +
		"/clear_cache - очистить кеш (для отладки)\n" +
		"/edit <id> <сумма> [категория] - изменить запись\n" +
		"/delete <id> - удалить запись\n" +
		"/help - справка"
}

// Команда /add: добавление расхода
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
		logrus.WithError(err).Error("DB Create error")
		return
	}

	// Инвалидация кеша для команд /list и /stats
	cacheKeys := []string{
		fmt.Sprintf("user:%d:list", userID),
		fmt.Sprintf("user:%d:stats", userID),
	}
	for _, key := range cacheKeys {
		if err := cache.Client.Del(cache.Ctx, key).Err(); err != nil {
			logrus.WithField("key", key).WithError(err).Warn("Ошибка удаления кеша")
		}
	}

	response.Text = fmt.Sprintf("✅ Добавлено: %.2f руб. в категорию «%s»", amount, category)
}

// Команда /list: вывод последних 10 расходов
func handleListCommand(userID int64, response *tgbotapi.MessageConfig) {
	cacheKey := fmt.Sprintf("user:%d:list", userID)
	if cached, err := cache.Client.Get(cache.Ctx, cacheKey).Result(); err == nil {
		response.Text = cached
		return
	} else if err.Error() != "redis: nil" {
		logrus.WithError(err).Warn("Ошибка получения кеша")
	}

	var expenses []models.Expense
	if err := db.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(10).Find(&expenses).Error; err != nil {
		response.Text = "Ошибка получения данных"
		logrus.WithError(err).Error("DB error in list")
		return
	}

	if len(expenses) == 0 {
		response.Text = "🗒 Список расходов пуст"
		return
	}

	var builder strings.Builder
	builder.WriteString("📝 Последние расходы:\n\n")
	for _, e := range expenses {
		builder.WriteString(fmt.Sprintf("[%d] %s - %.2f руб. (%s)\n",
			e.ID, e.CreatedAt.Format("02.01 15:04"), e.Amount, e.Category))
	}
	output := builder.String()
	response.Text = output

	if err := cache.Client.Set(cache.Ctx, cacheKey, output, 10*time.Minute).Err(); err != nil {
		logrus.WithError(err).Warn("Ошибка установки кеша")
	}
}

// Команда /stats_category: статистика по конкретной категории
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
		logrus.WithError(err).Error("DB error in stats_category")
		return
	}

	if total == 0 {
		response.Text = fmt.Sprintf("📊 Нет данных по категории «%s» за выбранный период", category)
		return
	}

	response.Text = fmt.Sprintf("📊 Статистика по категории «%s» за %s:\n%.2f руб.", category, period, total)
}

// Команда /top_categories: топ категорий по расходам
func handleTopCategoriesCommand(args string, userID int64, response *tgbotapi.MessageConfig) {
	period := "week"
	if args != "" {
		period = args
	}

	now := time.Now()
	start := now.AddDate(0, 0, -7)
	if period == "month" {
		start = now.AddDate(0, -1, 0)
	}

	type CategoryStat struct {
		Category string
		Total    float64
	}
	var stats []CategoryStat
	if err := db.DB.Model(&models.Expense{}).
		Select("category, sum(amount) as total").
		Where("user_id = ? AND created_at >= ?", userID, start).
		Group("category").
		Order("total desc").
		Scan(&stats).Error; err != nil {
		response.Text = "Ошибка получения статистики"
		logrus.WithError(err).Error("DB error in top_categories")
		return
	}

	if len(stats) == 0 {
		response.Text = fmt.Sprintf("📊 Нет данных за %s", period)
		return
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("🏆 Топ категорий за %s:\n\n", period))
	for _, stat := range stats {
		builder.WriteString(fmt.Sprintf("%s: %.2f руб.\n", stat.Category, stat.Total))
	}
	response.Text = builder.String()
}

// Команда /export: экспорт расходов в CSV
func handleExportCommand(userID int64, response *tgbotapi.MessageConfig) {
	var expenses []models.Expense
	if err := db.DB.Where("user_id = ?", userID).Order("created_at asc").Find(&expenses).Error; err != nil {
		response.Text = "Ошибка получения данных для экспорта"
		logrus.WithError(err).Error("DB error in export")
		return
	}

	if len(expenses) == 0 {
		response.Text = "Нет расходов для экспорта"
		return
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.Write([]string{"ID", "Дата", "Сумма", "Категория"}); err != nil {
		logrus.WithError(err).Error("Ошибка при создании CSV writer")
		return
	}

	for _, exp := range expenses {
		if err := writer.Write([]string{
			strconv.Itoa(int(exp.ID)),
			exp.CreatedAt.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.2f", exp.Amount),
			exp.Category,
		}); err != nil {
			logrus.WithError(err).Error("Ошибка при записи в CSV")
			return
		}
	}
	writer.Flush()
	response.Text = "Экспорт расходов:\n" + buf.String()
}

// Команда /clear_cache: очистка кеша пользователя
func handleClearCacheCommand(userID int64, response *tgbotapi.MessageConfig) {
	pattern := fmt.Sprintf("user:%d:*", userID)
	iter := cache.Client.Scan(cache.Ctx, 0, pattern, 0).Iterator()
	for iter.Next(cache.Ctx) {
		if err := cache.Client.Del(cache.Ctx, iter.Val()).Err(); err != nil {
			logrus.WithField("key", iter.Val()).WithError(err).Error("Ошибка удаления кеша")
		}
	}
	if err := iter.Err(); err != nil {
		logrus.WithError(err).Error("Ошибка при сканировании кеша")
	}
	response.Text = "Кеш очищен"
}

// Команда /edit: изменение записи
// Формат: /edit <id> <сумма> [категория]
func handleEditCommand(args string, userID int64, response *tgbotapi.MessageConfig) {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		response.Text = "Формат: /edit <id> <сумма> [категория]"
		return
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		response.Text = "Неверный формат ID"
		return
	}
	amount, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		response.Text = "Неверный формат суммы"
		return
	}
	var expense models.Expense
	if err := db.DB.First(&expense, id).Error; err != nil {
		response.Text = "Запись не найдена"
		return
	}
	if expense.UserID != userID {
		response.Text = "Нет доступа для изменения этой записи"
		return
	}
	expense.Amount = amount
	if len(parts) > 2 {
		expense.Category = strings.Join(parts[2:], " ")
	}
	if err := db.DB.Save(&expense).Error; err != nil {
		response.Text = "Ошибка обновления записи"
		logrus.WithError(err).Error("DB update error in edit")
		return
	}

	// Инвалидация кеша (для /list и /stats)
	cacheKeys := []string{
		fmt.Sprintf("user:%d:list", userID),
		fmt.Sprintf("user:%d:stats", userID),
	}
	for _, key := range cacheKeys {
		if err := cache.Client.Del(cache.Ctx, key).Err(); err != nil {
			logrus.WithField("key", key).WithError(err).Warn("Ошибка удаления кеша в edit")
		}
	}

	response.Text = "✅ Запись успешно обновлена"
}

// Команда /delete: удаление записи
// Формат: /delete <id>
func handleDeleteCommand(arg string, response *tgbotapi.MessageConfig) {
	id, err := strconv.Atoi(arg)
	if err != nil {
		response.Text = "Формат: /delete <id>"
		return
	}
	result := db.DB.Delete(&models.Expense{}, id)
	if result.Error != nil {
		response.Text = "Ошибка при удалении записи"
		logrus.WithError(result.Error).Error("DB delete error")
		return
	}
	if result.RowsAffected == 0 {
		response.Text = "❌ Запись не найдена"
	} else {
		response.Text = "✅ Запись удалена"
	}
}
