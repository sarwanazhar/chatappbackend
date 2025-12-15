package libs

import (
	"github.com/sarwanazhar/chatappbackend/model"
	"google.golang.org/genai"
)

// Helper function to map your application messages to Gemini API format
func ConvertHistoryToGenaiContent(appHistory []model.Message) []*genai.Content {
	var contents []*genai.Content
	for _, msg := range appHistory {
		role := genai.RoleUser // Default
		if msg.Role == "model" {
			role = genai.RoleModel
		}

		// Create a Content object with the message text as a Part
		content := &genai.Content{
			Role: role,
			Parts: []*genai.Part{
				{
					Text: msg.Content,
				},
			},
		}
		contents = append(contents, content)
	}
	return contents
}
