package crawl

import (
	"context"
	"encoding/json"
	"log"

	"github.com/letieu/idea-extractor/config"
	"github.com/letieu/idea-extractor/internal/analysis"
	"github.com/letieu/idea-extractor/internal/database"
	"github.com/letieu/idea-extractor/internal/embeddings"
	"github.com/letieu/idea-extractor/internal/reddit"
)

type Crawler struct {
	redditClient *reddit.RedditClient
	db           CrawlerStore
	analyzer     *analysis.Analyzer
	config       *config.Config
}

type CrawlerStore interface {
	SourceItemExists(source string, sourceItemID string) (bool, error)
	CreateSourceItem(item *database.SourceItem, embedding []float32, analysisResult string) error
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
	log.Printf("Crawling r/%s for problems, ideas, and products...", subreddit)
	posts, err := c.redditClient.FetchPosts(ctx, subreddit, c.config.Crawler.PostLimit)
	if err != nil {
		log.Printf("Error fetching posts from r/%s: %v", subreddit, err)
		return err
	}

	for _, post := range posts {
		existed, err := c.db.SourceItemExists("reddit", post.ID)
		if err != nil {
			log.Printf("Fail to check source item existence %v", err)
			continue
		}

		if existed {
			log.Printf("Source item already existed, ignoring: %s", post.Title)
			continue
		}

		log.Printf("Found new post: %s", post.Title)

		text := post.Title + "\n" + post.Content

		analysisResult, err := c.analyzer.ExtractAnalysis(ctx, text)
		if err != nil {
			log.Printf("Failed to extract analysis from post: %v", err)
			continue
		}

		if analysisResult.IsMeta {
			log.Printf("Post is meta, ignoring: %s", post.Title)
			continue
		}

		isEmpty := analysisResult.Idea.Score == 0 && analysisResult.Problem.Score == 0 && len(analysisResult.Products) == 0
		if isEmpty {
			log.Printf("Empty post, ignore: %s", post.Title)
			continue
		}

		embedding, err := embeddings.GenerateEmbedding(ctx, text)
		if err != nil {
			log.Printf("Failed to get embedding for item: %v", err)
			continue
		}

		analysisResultBytes, err := json.Marshal(analysisResult)
		if err != nil {
			log.Printf("Failed to marshal analysis result: %v", err)
			continue
		}
		analysisResultStr := string(analysisResultBytes)

		sourceItem := database.SourceItem{
			Source:          "reddit",
			SourceItemID:    post.ID,
			Title:           post.Title,
			Content:         post.Content,
			Author:          post.Author,
			URL:             post.URL,
			Score:           post.Score,
			SourceCreatedAt: post.CreatedAt,
			AnalysisResult:  analysisResultStr,
		}

		if err := c.db.CreateSourceItem(&sourceItem, embedding, analysisResultStr); err != nil {
			log.Printf("Failed to save source item: %v", err)
		}
	}

	return nil
}
