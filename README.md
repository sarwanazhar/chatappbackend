# ChatApp Backend

A real-time chat application backend built with Go, MongoDB, and Gin framework. This application provides user authentication, chat management, and AI-powered conversation capabilities using Google's Gemini API with free internet search integration via DuckDuckGo.

## Table of Contents

- [Project Overview](#project-overview)
- [Technology Stack](#technology-stack)
- [Project Structure](#project-structure)
- [Authentication](#authentication)
- [API Routes](#api-routes)
- [Environment Variables](#environment-variables)
- [Installation](#installation)
- [Running the Application](#running-the-application)

## Project Overview

This is a backend API for a chat application that allows users to:
- Register and authenticate with email/password
- Create multiple chat sessions
- Send messages in real-time using Server-Sent Events (SSE)
- Receive AI-generated responses from Google Gemini API
- Access free internet search capabilities via DuckDuckGo for up-to-date information
- View chat history with intelligent token management

The application uses JWT tokens for authentication, MongoDB for data persistence, and streams AI responses in real-time to provide a seamless chat experience.

## Technology Stack

### Go Packages Used

**Core Framework & HTTP:**
- `github.com/gin-gonic/gin` - HTTP web framework for routing and middleware
- `github.com/ljahier/gin-ratelimit` - Rate limiting
- `github.com/golang-jwt/jwt/v5` - JWT token creation and validation
- `github.com/gorilla/websocket` - WebSocket support (available but not currently used)

**Database:**
- `go.mongodb.org/mongo-driver/v2` - MongoDB driver for Go (v2.4.1)
- `go.mongodb.org/mongo-driver` - Legacy MongoDB driver (v1.17.6)

**AI Integration:**
- `google.golang.org/genai` - Google Gemini AI API client
- `cloud.google.com/go` - Google Cloud client libraries

**Internet Search (Free):**
- `github.com/PuerkitoBio/goquery` - HTML parsing for web scraping
- Built-in DuckDuckGo search integration (no API key required)

**Security & Utilities:**
- `golang.org/x/crypto/bcrypt` - Password hashing
- `github.com/joho/godotenv` - Environment variable management
- `github.com/go-playground/validator/v10` - Request validation

**Additional Dependencies:**
- `github.com/goccy/go-json` - Fast JSON serialization
- `github.com/gabriel-vasile/mimetype` - MIME type detection
- Various Google Cloud and OpenTelemetry packages for monitoring

## Project Structure

```
chatApp/
├── controlers/          # HTTP controllers/handlers
│   ├── chat.go         # Chat-related operations
│   └── user.go         # User authentication and profile
├── database/           # Database connection and utilities
│   └── mongo.go        # MongoDB connection setup
├── libs/               # Helper functions and middleware
│   ├── middleware.go   # JWT authentication middleware
│   ├── user.go         # User-related database operations
│   ├── ConvertHistoryToGenaiContent.go # AI message formatting
│   ├── DecideSearch.go # AI-powered search decision logic
│   └── DuckDuckGoSearch.go # Free internet search implementation
├── model/              # Data models
│   └── model.go        # User, Message, and Chat models
├── routes/             # Route definitions
│   └── routes.go       # API route configuration
├── go.mod              # Go module dependencies
├── go.sum              # Dependency checksums
├── main.go             # Application entry point
└── README.md           # This documentation
```

## Authentication

The application uses JWT (JSON Web Tokens) for authentication with the following flow:

### Registration Flow
1. Client sends `POST /auth/register` with email and password
2. Server validates email uniqueness
3. Password is hashed using bcrypt
4. User is created in MongoDB
5. Returns success message

### Login Flow
1. Client sends `POST /auth/login` with email and password
2. Server verifies email exists and password matches
3. JWT token is generated with user ID and 24-hour expiration
4. Token is returned to client for subsequent requests

### Protected Routes
- All protected routes require `Authorization: Bearer <token>` header
- JWT middleware validates token and extracts user ID
- User ID is stored in request context for handler use

## API Routes (rate limit = 30 request per minute)

### Public Routes (No Authentication Required)

#### 1. Health Check
```
GET /
```
**Description:** Simple health check endpoint

**Response:**
```json
{
  "test": "test"
}
```

#### 2. User Registration
```
POST /auth/register
```
**Description:** Create a new user account

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Success Response (201):**
```json
{
  "message": "User created successfully"
}
```

**Error Responses:**
- `400` - Invalid JSON or missing fields
- `409` - Email already exists
- `500` - Server error

#### 3. User Login
```
POST /auth/login
```
**Description:** Authenticate user and receive JWT token

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Success Response (200):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "60d5ecb74f4c8a1234567890",
    "email": "user@example.com"
  }
}
```

**Error Responses:**
- `400` - Invalid JSON or missing fields
- `401` - Invalid email or password
- `500` - Server error

### Protected Routes (Require JWT Authentication)

#### 4. Get User Profile
```
GET /me
```
**Headers:** `Authorization: Bearer <token>`

**Description:** Get current user's profile information

**Success Response (200):**
```json
{
  "id": "60d5ecb74f4c8a1234567890",
  "email": "user@example.com"
}
```

**Error Responses:**
- `401` - Missing or invalid token
- `404` - User not found
- `500` - Server error

#### 5. Create New Chat
```
POST /chat/create
```
**Headers:** `Authorization: Bearer <token>`

**Description:** Create a new chat session for the authenticated user

**Request Body:** None required

**Success Response (201):**
```json
{
  "message": "chat created",
  "chatId": "60d5ecb74f4c8a1234567891"
}
```

**Error Responses:**
- `401` - Missing or invalid token
- `404` - User not found
- `500` - Server error

#### 6. Get All User Chats
```
GET /chat/getall
```
**Headers:** `Authorization: Bearer <token>`

**Description:** Retrieve all chat sessions for the authenticated user, ordered by creation date (newest first)

**Success Response (200):**
```json
{
  "chats": [
    {
      "_id": "60d5ecb74f4c8a1234567891",
      "userId": "60d5ecb74f4c8a1234567890",
      "title": "new chat",
      "messages": [
        {
          "role": "user",
          "content": "Hello, how are you?",
          "createdAt": "2024-01-01T12:00:00Z"
        },
        {
          "role": "model",
          "content": "I'm doing well, thank you!",
          "createdAt": "2024-01-01T12:00:01Z"
        }
      ],
      "createdAt": "2024-01-01T12:00:00Z",
      "updatedAt": "2024-01-01T12:00:01Z"
    }
  ]
}
```

**Error Responses:**
- `401` - Missing or invalid token
- `404` - User not found
- `500` - Server error

#### 7. Send Message (Streaming)
```
POST /chat/message
```
**Headers:** `Authorization: Bearer <token>`

**Description:** Send a message to a chat and receive AI responses via Server-Sent Events (SSE). This endpoint:
- Validates chat ownership
- Saves user message to database
- Automatically determines if internet search is needed using AI routing
- Performs free DuckDuckGo search when required for up-to-date information
- Streams AI response in real-time with search context when applicable
- Saves AI response to database
- Maintains conversation history (last 6 messages for token efficiency)

**Request Body:**
```json
{
  "chat_id": "60d5ecb74f4c8a1234567891",
  "prompt": "What is the capital of France?"
}
```

**SSE Response Format:**
```
data: {"delta":"Paris"}

data: {"delta":" is"}

data: {"delta":" the"}

data: {"delta":" capital"}

event: done
data: "end"
```

**Error Responses:**
- `400` - Missing chat_id or prompt
- `401` - Missing or invalid token
- `404` - Chat not found or doesn't belong to user
- `500` - Server error or AI API key not set

**SSE Error Format:**
```
data: {"error":"AI API key not set"}
```

#### 8. Delete Chat
```
POST /chat/delete
```
**Headers:** `Authorization: Bearer <token>`
**Description:** Deletes the chat

**Request Body:**
```json
{
  "chat_id": "60d5ecb74f4c8a1234567891",
}
```
**Success Response (200):**
```json
{
  "message": "Chat deleted successfully",
}
```
**Error Responses:**
- `401` - Missing or invalid token
- `404` - User not found
- `500` - Server error

## Environment Variables

Create a `.env` file in the project root with the following variables:

```env
# Server Configuration
PORT=8080

# Database Configuration
MONGODB_URI=mongodb://localhost:27017/chatApp

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-here

# AI Configuration
GEMINI_API_KEY=your-google-gemini-api-key-here
```

### Environment Variables Details

- **PORT** (optional, default: 8080): The port the server will listen on
- **MONGODB_URI** (required): MongoDB connection string with database name
- **JWT_SECRET** (required): Secret key for signing JWT tokens (use a strong, random string)
- **GEMINI_API_KEY** (required): Google Gemini API key for AI chat functionality

## Installation

### Prerequisites

- Go 1.24.6 or higher
- MongoDB (local or remote instance)
- Google Gemini API key (for AI responses)
- Internet connection (for free DuckDuckGo search functionality)

### Setup Steps

1. **Clone and navigate to the project:**
   ```bash
   cd /path/to/chatApp
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Set up environment variables:**
   ```bash
   cp .env.example .env  # If you have an example file
   # Edit .env with your configuration
   ```

4. **Ensure MongoDB is running:**
   ```bash
   # For local MongoDB
   mongod
   
   # Or use MongoDB Atlas (cloud)
   # Update MONGODB_URI in .env
   ```

## Running the Application

### Development Mode

```bash
# Start the server
go run main.go

# Server will start on http://localhost:8080 (or your configured PORT)
```

### Production Build

```bash
# Build the application
go build -o chatapp main.go

# Run the binary
./chatapp
```

### Testing the API

You can test the API using curl or tools like Postman:

1. **Register a new user:**
   ```bash
   curl -X POST http://localhost:8080/auth/register \
     -H "Content-Type: application/json" \
     -d '{"email":"test@example.com","password":"password123"}'
   ```

2. **Login to get JWT token:**
   ```bash
   curl -X POST http://localhost:8080/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"test@example.com","password":"password123"}'
   ```

3. **Create a chat (with JWT token):**
   ```bash
   curl -X POST http://localhost:8080/chat/create \
     -H "Authorization: Bearer YOUR_JWT_TOKEN_HERE"
   ```

4. **Send a message (SSE streaming):**
   ```bash
   curl -X POST http://localhost:8080/chat/message \
     -H "Authorization: Bearer YOUR_JWT_TOKEN_HERE" \
     -H "Content-Type: application/json" \
     -d '{"chat_id":"CHAT_ID_HERE","prompt":"Hello, how are you?"}'
   ```

## Database Schema

### Users Collection
```go
type User struct {
    ID        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
    Email     string             `json:"email" bson:"email"`
    Password  string             `json:"password" bson:"password"`
    CreatedAt time.Time          `json:"createdAt" bson:"created_at"`
    UpdatedAt time.Time          `json:"updatedAt" bson:"updated_at"`
}
```

### Chat Collection
```go
type Chat struct {
    ID        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
    UserID    primitive.ObjectID `json:"userId" bson:"user_id"`
    Title     string             `json:"title" bson:"title"`
    Messages  []Message          `json:"messages" bson:"messages,omitempty"`
    CreatedAt time.Time          `json:"createdAt" bson:"created_at"`
    UpdatedAt time.Time          `json:"updatedAt" bson:"updated_at"`
}

type Message struct {
    Role      string    `json:"role" bson:"role"`       // "user" | "model"
    Content   string    `json:"content" bson:"content"`
    CreatedAt time.Time `json:"createdAt" bson:"created_at"`
}
```

## Security Features

- **Password Hashing:** All passwords are hashed using bcrypt with default cost
- **JWT Tokens:** Secure token-based authentication with 24-hour expiration
- **Email Uniqueness:** Prevents duplicate email registration
- **Chat Ownership:** Users can only access their own chats
- **Token Management:** JWT tokens include user ID for easy validation

## AI Integration

The application integrates with Google's Gemini API to provide intelligent chat responses:

- **Model:** Uses `gemini-2.5-flash` for fast, efficient responses
- **History Management:** Maintains last 6 messages for context while optimizing token usage
- **Streaming:** Real-time response streaming via Server-Sent Events
- **Error Handling:** Graceful handling of API failures and timeouts

## Internet Search Feature

The application includes a completely free internet search capability:

- **Search Engine:** DuckDuckGo (no API key required)
- **Smart Routing:** AI-powered decision system determines when search is needed
- **Search Triggers:** Questions about current events, recent information, news, prices, or real-world data
- **No Search:** General knowledge, programming, math, logic, and explanations
- **Results Processing:** Extracts top 5 relevant results with titles and snippets
- **Context Integration:** Search results are injected into AI system instructions for informed responses
- **Cost Effective:** No additional costs beyond standard Gemini API usage

## Future Enhancements

Potential improvements for this application:

- [ ] WebSocket support for bidirectional communication
- [ ] Message encryption at rest
- [ ] Chat search and filtering
- [x] Free internet search via DuckDuckGo
- [ ] Multiple AI model support
- [ ] File upload capabilities
- [ ] User presence and typing indicators
- [ ] Chat sharing and collaboration features

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

# Created By Sarwan Azhar

## Support

For questions or issues, please open an issue in the project repository or contact me.