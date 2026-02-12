package group

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/letieu/idea-extractor/config"
	"github.com/letieu/idea-extractor/internal/analysis"
	"github.com/letieu/idea-extractor/internal/database"
	"github.com/letieu/idea-extractor/internal/embeddings"
)

type Groupper struct {
	db     *database.DB
	config *config.Config
}

func New() (*Groupper, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	db, err := database.NewDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return &Groupper{db: db, config: cfg}, nil
}

func (g *Groupper) Close() error {
	return g.db.Close()
}

func (g *Groupper) ProcessSourceItems(ctx context.Context) error {
	sourceItems, err := g.db.GetUngroupedSourceItems()
	if err != nil {
		return fmt.Errorf("get ungrouped source items: %w", err)
	}
	if len(sourceItems) == 0 {
		log.Println("No new source items to process.")
		return nil
	}

	log.Printf("Found %d new source items to process.", len(sourceItems))

	for _, item := range sourceItems {
		var analysisResult analysis.AnalysisResult
		if err := json.Unmarshal([]byte(item.AnalysisResult), &analysisResult); err != nil {
			log.Printf("Warning: could not unmarshal analysis result for source item %d: %v", item.ID, err)
			continue
		}

		problemId, err := g.createProblem(ctx, item.ID, analysisResult.Problem)
		if err != nil {
			log.Printf("Failed to create problem: %v", err)
		}

		ideaId, err := g.createIdea(ctx, item.ID, analysisResult.Idea)
		if err != nil {
			log.Printf("Failed to create idea: %v", err)
		}

		if problemId != 0 && ideaId != 0 {
			g.db.CreateProblemIdea(problemId, ideaId)
		}

		for _, p := range analysisResult.Products {
			productId, err := g.createProduct(ctx, item.ID, p)
			if err != nil {
				log.Printf("Failed to create product: %v", err)
			}
			if problemId != 0 && productId != 0 {
				g.db.LinkProblemProduct(problemId, productId)
			}
			if ideaId != 0 && productId != 0 {
				g.db.LinkIdeaProduct(ideaId, productId)
			}
		}
	}

	log.Println("Grouper finished processing source items.")
	return nil
}

func (g *Groupper) createProblem(ctx context.Context, sourcId int, p analysis.AnalysisResultProblem) (int, error) {
	if p.Score == 0 {
		return 0, nil
	}

	const problemSimilarityThreshold float32 = 0.2 // Adjust this value based on desired similarity
	const maxSimilarProblems = 5                   // Number of similar problems to fetch

	// Generate embedding for the problem
	embedding, err := embeddings.GenerateEmbedding(ctx, p.Title)
	if err != nil {
		log.Printf("Failed to generate embedding for problem '%s': %v", p.Title, err)
		return 0, err
	}

	// Check for similar problems
	similarProblems, err := g.db.FindSimilarProblems(embedding, maxSimilarProblems, 0.2)
	if err != nil {
		log.Printf("Failed to find similar problems for '%s': %v", p.Title, err)
		// Continue to create new problem if similarity check fails
	}

	log.Printf("Found %d similar problems", len(similarProblems))

	var problemId int

	if len(similarProblems) > 0 {
		problemId = similarProblems[0].ID // Use the most similar problem
		log.Printf("Found similar problem (ID: %d) for '%s'. >>> %s .", problemId, p.Title, similarProblems[0].Title)
	} else {
		problem := &database.Problem{
			Title:       p.Title,
			Description: p.Description,
			PainPoints:  p.PainPoints,
			Score:       p.Score, // Use overall score for now
			Categories:  p.Categories,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Slug:        CreateSlug(p.Title),
			Embedding:   embedding,
		}

		// No similar problem found, create a new one
		problemId, err = g.db.CreateProblem(problem)
		if err != nil {
			log.Printf("Failed to create problem: %v", err)
			return 0, err
		}
		log.Printf("Created new problem ID: %d '%s'", problemId, problem.Title)
	}

	g.db.UpdateSourceItemProblemID([]int{sourcId}, problemId)
	return problemId, nil
}

func (g *Groupper) createIdea(ctx context.Context, sourceId int, analysisResult analysis.AnalysisResultIdea) (int, error) {
	idea := &database.Idea{
		Title:       analysisResult.Title,
		Description: analysisResult.Description,
		Features:    analysisResult.Features,
		Score:       analysisResult.Score,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Categories:  analysisResult.Categories,
		Slug:        CreateSlug(analysisResult.Title),
	}

	ideaId, err := g.db.CreateIdea(idea)
	if err != nil {
		return 0, err
	}
	g.db.UpdateSourceItemIdeaID([]int{sourceId}, ideaId)
	return ideaId, nil
}

func (g *Groupper) createProduct(ctx context.Context, sourceId int, analysisResult analysis.AnalysisResultProduct) (int, error) {
	product := &database.Product{
		Name:        analysisResult.Name,
		Description: analysisResult.Description,
		URL:         analysisResult.URL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Categories:  analysisResult.Categories,
		Slug:        CreateSlug(analysisResult.Name),
	}

	productId, err := g.db.CreateProduct(product)
	if err != nil {
		return 0, err
	}

	g.db.UpdateSourceItemProductID([]int{sourceId}, productId)

	return productId, nil
}
