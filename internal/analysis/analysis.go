package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/letieu/idea-extractor/config"
)

type Analyzer struct {
	apiKey string
	model  string
}

type AnalyzeResponse struct {
	Summarize      string   `json:"summarize"`
	Content        string   `json:"content"`
	Score          int      `json:"score"`
	IsMeta         bool     `json:"is_meta"`
	Categories     []string `json:"categories"`
	ReferenceLinks []string `json:"reference_links"`
}

const PROMPT = `
Analyze the following Reddit post or comment to determine if it contains a startup or business idea or a paint point that can be a idea.

Return the result in EXACTLY these 5 fields: summarize, content, score, is_meta, categories, reference_links

Your output MUST be suitable for displaying in a public startup idea database.  
Do NOT mention brand names, product names, company names, real founders, personal stories, or any identifiable details.  
Rewrite the idea in a clean, neutral, product-agnostic format.

---

## 1. Meta-post detection (IMPORTANT)
If the text is a meta-post such as:
- “Share what you’re building…”
- “What are you working on?”
- “Show your project”
- “Share your startup idea”
- “Post your ideas here”
- Weekly / open threads asking people to submit ideas

Then return:

- summarize = "This is a meta-post inviting others to share ideas."
- content = "This post does not contain an idea. The crawler should analyze comments instead."
- score = 0
- is_meta = true
- categories = [] 
- reference_links = []

Do NOT treat these as startup ideas.

---

## 2. Normal idea detection
If the post contains a startup or business idea, rewrite it cleanly:

### summarize
- A short 1–2 sentence summary of the idea.
- Must NOT include brand names or personal details.
- Must describe the idea itself, not the Reddit post context.
- Example: “A mobile-first personal CRM that helps users maintain relationships using location-based context.”

### content
A clear, in markdown format, should easy to understand, rewritten explanation including:
- the problem  
- target users  
- the solution  
- how it might work  
- possible monetization  
- why it could be useful  

All rewritten in a polished, idea-hub-friendly way.
Exclude:
- founder stories  
- personal hacks or life events  
- real brand/product names  
- anything irrelevant to the idea itself  

### categories
In array format
Return 2–4 idea categories such as:
SaaS, AI, Productivity, Developer Tools, Fintech, Web3, Marketplaces, Health, EduTech, E-commerce, etc.

### is_meta = false

### reference_links
In array format
If post contains any product link

---

## 3. Scoring rules (strict)
Score the idea from 0–100 based on quality:

- **0–25** = Very weak / vague / tiny audience  
- **26–50** = Basic or average idea; limited users or unclear value  
- **51–75** = Good idea with clear demand  
- **76–90** = Strong, scalable idea  
- **91–100** = Exceptional and rare  

Be realistic and strict.  
Do NOT give high scores to simple, niche, copycat, or unclear concepts.

---

## 4. Output expectations
All output should be clean, professional, rewritten, and ready to be displayed in “Idea Hub”.  
Make sure that "content" should well format in Markdown, with title, good level render (##, ### ....)
Do NOT merely summarize the Reddit post — transform it into a standalone startup idea description.
`

func New(ctx context.Context, cnf config.Config) (*Analyzer, error) {
	return &Analyzer{
		apiKey: cnf.Mistral.APIKey,
		model:  cnf.Mistral.Model,
	}, nil
}

type MistralChatRequest struct {
	Model          string                 `json:"model"`
	Messages       []MistralMessage       `json:"messages"`
	ResponseFormat *MistralResponseFormat `json:"response_format,omitempty"`
}

type MistralMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MistralResponseFormat struct {
	Type       string            `json:"type"`
	JSONSchema MistralJSONSchema `json:"json_schema"`
}

type MistralJSONSchema struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type MistralChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int            `json:"index"`
		Message MistralMessage `json:"message"`
	} `json:"choices"`
}

func (a *Analyzer) ExtractIdea(ctx context.Context, text string) (*AnalyzeResponse, error) {
	prompt := PROMPT + "\n\nIdea:\n" + text

	reqBody := MistralChatRequest{
		Model: a.model,
		Messages: []MistralMessage{
			{Role: "user", Content: prompt},
		},
		ResponseFormat: &MistralResponseFormat{
			Type: "json_object",
			JSONSchema: MistralJSONSchema{
				Name: "idea_analysis",
				Schema: map[string]any{
					"type":     "object",
					"required": []string{"summarize", "content", "score", "is_meta", "categories", "reference_links"},
					"properties": map[string]any{
						"summarize": map[string]any{
							"type": "string",
						},
						"content": map[string]any{
							"type": "string",
						},
						"score": map[string]any{
							"type": "integer",
						},
						"is_meta": map[string]any{
							"type": "boolean",
						},
						"categories": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"reference_links": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
					},
				},
				Strict: true,
			},
		},
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx,
		"POST",
		"https://api.mistral.ai/v1/chat/completions",
		bytes.NewBuffer(raw),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Mistral API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("Mistral API error (status %d): %s", resp.StatusCode, errBody.String())
	}

	var mistralResp MistralChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&mistralResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(mistralResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	var idea AnalyzeResponse
	if err := json.Unmarshal([]byte(mistralResp.Choices[0].Message.Content), &idea); err != nil {
		return nil, fmt.Errorf("failed to unmarshal idea: %w", err)
	}

	return &idea, nil
}

type OllamaEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type OllamaEmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	reqBody := OllamaEmbeddingRequest{
		Model: "embeddinggemma",
		Input: text,
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx,
		"POST",
		"http://localhost:11434/api/embed",
		bytes.NewBuffer(raw),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	if len(parsed.Embeddings) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return parsed.Embeddings[0], nil
}
