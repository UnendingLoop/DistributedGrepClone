// Package engine returns configured Server(for main) and Engine(for tests)
package engine

import (
	"log"
	"net/http"

	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/ginext"
)

func NewServerEngine(c *config.Config, mode string) (*http.Server, *ginext.Engine) {
	engine := ginext.New(c.GetString("GIN_MODE"))
	engine.GET("/ping", h.SimplePinger)
	engine.Static("/ui", "./internal/web") // UI админа/юзера - функциональность и контент зависит от роли

	auth := engine.Group("/auth")
	auth.POST("/signup", h.SignUpUser) // регистрация пользователя
	auth.POST("/login", h.LoginUser)   // авторизация

	var items *ginext.RouterGroup
	switch mode {
	case "PROD":
		items = engine.Group("/items", mwauthlog.RequireAuth([]byte(c.GetString("SECRET"))))
	case "TEST":
		items = engine.Group("/items", mwauthlog.RequireAuthTest([]byte(c.GetString("SECRET"))))
	default:
		log.Fatalf("Incorrect mode %q provided to configure routers. Must be 'PROD' or 'TEST'.", mode)
	}

	items.POST("", h.CreateItem)                    // создание Item
	items.PATCH("/:id", h.UpdateItem)               // обновление Item по ID
	items.GET("/:id", h.GetItemByID)                // получение Item по ID
	items.GET("/:id/history", h.GetItemHistoryByID) // получение History товара по его ID
	items.DELETE("/:id", h.DeleteItem)              // удаление Item по ID
	items.GET("", h.GetItemsList)                   // получение всех Item

	items.GET("/history", h.GetItemsHistoryList) // получение History всех товаров - JSON

	items.GET("/csv", h.ExportItemsCSV)                     // CSV: получение всех Item
	items.GET("/:id/history/csv", h.ExportItemIDHistoryCSV) // CSV: получение History товара по его ID
	items.GET("/history/csv", h.ExportItemsHistoryCSV)      // CSV: получение History всех товаров

	return &http.Server{
		Addr:    ":" + c.GetString("APP_PORT"),
		Handler: engine,
	}, engine
}
