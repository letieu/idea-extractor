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

		if problemId != "" && ideaId != "" {
			g.db.CreateProblemIdea(problemId, ideaId)
		}

		for _, p := range analysisResult.Products {
			productId, err := g.createProduct(ctx, item.ID, p)
			if err != nil {
				log.Printf("Failed to create product: %v", err)
			}
			if problemId != "" && productId != "" {
				g.db.CreateProblemProduct(problemId, productId)
			}
		}
	}

	log.Println("Grouper finished processing source items.")
	return nil
}

func (g *Groupper) createProblem(ctx context.Context, sourcId int, p analysis.AnalysisResultProblem) (string, error) {
	if p.Score == 0 {
		return "", nil
	}

	const problemSimilarityThreshold float32 = 0.2 // Adjust this value based on desired similarity
	const maxSimilarProblems = 5                   // Number of similar problems to fetch

	// Generate embedding for the problem
	embedding, err := embeddings.GenerateEmbedding(ctx, p.Title)
	if err != nil {
		log.Printf("Failed to generate embedding for problem '%s': %v", p.Title, err)
		return "", err
	}

	// Check for similar problems
	similarProblems, err := g.db.FindSimilarProblems(embedding, maxSimilarProblems, problemSimilarityThreshold)
	if err != nil {
		log.Printf("Failed to find similar problems for '%s': %v", p.Title, err)
		// Continue to create new problem if similarity check fails
	}

	log.Printf("Found %d similar problems", len(similarProblems))

	var targetProblem *database.Problem
	if len(similarProblems) > 0 {
		targetProblem = similarProblems[0] // Use the most similar problem
		log.Printf("Found similar problem (ID: %s) for '%s'. Linking to existing problem.", targetProblem.ID, p.Title)
		log.Printf("New: %s, Old: %s", p.Title, targetProblem.Title)
		log.Println("_____")
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
		}

		// No similar problem found, create a new one
		if err := g.db.CreateProblem(problem); err != nil {
			log.Printf("Failed to create problem: %v", err)
			return "", err
		}
		targetProblem = problem
		log.Printf("Created new problem (ID: %s): '%s'", targetProblem.ID, problem.Title)
	}

	g.db.UpdateSourceItemProblemID([]int{sourcId}, targetProblem.ID)
	return targetProblem.ID, nil
}

func (g *Groupper) createIdea(ctx context.Context, sourceId int, analysisResult analysis.AnalysisResultIdea) (string, error) {
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
	if err := g.db.CreateIdea(idea); err != nil {
		return "", err
	}
	g.db.UpdateSourceItemIdeaID([]int{sourceId}, idea.ID)
	return idea.ID, nil
}

func (g *Groupper) createProduct(ctx context.Context, sourceId int, analysisResult analysis.AnalysisResultProduct) (string, error) {
	product := &database.Product{
		Name:        analysisResult.Name,
		Description: analysisResult.Description,
		URL:         analysisResult.URL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Categories:  analysisResult.Categories,
		Slug:        CreateSlug(analysisResult.Name),
	}
	if err := g.db.CreateProduct(product); err != nil {
		return "", err
	}
	g.db.UpdateSourceItemProductID([]int{sourceId}, product.ID)
	return product.ID, nil
}
