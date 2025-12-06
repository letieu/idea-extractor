package database

import "time"

type Idea struct {
	ID              int       `json:"id"`
	RedditID        string    `json:"reddit_id"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	Author          string    `json:"author"`
	Subreddit       string    `json:"subreddit"`
	URL             string    `json:"url"`
	Score           int       `json:"score"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	RedditCreatedAt time.Time `json:"reddit_created_at"`
	Categories      string    `json:"categories"`
}
