package controlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sarwanazhar/chatappbackend/database"
	"github.com/sarwanazhar/chatappbackend/libs"
	"github.com/sarwanazhar/chatappbackend/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// this creates a chat with userid
func CreateChat(c *gin.Context) {
	userID := c.GetString("userId")
	fmt.Print(userID)

	user, err := libs.FindUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	chat := model.Chat{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		Title:     "new chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = database.GetCollection("chatApp", "chat").InsertOne(ctx, chat)

	if err != nil {
		// Log the error for debugging
		log.Printf("Error inserting document: %v", err)

		// Check for specific common errors
		if mongo.IsDuplicateKeyError(err) {
			// Handle the case where the document already exists
			c.JSON(http.StatusConflict, gin.H{"error": "chat document already exists"})
			return
		}

		// Check for context cancellation
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "database operation timed out"})
			return
		}

		// Return a generic database error
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert document: %v", err)})
		return
	}

	// Successfully created chat
	c.JSON(http.StatusCreated, gin.H{"message": "chat created", "chatId": chat.ID.Hex()})
}

// this gets all the chat of the user with user id
func GetChat(c *gin.Context) {
	userID := c.GetString("userId")
	fmt.Print(userID)

	user, err := libs.FindUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"user_id": user.ID}

	collection := database.GetCollection("chatApp", "chat")
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "database operation timed out"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to find chats: %v", err)})
		return
	}
	defer cursor.Close(ctx)

	var chats []model.Chat
	if err := cursor.All(ctx, &chats); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to decode chats: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

func CreateMessage(c *gin.Context) {
	// validating user owns this chat first of all

	type Body struct {
		ChatId string `json:"chat_id"`
		Prompt string `json:"prompt"`
	}
	var body Body
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.ChatId == "" || body.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ChatId and Prompt are required fields and cannot be empty"})
		return
	}

	userID := c.GetString("userId")
	fmt.Print(userID)

	user, err := libs.FindUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	objectID, err := primitive.ObjectIDFromHex(body.ChatId)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"_id":     objectID,
		"user_id": user.ID,
	}

	var chat model.Chat

	err = database.GetCollection("chatApp", "chat").FindOne(ctx, filter).Decode(&chat)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Document not found, OR it was found but the user_id did not match.
			// Treat this as a "Not Found" or "Access Denied" error for security.
			c.JSON(404, gin.H{"error": "not found or the chat does not belong to you"})
			return
		}
		c.JSON(500, gin.H{"error": "database query error"})
		return
	}

	// --- NEXT STEP: Create the User Message and Update the Database ---

	// 3. Create the new user message structure
	userMessage := model.Message{
		Role:      "user", // This message always comes from the user
		Content:   body.Prompt,
		CreatedAt: time.Now(),
	}

	// 4. Define the Update Operation
	// We use the $push operator to append the new message to the 'messages' array
	// and $set to update the 'updated_at' timestamp.
	update := bson.M{
		"$push": bson.M{"messages": userMessage},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	// 5. Execute the Update
	// We use the same 'filter' to ensure we update the correct, owned chat.
	updateResult, err := database.GetCollection("chatApp", "chat").UpdateOne(ctx, filter, update)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chat document"})
		return
	}

	if updateResult.ModifiedCount == 0 {
		// Should not happen if the FindOne succeeded, but it's a good defensive check
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat document was not modified"})
		return
	}

}
