package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sarwanazhar/chatappbackend/controlers"
	"github.com/sarwanazhar/chatappbackend/libs"
)

func InitRoutes(router *gin.Engine) {
	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"test": "test",
		})
	})
	Auth(router)
	auth := router.Group("/")
	auth.Use(libs.JWTMiddleware())
	{
		auth.GET("/me", controlers.GetProfiles)
	}
}

func Auth(router *gin.Engine) {
	router.POST("/auth/register", controlers.CreateUser)
	router.POST("/auth/login", controlers.LoginUser)
}
