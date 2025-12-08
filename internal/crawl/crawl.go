package crawl

import (
	"context"
	"log"
	"strings"

	"github.com/letieu/idea-extractor/config"
	"github.com/letieu/idea-extractor/internal/analysis"
	"github.com/letieu/idea-extractor/internal/database"
	"github.com/letieu/idea-extractor/internal/reddit"
)

type Crawler struct {
	redditClient *reddit.RedditClient
	db           CrawlerStore
	analyzer     *analysis.Analyzer
	config       *config.Config
}

type CrawlerStore interface {
	IdeaItemExists(source string, sourceItemID string) (bool, error)
	CreateIdeaItem(item *database.IdeaItem, embedding []float32) error
	Close() error
}

func New(ctx context.Context) (*Crawler, error) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
		return nil, err
	}

	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return nil, err
	}

	anl, err := analysis.New(ctx, *cfg)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	redditClient := reddit.NewClient()

	return &Crawler{
		redditClient: redditClient,
		db:           db,
		analyzer:     anl,
		config:       cfg,
	}, nil
}

func (c *Crawler) Close() error {
	return c.db.Close()
}

func (c *Crawler) CrawlAll(ctx context.Context) {
	for _, subreddit := range c.config.Crawler.Subreddits {
		err := c.CrawlSubreddit(ctx, subreddit)
		if err != nil {
			log.Printf("Error when crawl subreddit: %s, error: %v", subreddit, err)
		}
	}
}

func (c *Crawler) CrawlSubreddit(ctx context.Context, subreddit string) error {
	log.Printf("Crawling r/%s...", subreddit)
	posts, err := c.redditClient.FetchPosts(ctx, subreddit, c.config.Crawler.PostLimit)
	if err != nil {
		log.Printf("Error fetching posts from r/%s: %v", subreddit, err)
		return err
	}

	for _, post := range posts {
		existed, err := c.db.IdeaItemExists("reddit", post.ID)
		if err != nil {
			log.Printf("Fail to check idea existed %v", err)
			continue
		}

		if existed {
			log.Printf("Existed, ignore \n")
			continue
		}

		log.Printf("Found new post: %s", post.Title)

		text := post.Title + "\n" + post.Content
		result, err := c.analyzer.ExtractIdea(ctx, text)
		if err != nil {
			log.Printf("Fail to analyze the idea %v", err)
			continue
		}

		if result.IsMeta {
			log.Printf("Is meta %s", post.Title)
			continue
		}

		embedding, err := analysis.GetEmbedding(ctx, result.Summarize)
		if err != nil {
			log.Printf("Failed to get embedding for item: %v", err)
			continue
		}

		ideaItem := database.IdeaItem{
			Source:          "reddit",
			SourceItemID:    post.ID,
			Title:           result.Summarize,
			Content:         result.Content,
			Author:          post.Author,
			URL:             post.URL,
			Score:           result.Score,
			Categories:      strings.Join(result.Categories, ","),
			ReferenceLinks:  strings.Join(result.ReferenceLinks, ","),
			SourceCreatedAt: post.CreatedAt,
		}

		if err := c.db.CreateIdeaItem(&ideaItem, embedding); err != nil {
			log.Printf("Fail to save idea item %v", err)
		}
	}

	return nil
}
