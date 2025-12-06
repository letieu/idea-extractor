package analyzer

import (
	"context"
	"encoding/json"
	"log"

	"github.com/letieu/idea-extractor/config"
	"google.golang.org/genai"
)

type Analyzer struct {
	client *genai.Client
	config *genai.GenerateContentConfig
}

type AnalyzeResponse struct {
	Summarize string `json:"summarize"`
	Content   string `json:"content"`
	Score     int    `json:"score"`
	IsMeta    bool   `json:"is_meta"`
}

const PROMPT = `
Analyze the following Reddit post or comment to determine if it contains a startup or business idea.

Return the result in exactly 4 fields: summarize, content, score, is_meta.

### 1. Meta-post detection (IMPORTANT)
If the text is a meta-post such as:
- “Share what you’re building…”
- “What are you working on?”
- “Show your project”
- “Share your startup idea”
- “Post your ideas here”
- Any open-ended or weekly thread asking others to submit ideas

Then return:
- summarize = "This is a meta-post inviting others to share ideas."
- content = "This post does not contain an idea. The crawler should check comments instead."
- score = 0
- is_meta = true

Do NOT score or analyze these as startup ideas.

### 2. Normal idea detection
If the post contains a startup or business idea:
- summarize: 1–2 sentence summary of the idea.
- content: Clear explanation including problem, target users, solution, monetization, and reasoning.
- score: A number from 0–100 (see scoring rules below).
- is_meta = false

### 3. Scoring rules (strict)
- 0–25 = Trash / vague / very weak idea / tiny audience  
- 26–50 = Basic or average idea; limited users or unclear value  
- 51–75 = Good idea with clear demand and realistic advantages  
- 76–90 = Strong, scalable idea with large potential  
- 91–100 = Exceptional and rare — only for truly outstanding concepts

Be strict and realistic. Do NOT give high scores to weak, simple, or low-value ideas.
`

func New(ctx context.Context, cnf config.Config) (*Analyzer, error) {
	genaiClient, err := genai.NewClient(ctx,
		&genai.ClientConfig{
			APIKey: cnf.Gemini.APIKey,
		},
	)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseJsonSchema: map[string]any{
			"type":     "object",
			"required": []string{"summarize", "content", "score", "is_meta"},
			"properties": map[string]any{
				"summarize": map[string]any{
					"type":        "string",
					"description": "A short summary of the idea or explanation.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Detailed explanation of the idea, problem, users, monetization, etc.",
				},
				"score": map[string]any{
					"type":        "integer",
					"description": "Score of the startup idea, from 0 to 100.",
				},
				"is_meta": map[string]any{
					"type":        "boolean",
					"description": "True if this is a meta-post asking others to share ideas, requiring comment analysis.",
				},
			},
		},
	}

	return &Analyzer{client: genaiClient, config: config}, nil
}

func (a *Analyzer) ExtractIdea(ctx context.Context, text string) (*AnalyzeResponse, error) {
	prompt := PROMPT + "\n\n" + text
	result, _ := a.client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash",
		genai.Text(prompt),
		a.config,
	)

	raw := result.Text()
	var idea AnalyzeResponse
	if err := json.Unmarshal([]byte(raw), &idea); err != nil {
		return nil, err
	}

	return &idea, nil
}
