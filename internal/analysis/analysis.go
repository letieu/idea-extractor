package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/letieu/idea-extractor/config"
)

type Analyzer struct {
	apiKey string
	model  string
}

type AnalysisResultProblem struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	PainPoints  []string `json:"pain_points"`
	Score       int      `json:"score"`
	Categories  []string `json:"categories"`
}

type AnalysisResultIdea struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	Score       int      `json:"score"`
	Categories  []string `json:"categories"`
}

type AnalysisResultProduct struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Categories  []string `json:"categories"`
}

// AnalysisResult holds the structured output from the LLM after analyzing a post for problem, idea, and products.
type AnalysisResult struct {
	IsMeta   bool                    `json:"is_meta"`
	Problem  AnalysisResultProblem   `json:"problem"`
	Idea     AnalysisResultIdea      `json:"idea"`
	Products []AnalysisResultProduct `json:"products"`
}

const PROMPT = `
You will check a reddit post to find some data that can display on my 'IdeaDB' web site, my site will display some paint points, idea, start products, link between them, user can go to and see what is the potential problem, some good startup idea, or check another found work.

Analyze the following text to identify and extract three types of entities: Problem, Idea, and Products. Also, identify any links between them.

Return the result in a JSON object with these fields: "problem", "idea", "products", "is_meta".

- **Problem**: 1 User pain points or unmet needs. (Can be a start point to create a saas, bussiness from this problem, if it is some random problem that don't good to display in 'problem hub for startup founder', don't grab it).
- **Idea**: 1 Potential solutions to problem.
- **Products**: Existing implementations of idea (startups, projects).

Your output MUST be suitable for a public database. Do NOT mention brand names, company names, or personal details unless it's a product name.

---

## 1. Entity Extraction
Analyze the text and populate the "problem" and "idea" as objects, and "products" as an array.

### For Problem:
- **title**: A concise summary of the core problem.
- **description**: A clear explanation of the problem, who has it, and its consequences (In well markdown format, with heading).
- **pain_points**: 2-5 specific user pain points.
- **score**: Score of the problem in realword, can profit, 0-100
- **categories**: Categories of problem, in array format.

### For Idea:
- **title**: A concise summary of the solution.
- **description**: A clear explanation of the idea, how it works, and its potential (In well markdown format, with heading).
- **features**: 2-5 key features of the proposed solution.
- **score**: Score of the idea in realword, can profit, 0-100
- **categories**: Categories of idea, in array format.

### For each Product:
- **name**: The name of the product or startup.
- **description**: A brief description of what the product does. (In well markdown format, with heading).
- **url**: The URL of the product, if available.
- **categories**: Categories of product, in array format.

---

## 2. Meta-post detection
If the text is a meta-post (e.g., "Share your project"), set "is_meta" to true and leave the other arrays empty.

---

## Output Expectations
- The final output must be a single JSON object.
- If no entities of a certain type are found, the idea or problem should have score is 0, for the products, it should empty array.
- categories should be 2 -> 5 item, in this list: [technology, healthcare, finance, education, e-commerce, productivity, communication, entertainment, travel, food-beverage, fitness, real-estate, transportation, automotive, fashion, beauty, home-garden, pets, sports, gaming, music, art-design, photography, legal, hr-recruiting, marketing, sales, customer-service, analytics, security, sustainability, social-media, ai-ml, iot, blockchain, saas, mobile, web, hardware, infrastructure]
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

func (a *Analyzer) ExtractAnalysis(ctx context.Context, text string) (*AnalysisResult, error) {
	prompt := PROMPT + "\n\nPost:\n" + text

	reqBody := MistralChatRequest{
		Model: a.model,
		Messages: []MistralMessage{
			{Role: "user", Content: prompt},
		},
		ResponseFormat: &MistralResponseFormat{
			Type: "json_object",
			JSONSchema: MistralJSONSchema{
				Name: "entity_analysis",
				Schema: map[string]any{
					"type":     "object",
					"required": []string{"problem", "idea", "products", "is_meta"},
					"properties": map[string]any{
						"problem": map[string]any{
							"type":     "object",
							"required": []string{"title", "description", "pain_points", "score", "categories"},
							"properties": map[string]any{
								"title":       map[string]any{"type": "string"},
								"description": map[string]any{"type": "string"},
								"pain_points": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
								"score":       map[string]any{"type": "integer"},
								"categories": map[string]any{
									"type":  "array",
									"items": map[string]any{"type": "string"},
								},
							},
						},
						"idea": map[string]any{
							"type":     "object",
							"required": []string{"title", "description", "features", "score", "categories"},
							"properties": map[string]any{
								"title":       map[string]any{"type": "string"},
								"description": map[string]any{"type": "string"},
								"features":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
								"score":       map[string]any{"type": "integer"},
								"categories": map[string]any{
									"type":  "array",
									"items": map[string]any{"type": "string"},
								},
							},
						},
						"products": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":     "object",
								"required": []string{"name", "description", "url"},
								"properties": map[string]any{
									"name":        map[string]any{"type": "string"},
									"description": map[string]any{"type": "string"},
									"url":         map[string]any{"type": "string"},
									"categories": map[string]any{
										"type":  "array",
										"items": map[string]any{"type": "string"},
									},
								},
							},
						},
						"is_meta": map[string]any{"type": "boolean"},
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

	var analysis AnalysisResult
	if err := json.Unmarshal([]byte(mistralResp.Choices[0].Message.Content), &analysis); err != nil {
		log.Printf("%s", mistralResp.Choices[0].Message.Content)
		return nil, fmt.Errorf("failed to unmarshal analysis: %w", err)
	}

	return &analysis, nil
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
