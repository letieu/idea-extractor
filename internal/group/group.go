package group

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/letieu/idea-extractor/config"
	"github.com/letieu/idea-extractor/internal/database"
)

type Groupper struct {
	db                  *database.DB
	config              *config.Config
	similarityThreshold float32
	neighborLimit       int
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

	return &Groupper{
		db:                  db,
		config:              cfg,
		similarityThreshold: 0.6,
		neighborLimit:       100,
	}, nil
}

func (g *Groupper) Close() error {
	return g.db.Close()
}

func (g *Groupper) GroupIdea() error {
	items, err := g.db.GetUngroupedIdeaItems()
	if err != nil {
		return fmt.Errorf("get ungrouped items: %w", err)
	}
	if len(items) == 0 {
		log.Println("No new items to group.")
		return nil
	}

	log.Printf("Found %d new items to group.", len(items))

	itemIDs := extractIDs(items)

	embeddings, err := g.db.GetEmbeddings(itemIDs)
	if err != nil {
		return fmt.Errorf("get embeddings: %w", err)
	}

	clustered := make(map[int]bool)

	for _, item := range items {
		if clustered[item.ID] {
			continue
		}

		clusterItems, clusterIDs := g.buildCluster(item, items, embeddings, clustered)
		if len(clusterItems) == 0 {
			continue
		}

		idea, err := g.createClusterIdea(clusterItems)
		if err != nil {
			log.Printf("create idea failed: %v", err)
			continue
		}

		if err := g.db.UpdateIdeaIDForItems(clusterIDs, idea.ID); err != nil {
			log.Printf("update idea_id failed: %v", err)
		}

		log.Printf("Created idea %d for cluster of %d items.", idea.ID, len(clusterIDs))
	}

	log.Println("Grouper finished.")
	return nil
}

func (g *Groupper) buildCluster(
	item *database.IdeaItem,
	items []*database.IdeaItem,
	embeddings map[int][]float32,
	clustered map[int]bool,
) ([]*database.IdeaItem, []int) {

	itemEmbedding, ok := embeddings[item.ID]
	if !ok {
		log.Printf("Missing embedding for item %d", item.ID)
		return nil, nil
	}

	neighbors, err := g.db.FindSimilarItems(itemEmbedding, g.neighborLimit)
	if err != nil {
		log.Printf("FindSimilarItems failed for item %d: %v", item.ID, err)
		return nil, nil
	}

	var clusterItems []*database.IdeaItem
	var clusterIDs []int

	// Add the main item
	clusterItems = append(clusterItems, item)
	clusterIDs = append(clusterIDs, item.ID)
	clustered[item.ID] = true

	for _, n := range neighbors {
		if 1-n.Distance <= g.similarityThreshold {
			continue
		}
		if clustered[n.ID] {
			continue
		}

		if match := findItemByID(items, n.ID); match != nil {
			clusterItems = append(clusterItems, match)
			clusterIDs = append(clusterIDs, match.ID)
			clustered[match.ID] = true
		}
	}

	return clusterItems, clusterIDs
}

func (g *Groupper) createClusterIdea(clusterItems []*database.IdeaItem) (*database.Idea, error) {
	totalScore := 0
	categorySet := make(map[string]bool)
	referenceSet := make(map[string]bool)

	for _, it := range clusterItems {
		totalScore += it.Score

		// Parse categories
		for _, c := range parseUniqueList(it.Categories) {
			categorySet[c] = true
		}

		// Parse reference links
		for _, link := range parseUniqueList(it.ReferenceLinks) {
			referenceSet[link] = true
		}
	}

	avgScore := totalScore / len(clusterItems)

	// Convert sets to sorted lists
	categories := mapToSortedSlice(categorySet)
	referenceLinks := mapToSortedSlice(referenceSet)

	idea := &database.Idea{
		Title:          clusterItems[0].Title,
		Content:        clusterItems[0].Content,
		Score:          avgScore,
		Categories:     strings.Join(categories, ", "),
		ReferenceLinks: strings.Join(referenceLinks, ", "),
	}

	if err := g.db.CreateIdea(idea); err != nil {
		return nil, err
	}

	return idea, nil
}

func extractIDs(items []*database.IdeaItem) []int {
	ids := make([]int, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	return ids
}

func findItemByID(items []*database.IdeaItem, id int) *database.IdeaItem {
	for _, it := range items {
		if it.ID == id {
			return it
		}
	}
	return nil
}

func mapToSortedSlice(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for v := range m {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func parseUniqueList(input string) []string {
	set := make(map[string]bool)

	parts := strings.Split(input, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			set[p] = true
		}
	}

	result := make([]string, 0, len(set))
	for v := range set {
		result = append(result, v)
	}
	sort.Strings(result)
	return result
}
