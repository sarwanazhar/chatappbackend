package libs

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sarwanazhar/chatappbackend/database"
	"github.com/sarwanazhar/chatappbackend/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

const dbName = "chatApp"
const userCollection = "users"

func getUserCollection() *mongo.Collection {
	return database.GetCollection(dbName, userCollection)
}

func CreateUser(ctx context.Context, user *model.User) (primitive.ObjectID, error) {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := getUserCollection().InsertOne(ctx, user)
	return user.ID, err
}

func SearchForExistingEmail(email string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Define the Filter:
	// This creates the MongoDB query: { "email": "the_email_to_search" }
	filter := bson.D{
		bson.E{Key: "email", Value: email},
	}

	var user model.User

	// 2. Execute FindOne:
	// We use FindOne, as we only expect (or care about) one match for a unique email.
	err := database.GetCollection("chatApp", "users").FindOne(ctx, filter).Decode(&user)

	switch err {
	case nil:
		// Document found successfully
		return true, nil
	case mongo.ErrNoDocuments:
		// Document not found
		//
		return false, nil
	default:
		// Other error occurred (e.g., connection issue, server error)
		return false, fmt.Errorf("database error during email search: %w", err)
	}
}

func HashPassword(password string) (string, error) {
	// We use DefaultCost, which is 10 as of now,
	// but you can increase it for more security (e.g., 12 or 14).
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	// CompareHashAndPassword handles the hashing of the 'password'
	// internally and compares it to the 'hash'.
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func FindUserByEmail(email string) (*model.User, error) {
	// 1. Create a context for the operation (e.g., with a timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 2. Define the filter to search by. We want a document where 'email' matches the input email.
	// The filter is a BSON document.
	filter := bson.M{"email": email}

	// 3. Prepare an empty User struct to hold the result
	var user model.User

	// 4. Call FindOne to execute the query
	// The result from FindOne is a *mongo.SingleResult
	result := database.GetCollection("chatApp", "users").FindOne(ctx, filter)

	// 5. Check for errors
	if result.Err() != nil {
		// If the error is mongo.ErrNoDocuments, the user was not found
		if result.Err() == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user with email '%s' not found", email)
		}
		// Handle other potential errors (connection, server, etc.)
		return nil, fmt.Errorf("error finding user: %w", result.Err())
	}

	// 6. Decode the result into the User struct
	err := result.Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("error decoding user document: %w", err)
	}

	// 7. Return the found user
	return &user, nil
}

// Secret key (store in .env in production)
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// GenerateJWT creates a signed token for a user
func GenerateJWT(userID string) (string, error) {
	// Define claims
	claims := jwt.MapClaims{
		"userId": userID, // store user ID
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	return token.SignedString(jwtSecret)
}
func FindUserByID(id string) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user id format")
	}

	filter := bson.M{"_id": objID}

	var user model.User
	result := getUserCollection().FindOne(ctx, filter)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user with id '%s' not found", id)
		}
		return nil, fmt.Errorf("error finding user: %w", result.Err())
	}

	if err := result.Decode(&user); err != nil {
		return nil, fmt.Errorf("error decoding user document: %w", err)
	}

	return &user, nil
}
