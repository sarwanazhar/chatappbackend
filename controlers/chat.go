package controlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sarwanazhar/chatappbackend/database"
	"github.com/sarwanazhar/chatappbackend/libs"
	"github.com/sarwanazhar/chatappbackend/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/genai"
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
	opts := options.Find().SetSort(bson.M{
		"created_at": -1, // newest created chat first
	})

	collection := database.GetCollection("chatApp", "chat")
	cursor, err := collection.Find(ctx, filter, opts)
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

	for i := range chats {
		sort.Slice(chats[i].Messages, func(a, b int) bool {
			return chats[i].Messages[a].CreatedAt.Before(
				chats[i].Messages[b].CreatedAt,
			)
		})
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

// this is a really complex function it takes {"chat_id":"", "prompt":""}
// first checks if the chat belongs to user then saves the user prompt then streams ai response then streams that using Server Sent Event(SSE) then saves the ai response
// the history given to ai is only previous 6 messages to save tokens
func CreateMessage(c *gin.Context) {
	// --- Parse request ---
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "ChatId and Prompt are required"})
		return
	}

	// --- Load user + chat document ---
	userID := c.GetString("userId")
	user, err := libs.FindUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	objectID, err := primitive.ObjectIDFromHex(body.ChatId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ChatId"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	filter := bson.M{"_id": objectID, "user_id": user.ID}
	var chat model.Chat
	if err := database.GetCollection("chatApp", "chat").FindOne(ctx, filter).Decode(&chat); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query error"})
		}
		return
	}

	// --- Save the user message to DB ---
	userMessage := model.Message{
		Role:      "user",
		Content:   body.Prompt,
		CreatedAt: time.Now(),
	}
	if chat.Messages == nil {
		chat.Messages = []model.Message{}
	}
	chat.Messages = append(chat.Messages, userMessage)

	if _, err := database.GetCollection("chatApp", "chat").UpdateOne(ctx, filter,
		bson.M{"$push": bson.M{"messages": userMessage}, "$set": bson.M{"updated_at": time.Now()}}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user message"})
		return
	}

	// --- Prepare history for AI ---
	history := chat.Messages
	if len(history) > 6 {
		history = history[len(history)-6:]
	}
	genaiContents := libs.ConvertHistoryToGenaiContent(history)

	// --- Setup SSE ---
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	// --- Initialize the Gemini client ---
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Fprintf(c.Writer, "data: %s\n\n", "{\"error\":\"AI API key not set\"}")
		c.Writer.Flush()
		return
	}

	genaiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		fmt.Fprintf(c.Writer, "data: %s\n\n", "{\"error\":\"Failed to init AI client\"}")
		c.Writer.Flush()
		return
	}

	// --- Stream from Gemini ---
	stream := genaiClient.Models.GenerateContentStream(ctx,
		"gemini-2.5-flash",
		genaiContents,
		nil,
	)

	fullResponse := ""
	for chunk, streamErr := range stream {
		if streamErr != nil {
			// send error chunk
			fmt.Fprintf(c.Writer, "data: %s\n\n", fmt.Sprintf("{\"error\":\"%v\"}", streamErr))
			c.Writer.Flush()
			break
		}

		// Extract text from this chunk
		// See official example — Use the helper .Text() if available
		text := ""
		if len(chunk.Candidates) > 0 {
			for _, part := range chunk.Candidates[0].Content.Parts {
				text += part.Text
			}
		}

		fullResponse += text

		// Send the incremental text over SSE
		eventData := fmt.Sprintf("{\"delta\":%q}", text)
		fmt.Fprintf(c.Writer, "data: %s\n\n", eventData)
		c.Writer.Flush()
	}

	// --- Save final AI message in DB ---
	aiMessage := model.Message{
		Role:      "model",
		Content:   fullResponse,
		CreatedAt: time.Now(),
	}
	_, _ = database.GetCollection("chatApp", "chat").UpdateOne(ctx, filter,
		bson.M{"$push": bson.M{"messages": aiMessage}, "$set": bson.M{"updated_at": time.Now()}})

	// send done event
	fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", "\"end\"")
	c.Writer.Flush()
}
