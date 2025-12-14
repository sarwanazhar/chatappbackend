package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sarwanazhar/chatappbackend/database"
	"github.com/sarwanazhar/chatappbackend/routes"
)

func main() {
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

	database.ConnectMongo(backendUri)

	r := gin.Default()

	routes.InitRoutes(r)

	address := fmt.Sprintf(":%s", port)

	fmt.Printf("✅ Starting server on address %s\n", address)
	err = r.Run(address)

	if err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}

}
