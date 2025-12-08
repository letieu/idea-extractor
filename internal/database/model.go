package database

import "time"

type Idea struct {
	ID             int       `json:"id"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	Score          int       `json:"score"`
	Categories     string    `json:"categories"`
	ReferenceLinks string    `json:"reference_links"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type IdeaItem struct {
	ID              int       `json:"id"`
	IdeaID          int       `json:"idea_id"`
	Source          string    `json:"source"`
	SourceItemID    string    `json:"source_item_id"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	Author          string    `json:"author"`
	URL             string    `json:"url"`
	Score           int       `json:"score"`
	Categories      string    `json:"categories"`
	ReferenceLinks  string    `json:"reference_links"`
	CreatedAt       time.Time `json:"created_at"`
	SourceCreatedAt time.Time `json:"source_created_at"`
}
