package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	ginratelimit "github.com/ljahier/gin-ratelimit"
	"github.com/sarwanazhar/chatappbackend/database"
	"github.com/sarwanazhar/chatappbackend/routes"
)

func init() {
	// Load .env only if running locally (PORT not set)
	if os.Getenv("PORT") == "" {
		err := godotenv.Load()
		if err != nil {
			log.Println("⚠️  No .env file found, continuing...")
		} else {
			log.Println("✅ .env loaded")
		}
	}
}

func main() {
	port := os.Getenv("PORT")
	backendUri := os.Getenv("MONGODB_URI")

	if port == "" {
		port = "8080"
	}
	if backendUri == "" {
		log.Fatal("❌ MONGODB_URI is empty")
	}

	// Connect to MongoDB
	database.ConnectMongo(backendUri)

	r := gin.Default()

	// --- RATE LIMITER SETUP ---
	tb := ginratelimit.NewTokenBucket(30, 1*time.Minute)

	// Middleware that uses userId from context
	r.Use(func(ctx *gin.Context) {
		userId := ctx.GetString("userId")
		ginratelimit.RateLimitByUserId(tb, userId)(ctx)
	})

	// Register routes
	routes.InitRoutes(r)

	address := fmt.Sprintf(":%s", port)
	fmt.Printf("✅ Starting server on %s\n", address)

	if err := r.Run(address); err != nil {
		log.Fatalf("❌ Server failed to run: %v", err)
	}
}
