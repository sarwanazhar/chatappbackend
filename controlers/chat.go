package controlers

import (
	"context"
	"fmt"
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
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/genai"
)

func CreateChat(c *gin.Context) {
	userID := c.GetString("userId")

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create chat"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "chat created",
		"chatId":  chat.ID.Hex(),
	})
}

func DeleteChat(c *gin.Context) {
	type Body struct {
		ChatId string `json:"chat_id"`
	}

	var body Body
	if err := c.ShouldBindJSON(&body); err != nil || body.ChatId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ChatId is required"})
		return
	}

	userID := c.GetString("userId")

	user, err := libs.FindUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	chatObjID, err := primitive.ObjectIDFromHex(body.ChatId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ChatId"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"_id": chatObjID, "user_id": user.ID}

	// Attempt to delete the chat
	res, err := database.GetCollection("chatApp", "chat").DeleteOne(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chat"})
		return
	}

	if res.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found or not owned by user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat deleted successfully"})
}

func GetChat(c *gin.Context) {
	userID := c.GetString("userId")

	user, err := libs.FindUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"user_id": user.ID}
	opts := options.Find().SetSort(bson.M{"created_at": -1})

	cursor, err := database.GetCollection("chatApp", "chat").Find(ctx, filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch chats"})
		return
	}
	defer cursor.Close(ctx)

	var chats []model.Chat
	if err := cursor.All(ctx, &chats); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode chats"})
		return
	}

	for i := range chats {
		sort.Slice(chats[i].Messages, func(a, b int) bool {
			return chats[i].Messages[a].CreatedAt.Before(chats[i].Messages[b].CreatedAt)
		})
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}
func CreateMessage(c *gin.Context) {
	type Body struct {
		ChatId string `json:"chat_id"`
		Prompt string `json:"prompt"`
	}

	var body Body
	if err := c.ShouldBindJSON(&body); err != nil || body.ChatId == "" || body.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ChatId and Prompt are required"})
		return
	}

	userID := c.GetString("userId")
	user, err := libs.FindUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	objID, err := primitive.ObjectIDFromHex(body.ChatId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ChatId"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	filter := bson.M{"_id": objID, "user_id": user.ID}
	var chat model.Chat
	if err := database.GetCollection("chatApp", "chat").FindOne(ctx, filter).Decode(&chat); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		return
	}

	// Save user message
	userMessage := model.Message{Role: "user", Content: body.Prompt, CreatedAt: time.Now()}
	_, _ = database.GetCollection("chatApp", "chat").UpdateOne(ctx, filter, bson.M{
		"$push": bson.M{"messages": userMessage}, "$set": bson.M{"updated_at": time.Now()},
	})

	// Agent decision & optional web search
	decision := libs.DecideSearch(body.Prompt)
	fmt.Println(decision)
	var systemInstruction string
	if decision == "SEARCH" {
		webResult := libs.SearchInternet(body.Prompt)
		if webResult != "" {
			systemInstruction = fmt.Sprintf(
				"You are a helpful assistant. Use the following information from the internet to answer the user's question. Do not make up details and include relevant info only:\n%s",
				webResult,
			)

		}
	}
	fmt.Println(systemInstruction)

	// Build contents + config
	contents, config := libs.BuildGenaiContents(chat.Messages, systemInstruction)

	// Ensure the *current* user prompt is included as the last content so model replies to it.
	// Convert current prompt to genai contents and append
	cur := genai.Text(body.Prompt)
	if len(cur) > 0 {
		contents = append(contents, cur...)
	}

	// SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Fprintf(c.Writer, "data: %s\n\n", `{"error":"AI API key not set"}`)
		return
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		fmt.Fprintf(c.Writer, "data: %s\n\n", `{"error":"Failed to init AI client"}`)
		return
	}

	// Stream from model
	stream := client.Models.GenerateContentStream(ctx, "gemini-2.5-flash-lite", contents, config)

	fullResponse := ""
	for chunk, streamErr := range stream {
		if streamErr != nil {
			// send error event to client
			fmt.Fprintf(c.Writer, "event: error\ndata: %q\n\n", streamErr.Error())
			c.Writer.Flush()
			break
		}
		text := ""
		if len(chunk.Candidates) > 0 {
			for _, p := range chunk.Candidates[0].Content.Parts {
				text += p.Text
			}
		}
		fullResponse += text
		fmt.Fprintf(c.Writer, "data: %s\n\n", fmt.Sprintf(`{"delta":%q}`, text))
		c.Writer.Flush()
	}

	// Save AI response
	aiMessage := model.Message{Role: "model", Content: fullResponse, CreatedAt: time.Now()}
	_, _ = database.GetCollection("chatApp", "chat").UpdateOne(ctx, filter, bson.M{
		"$push": bson.M{"messages": aiMessage}, "$set": bson.M{"updated_at": time.Now()},
	})

	// done
	fmt.Fprintf(c.Writer, "event: done\ndata: \"end\"\n\n")
	c.Writer.Flush()
}
