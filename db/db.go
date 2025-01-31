package db

import (
	"log"

	"github.com/sunriseex/tgboy-money/config"
	"github.com/sunriseex/tgboy-money/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init инициализирует подключение к БД и выполняет миграции
func Init(conf config.Config) {
	var err error
	DB, err = gorm.Open(postgres.Open(conf.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}

	if err := DB.AutoMigrate(&models.Expense{}); err != nil {
		log.Fatal("Ошибка миграции:", err)
	}
}
