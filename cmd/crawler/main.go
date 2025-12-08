package main

import (
	"context"
	"log"

	"github.com/letieu/idea-extractor/internal/crawl"
)

func main() {
	ctx := context.Background()
	crawler, err := crawl.New(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	crawler.CrawlAll(ctx)
}
