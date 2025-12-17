package controlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sarwanazhar/chatappbackend/database"
	"github.com/sarwanazhar/chatappbackend/libs"
	"github.com/sarwanazhar/chatappbackend/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// this creates a simple user in mongo db
// its a post needs json {"email": "", "password": ""}
func CreateUser(c *gin.Context) {
	type Body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var body Body
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	EmailExists, err := libs.SearchForExistingEmail(body.Email)

	if err != nil {
		log.Printf("Failed to check email existence for %s: %v", body.Email, err)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error. Please try again later."})
		return // <<-- FIX: use simple return
	}

	if EmailExists {
		c.JSON(http.StatusConflict, gin.H{"error": "This email address is already registered."})
		return
	}

	hashedPassword, err := libs.HashPassword(body.Password)

	if err != nil {
		log.Fatal(err)
	}

	contxt, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user := &model.User{
		Email:    body.Email,
		Password: hashedPassword,
	}

	newId, err := libs.CreateUser(contxt, user)
	if err != nil {
		log.Printf("Failed to create user %s: %v", body.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error. Please try again later."})
		return
	}
	fmt.Print("new user created id:")
	fmt.Println(newId)

	chat := model.Chat{
		ID:        primitive.NewObjectID(),
		UserID:    newId,
		Title:     "Chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = database.GetCollection("chatApp", "chat").InsertOne(ctx, chat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create chat"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

func LoginUser(c *gin.Context) {
	type Body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var body Body
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	foundUser, err := libs.FindUserByEmail(body.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid email or password",
		})
		return
	}

	// Log the successful result (avoid exposing sensitive fields in responses)
	fmt.Printf("User Found!\n")
	fmt.Printf("ID: %s\n", foundUser.ID.Hex())
	fmt.Printf("Email: %s\n", foundUser.Email)
	// ... print other fields

	isPasswordCorrect := libs.CheckPasswordHash(body.Password, foundUser.Password)

	if !isPasswordCorrect {
		c.JSON(401, gin.H{"error": "Invalid email or password"})
		return
	}

	// generate token
	token, err := libs.GenerateJWT(foundUser.ID.Hex())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not generate token",
		})
		return
	}

	// Return token to client
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    foundUser.ID.Hex(),
			"email": foundUser.Email,
		},
	})

}

// Protected routes

func GetProfiles(c *gin.Context) {
	userID := c.GetString("userId")
	fmt.Print(userID)

	user, err := libs.FindUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":    user.ID.Hex(),
		"email": user.Email,
	})

}
