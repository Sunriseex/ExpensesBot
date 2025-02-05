package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sunriseex/tgbot-money/config"
)

var Bot *tgbotapi.BotAPI

// InitBot инициализирует Telegram-бота
func InitBot(conf config.Config) {
	var err error
	Bot, err = tgbotapi.NewBotAPI(conf.TelegramToken)
	if err != nil {
		log.Fatal("Ошибка создания Telegram-бота:", err)
	}
}
