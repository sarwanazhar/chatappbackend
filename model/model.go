package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Email     string             `json:"email" bson:"email"`
	Password  string             `json:"password" bson:"password"`
	CreatedAt time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updated_at"`
}

type Message struct {
	Role      string    `json:"role" bson:"role"`       // "user" | "model"
	Content   string    `json:"content" bson:"content"` // For simplicity, keep it string here
	CreatedAt time.Time `json:"createdAt" bson:"created_at"`
}

// The Chat model now contains the messages array.
type Chat struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"userId" bson:"user_id"`
	Title     string             `json:"title" bson:"title"`
	Messages  []Message          `json:"messages" bson:"messages,omitempty"` // <- THE KEY CHANGE
	CreatedAt time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updated_at"`
}
