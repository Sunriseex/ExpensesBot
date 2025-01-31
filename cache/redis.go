package cache

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/sunriseex/tgboy-money/config"
)

var (
	Client *redis.Client
	Ctx    = context.Background()
)

// Init инициализирует Redis-клиент
func Init(conf config.Config) {
	Client = redis.NewClient(&redis.Options{
		Addr: conf.RedisURL,
		// Если нужно, можно добавить аутентификацию и номер БД
		Password: "",
		DB:       0,
	})

	if _, err := Client.Ping(Ctx).Result(); err != nil {
		log.Fatal("Ошибка подключения к Redis:", err)
	}
}
