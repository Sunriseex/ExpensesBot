package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config хранит переменные окружения
type Config struct {
	DatabaseURL   string
	RedisURL      string
	TelegramToken string
	WebPort       string
}

// LoadConfig загружает конфигурацию из .env файла или системных переменных
func LoadConfig() Config {
	// Попытка загрузки .env файла
	if err := godotenv.Load(); err != nil {
		log.Println("Нет файла .env, используем системные переменные")
	}

	conf := Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RedisURL:      os.Getenv("REDIS_URL"),
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		WebPort:       os.Getenv("WEB_PORT"),
	}

	if conf.DatabaseURL == "" || conf.RedisURL == "" || conf.TelegramToken == "" || conf.WebPort == "" {
		log.Fatal("Одна или несколько обязательных переменных окружения не заданы")
	}

	return conf
}
