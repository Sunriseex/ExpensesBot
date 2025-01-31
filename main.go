package main

import (
	"github.com/sunriseex/tgboy-money/bot"
	"github.com/sunriseex/tgboy-money/cache"
	"github.com/sunriseex/tgboy-money/config"
	"github.com/sunriseex/tgboy-money/db"
	"github.com/sunriseex/tgboy-money/web"
)

func main() {
	// Загрузка конфигурации
	conf := config.LoadConfig()

	// Инициализация компонентов
	db.Init(conf)
	cache.Init(conf)
	bot.InitBot(conf)

	// Запуск веб-сервера в горутине
	go web.StartWebServer(conf.WebPort)

	// Запуск Telegram-бота (блокирующий вызов)
	bot.StartBot()
}
