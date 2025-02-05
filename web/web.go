package web

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sunriseex/tgbot-money/db"
	"github.com/sunriseex/tgbot-money/models"
)

func StartWebServer(port string) {
	router := gin.Default()

	// Загружаем HTML-шаблоны
	router.LoadHTMLGlob("templates/*")

	// Эндпоинты:
	router.GET("/stats/:userID", statsHandler)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Не удалось запустить веб-сервер: %v", err)
	}
}

func statsHandler(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Неверный формат userID")
		return
	}

	var expenses []models.Expense
	if err := db.DB.Where("user_id = ?", userID).Find(&expenses).Error; err != nil {
		c.String(http.StatusInternalServerError, "Ошибка получения данных")
		return
	}

	c.HTML(http.StatusOK, "stats.html", gin.H{"expenses": expenses})
}
