package database

import "time"

// Problem represents an identified user problem or unmet need.
type Problem struct {
	ID          int       `json:"id" bson:"_id"`
	Slug        string    `json:"slug" bson:"slug"`
	Title       string    `json:"title" bson:"title"`
	Description string    `json:"description" bson:"description"`
	PainPoints  []string  `json:"pain_points" bson:"pain_points"`
	Categories  []string  `json:"categories" bson:"categories"`
	Score       int       `json:"score" bson:"score"` // Aggregated score
	Embedding   []float32 `json:"embedding" bson:"embedding"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

// Idea represents a potential solution to a problem.
type Idea struct {
	ID          int       `json:"id" bson:"_id"`
	Slug        string    `json:"slug" bson:"slug"`
	Title       string    `json:"title" bson:"title"`
	Description string    `json:"description" bson:"description"`
	Features    []string  `json:"features" bson:"features"`
	Categories  []string  `json:"categories" bson:"categories"`
	Score       int       `json:"score" bson:"score"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

// Product represents an existing implementation of an idea.
type Product struct {
	ID          int       `json:"id" bson:"_id"`
	Slug        string    `json:"slug" bson:"slug"`
	Name        string    `json:"name" bson:"name"`
	Description string    `json:"description" bson:"description"`
	URL         string    `json:"url" bson:"url"`
	Categories  []string  `json:"categories" bson:"categories"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

// SourceItem represents a raw item from a source.
type SourceItem struct {
	ID              int       `json:"id" bson:"_id"` // Auto-incrementing integer for sqlite-vec
	Source          string    `json:"source" bson:"source"`
	SourceItemID    string    `json:"source_item_id" bson:"source_item_id"`
	Title           string    `json:"title" bson:"title"`
	Content         string    `json:"content" bson:"content"`
	Author          string    `json:"author" bson:"author"`
	URL             string    `json:"url" bson:"url"`
	Score           int       `json:"score" bson:"score"`
	AnalysisResult  string    `json:"analysis_result" bson:"analysis_result"` // JSON of the analysis
	CreatedAt       time.Time `json:"created_at" bson:"created_at"`
	SourceCreatedAt time.Time `json:"source_created_at" bson:"source_created_at"`

	// Link to the grouped entities
	ProblemID string `json:"problem_id" bson:"problem_id"`
	IdeaID    string `json:"idea_id" bson:"idea_id"`
	ProductID string `json:"product_id" bson:"product_id"`
}

// ProblemIdeaLink links a Problem to an Idea that solves it.
type ProblemIdea struct {
	ProblemID string `json:"problem_id" bson:"problem_id"`
	IdeaID    string `json:"idea_id" bson:"idea_id"`
}

// IdeaProductLink links an Idea to a Product that implements it.
type ProblemProduct struct {
	ProblemID string `json:"problem_id" bson:"problem_id"`
	ProductID string `json:"product_id" bson:"product_id"`
}
