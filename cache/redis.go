package cache

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/sunriseex/tgbot-money/config"
)

var (
	Client *redis.Client
	Ctx    = context.Background()
)

// Init инициализирует Redis-клиент
func Init(conf config.Config) {
	Client = redis.NewClient(&redis.Options{
		Addr: conf.RedisURL,
		Password: "",
		DB:       0,
	})

	if _, err := Client.Ping(Ctx).Result(); err != nil {
		log.Fatal("Ошибка подключения к Redis:", err)
	}
}
