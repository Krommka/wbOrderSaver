package http

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"log/slog"
	"net/http"
	_ "wb_l0/docs"
	"wb_l0/internal/usecase"
)

func SetupRouter(uc *usecase.OrderUsecase, log *slog.Logger) *gin.Engine {
	router := gin.Default()

	router.Static("/static", "./web")
	router.LoadHTMLGlob("web/*.html")

	router.Use(gin.Recovery())

	orderHandler := NewOrderHandler(uc, log)

	router.GET("/health", orderHandler.HealthCheck)
	router.GET("/order/:order_uid", orderHandler.GetOrderByUID)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	router.GET("/result.html", func(c *gin.Context) {
		c.HTML(http.StatusOK, "result.html", nil)
	})

	return router
}
