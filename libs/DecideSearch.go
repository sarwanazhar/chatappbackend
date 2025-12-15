package libs

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"
)

func DecideSearch(prompt string) string {
	router := `
You are a routing agent.

Decide whether answering the user's question requires searching the internet.

Choose SEARCH if:
- Depends on current, recent, or changing information
- Involves real-world events, people, companies, prices, or news
- Asks for "latest", "current", "today", or similar

Choose NO_SEARCH if:
- Can be answered using general knowledge
- Is about programming, math, logic, or explanations
- Does not require up-to-date information

Respond with ONLY one word: SEARCH or NO_SEARCH.
Do not add any other text.
`

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "NO_SEARCH"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		fmt.Println("error here 123")
		return "NO_SEARCH"
	}

	// Use system instruction via config and pass prompt as content
	resp, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash-lite",
		genai.Text(prompt), // returns []*genai.Content, but variadic is accepted
		&genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{{Text: router}},
			},
		},
	)
	if err != nil {
		return "NO_SEARCH"
	}

	// Extract text from response
	var out string
	if len(resp.Candidates) > 0 {
		for _, p := range resp.Candidates[0].Content.Parts {
			out += p.Text
		}
	}

	out = strings.TrimSpace(strings.ToUpper(out))
	if out == "SEARCH" {
		return "SEARCH"
	}
	return "NO_SEARCH"
}
