package main

import (
	"context"
	"log"
	"time"

	"github.com/letieu/idea-extractor/config"
	"github.com/letieu/idea-extractor/internal/analyzer"
	"github.com/letieu/idea-extractor/internal/database"
	"github.com/letieu/idea-extractor/internal/reddit"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	redditClient := reddit.NewClient()
	analyzer, err := analyzer.New(ctx, *cfg)
	if err != nil {
		log.Fatal(err)
	}

	_ = redditClient

	log.Printf("Starting crawler for %d subreddits...", len(cfg.Crawler.Subreddits))
	for _, subreddit := range cfg.Crawler.Subreddits {
		log.Printf("Crawling r/%s...", subreddit)
		posts, err := redditClient.FetchPosts(ctx, subreddit, cfg.Crawler.PostLimit)
		if err != nil {
			log.Printf("Error fetching posts from r/%s: %v", subreddit, err)
			continue
		}

		for _, post := range posts {
			log.Printf("P %+v", post.Title)
			text := post.Title + "\n" + post.Content
			result, err := analyzer.ExtractIdea(ctx, text)
			if err != nil {
				log.Printf("Fail to analyze the idea %v", err)
			}

			log.Printf("%+v", result.IsMeta)
			if result.IsMeta {
				log.Printf("Is meta %s", post.Title)
				continue
			}

			log.Printf("Found %s", result.Summarize)
			if result.Score > 30 {
				idea := database.Idea{
					RedditID:        post.ID,
					Author:          post.Author,
					Subreddit:       post.Subreddit,
					URL:             post.URL,
					RedditCreatedAt: post.CreatedAt,

					Title:   result.Summarize,
					Content: result.Content,
					Score:   result.Score,
				}
				existed, err := db.IdeaExists(post.ID)
				if err != nil {
					log.Printf("Fail to check idea existed %v", err)
				}

				if existed == false {
					continue
				}

				err = db.SaveIdea(&idea)
				if err != nil {
					log.Printf("Fail to save idea %v", err)
				}
			} else {
				log.Printf("Ignore with score %d", result.Score)
			}
		}

		time.Sleep(20 * time.Second)
	}
}
