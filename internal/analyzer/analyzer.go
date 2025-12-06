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
	Summarize  string `json:"summarize"`
	Content    string `json:"content"`
	Score      int    `json:"score"`
	IsMeta     bool   `json:"is_meta"`
	Categories string `json:"categories"`
}

const PROMPT = `
Analyze the following Reddit post or comment to determine if it contains a startup or business idea or a paint point that can be a idea.

Return the result in EXACTLY these 5 fields: summarize, content, score, is_meta, categories.

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
- categories = ""

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
Return 2–4 idea categories such as:
SaaS, AI, Productivity, Developer Tools, Fintech, Web3, Marketplaces, Health, EduTech, E-commerce, etc.

### is_meta = false

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
Do NOT merely summarize the Reddit post — transform it into a standalone startup idea description.
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
					"description": "Detailed explanation of the idea, problem, users, monetization, etc., in well markdown format",
				},
				"score": map[string]any{
					"type":        "integer",
					"description": "Score of the startup idea, from 0 to 100.",
				},
				"is_meta": map[string]any{
					"type":        "boolean",
					"description": "True if this is a meta-post asking others to share ideas, requiring comment analysis.",
				},
				"categories": map[string]any{
					"type":        "string",
					"description": "Categories that idea belong to",
				},
			},
		},
	}

	return &Analyzer{client: genaiClient, config: config}, nil
}

func (a *Analyzer) ExtractIdea(ctx context.Context, text string) (*AnalyzeResponse, error) {
	prompt := PROMPT + "\n\n Idea: \n" + text
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
