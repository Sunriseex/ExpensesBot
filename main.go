package main

import (
	"github.com/sirupsen/logrus"
	"github.com/sunriseex/tgbot-money/bot"
	"github.com/sunriseex/tgbot-money/cache"
	"github.com/sunriseex/tgbot-money/config"
	"github.com/sunriseex/tgbot-money/db"
	"github.com/sunriseex/tgbot-money/web"
)

func main() {
	// Настраиваем Logrus
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.SetLevel(logrus.InfoLevel)
	logrus.Info("Запуск приложения")

	// Загрузка конфигурации
	conf := config.LoadConfig()

	// Инициализация компонентов
	db.Init(conf)
	cache.Init(conf)
	bot.InitBot(conf)

	// Запуск веб-сервера в отдельной горутине
	go web.StartWebServer(conf.WebPort)

	// Запуск Telegram-бота (блокирующий вызов)
	bot.StartBot()
}
