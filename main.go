package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	ginratelimit "github.com/ljahier/gin-ratelimit" // rate limiting middleware
	"github.com/sarwanazhar/chatappbackend/database"
	"github.com/sarwanazhar/chatappbackend/routes"
)

func main() {
	// load .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	port := os.Getenv("PORT")
	backendUri := os.Getenv("MONGODB_URI")

	if port == "" {
		port = "8080"
	}
	if backendUri == "" {
		log.Fatal("backend uri is empty")
	}

	// connect DB
	database.ConnectMongo(backendUri)

	r := gin.Default()

	// --- RATE LIMITER SETUP ---
	// 30 requests per minute per user (customize limit & interval)
	tb := ginratelimit.NewTokenBucket(30, 1*time.Minute)

	// Middleware that uses userId from context
	r.Use(func(ctx *gin.Context) {
		// Extract userId, e.g., from auth (must be set before this)
		userId := ctx.GetString("userId")
		// Apply rate limit per user
		ginratelimit.RateLimitByUserId(tb, userId)(ctx)
	})

	// Register routes (including CreateMessage)
	routes.InitRoutes(r)

	address := fmt.Sprintf(":%s", port)
	fmt.Printf("✅ Starting server on address %s\n", address)

	if err := r.Run(address); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
