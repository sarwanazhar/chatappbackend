package libs

import (
	"github.com/sarwanazhar/chatappbackend/model"
	"google.golang.org/genai"
)

// BuildGenaiContents builds the slice of contents (user/model messages) and
// the GenerateContentConfig containing the system instruction.
// - history: recent messages (will filter empty & last 6)
// - systemInstruction: if empty, a default assistant instruction is used
func BuildGenaiContents(history []model.Message, systemInstruction string) (
	contents []*genai.Content, cfg *genai.GenerateContentConfig,
) {
	contents = []*genai.Content{}

	// filter out empty messages and take last 6
	filtered := []model.Message{}
	for _, m := range history {
		if m.Content != "" {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) > 6 {
		filtered = filtered[len(filtered)-6:]
	}

	// convert each message into genai Content via genai.Text(...)
	for _, m := range filtered {
		// genai.Text returns []*genai.Content
		parts := genai.Text(m.Content)
		if len(parts) > 0 {
			contents = append(contents, parts...)
		}
	}

	// prepare config with system instruction
	if systemInstruction == "" {
		systemInstruction = "You are a helpful AI assistant."
	}

	cfg = &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: systemInstruction},
			},
		},
		// you can tune other params here if desired
	}
	return contents, cfg
}
