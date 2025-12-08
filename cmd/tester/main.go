package main

import (
	"context"
	"log"

	"github.com/letieu/idea-extractor/config"
	"github.com/letieu/idea-extractor/internal/analysis"
	"github.com/letieu/idea-extractor/internal/database"
)

func main() {
	// createEmbed()
	full()
}

func createEmbed() {
	ctx := context.Background()
	log.Println("Starting tester...")

	embedding, err := analysis.GetEmbedding(ctx, "Hello world")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%v", embedding)
}

func full() {
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

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting tester...")

	// 3. Test FindSimilarItems
	log.Println("Testing FindSimilarItems...")

	testText := "A tinder for pet"

	testEmbedding, err := analysis.GetEmbedding(ctx, testText)
	if err != nil {
		log.Fatalf("Failed to get embedding for test item: %v", err)
	}

	neighbors, err := db.FindSimilarItems(testEmbedding, 5)
	if err != nil {
		log.Fatalf("Failed to find similar items: %v", err)
	}

	log.Printf("Found %d neighbors for '%s':", len(neighbors), testText)
	for _, neighbor := range neighbors {
		log.Printf("  - Item ID: %d, Distance: %f", neighbor.ID, neighbor.Distance)
	}

	log.Println("Tester finished.")
}

func seed() {
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
	// 1. Mock data
	mockItems := []database.IdeaItem{
		{Source: "test", SourceItemID: "test1", Title: "A social network for pets", Content: "A place for pets to connect", Score: 80, Categories: "Social, Pets"},
		{Source: "test", SourceItemID: "test2", Title: "A tinder for dogs", Content: "Find a match for your furry friend", Score: 75, Categories: "Social, Pets"},
		{Source: "test", SourceItemID: "test3", Title: "A food delivery service for cats", Content: "Gourmet cat food delivered to your door", Score: 85, Categories: "E-commerce, Pets"},
		{Source: "test", SourceItemID: "test4", Title: "AI-powered personal finance assistant", Content: "An AI to manage your budget", Score: 90, Categories: "AI, Fintech"},
	}

	// 2. Insert mock data and generate embeddings
	log.Println("Inserting mock data...")
	for _, item := range mockItems {
		embedding, err := analysis.GetEmbedding(ctx, item.Title)
		if err != nil {
			log.Printf("Failed to get embedding for item '%s': %v", item.Title, err)
			continue
		}

		if err := db.CreateIdeaItem(&item, embedding); err != nil {
			log.Printf("Failed to save idea item '%s': %v", item.Title, err)
		}
	}

	log.Println("Mock data inserted.")
}
