package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RedditClient struct {
	httpClient *http.Client
	userAgent  string
}

type Post struct {
	ID        string
	Title     string
	Content   string
	Author    string
	Subreddit string
	URL       string
	CreatedAt time.Time
}

type redditListingResponse struct {
	Data struct {
		Children []struct {
			Data redditPost `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type redditPost struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Selftext   string  `json:"selftext"`
	Body       string  `json:"body"`
	Author     string  `json:"author"`
	Subreddit  string  `json:"subreddit"`
	URL        string  `json:"url"`
	Permalink  string  `json:"permalink"`
	CreatedUTC float64 `json:"created_utc"`
}

// NewClient using public Reddit API
func NewClient() *RedditClient {
	return &RedditClient{
		httpClient: &http.Client{Timeout: 20 * time.Second},
		userAgent:  "linux:reddit-idea-crawler:v1.0.0 (by /u/tieu_le_dev)",
	}
}

// Fetch subreddit posts using public JSON
func (r *RedditClient) FetchPosts(ctx context.Context, subreddit string, limit int) ([]*Post, error) {
	url := fmt.Sprintf(
		"https://www.reddit.com/r/%s/new.json?limit=%d",
		subreddit, limit,
	)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", r.userAgent)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("reddit error %d: %s", resp.StatusCode, body)
	}

	var listing redditListingResponse
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, err
	}

	var posts []*Post
	for _, c := range listing.Data.Children {
		p := c.Data
		posts = append(posts, &Post{
			ID:        p.ID,
			Title:     p.Title,
			Content:   p.Selftext,
			Author:    p.Author,
			Subreddit: p.Subreddit,
			URL:       "https://reddit.com" + p.Permalink,
			CreatedAt: time.Unix(int64(p.CreatedUTC), 0),
		})
	}

	return posts, nil
}

// Fetch post comments using public JSON
func (r *RedditClient) FetchComments(ctx context.Context, subreddit, postID string) ([]*Post, error) {
	url := fmt.Sprintf(
		"https://www.reddit.com/r/%s/comments/%s.json",
		subreddit, postID,
	)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", r.userAgent)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("reddit error %d: %s", resp.StatusCode, body)
	}

	var data []redditListingResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if len(data) < 2 {
		return []*Post{}, nil
	}

	commentListing := data[1]

	var comments []*Post
	for _, c := range commentListing.Data.Children {
		p := c.Data
		if p.Body == "" || p.Body == "[deleted]" || p.Body == "[removed]" {
			continue
		}

		comments = append(comments, &Post{
			ID:        p.ID,
			Title:     "", // optional
			Content:   p.Body,
			Author:    p.Author,
			Subreddit: subreddit,
			URL:       "https://reddit.com" + p.Permalink,
			CreatedAt: time.Unix(int64(p.CreatedUTC), 0),
		})
	}

	return comments, nil
}
