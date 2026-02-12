package database

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/letieu/idea-extractor/config"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type DB struct {
	conn *sql.DB
}

func NewDB(cfg *config.Config) (*DB, error) {
	dbUrl := cfg.Database.Url
	token := cfg.Database.Token

	// conn, err := sql.Open("sqlite3", connStr)
	conn, err := sql.Open("libsql", fmt.Sprintf("%s?authToken=%s", dbUrl, token))

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	db := &DB{conn: conn}

	return db, nil
}

func (db *DB) CreateProblem(problem *Problem) (int, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `INSERT INTO problems (title, description, pain_points, score, created_at, updated_at, slug, embedding)
	VALUES (?, ?, ?, ?, ?, ?, ?, vector32(?))`

	painPointsStr := strings.Join(problem.PainPoints, ",")

	result, err := tx.Exec(query,
		problem.Title,
		problem.Description,
		painPointsStr,
		problem.Score,
		problem.CreatedAt,
		problem.UpdatedAt,
		problem.Slug,
		vector32String(problem.Embedding),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert problem: %w", err)
	}

	insertedID, err := result.LastInsertId()
	log.Printf("inserted %d \n", insertedID)
	if err != nil {
		return 0, err
	}

	for _, category := range problem.Categories {
		_, err := tx.Exec(`INSERT INTO problem_categories (problem_id, category_slug) VALUES (?, ?)`, insertedID, category)
		if err != nil {
			return 0, fmt.Errorf("failed to insert problem category: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	id := int(insertedID)
	return id, nil
}

func (db *DB) CreateIdea(idea *Idea) (int, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `INSERT INTO ideas (title, description, features, score, created_at, updated_at, slug)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	featuresStr := strings.Join(idea.Features, ",")

	result, err := tx.Exec(query,
		idea.Title,
		idea.Description,
		featuresStr,
		idea.Score,
		idea.CreatedAt,
		idea.UpdatedAt,
		idea.Slug,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert idea: %w", err)
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	// create categories
	for _, category := range idea.Categories {
		_, err := tx.Exec(`INSERT INTO idea_categories (idea_id, category_slug) VALUES (?, ?)`, insertedID, category)
		if err != nil {
			return 0, fmt.Errorf("failed to insert idea category: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return int(insertedID), nil
}

func (db *DB) CreateProduct(product *Product) (int, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Check if slug exists, if so add random number to make it unique
	uniqueSlug := product.Slug
	for {
		var count int
		err := tx.QueryRow(`SELECT COUNT(*) FROM products WHERE slug = ?`, uniqueSlug).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		if count == 0 {
			break
		}
		// Slug exists, add random number
		uniqueSlug = fmt.Sprintf("%s-%d", product.Slug, rand.Intn(100000))
	}

	query := `INSERT INTO products (name, description, url, created_at, updated_at, slug)
	VALUES (?, ?, ?, ?, ?, ?)`

	result, err := tx.Exec(query,
		product.Name,
		product.Description,
		product.URL,
		product.CreatedAt,
		product.UpdatedAt,
		uniqueSlug,
	)

	if err != nil {
		return 0, fmt.Errorf("failed to insert product: %w", err)
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	// create categories
	for _, category := range product.Categories {
		_, err := tx.Exec(`INSERT INTO product_categories (product_id, category_slug) VALUES (?, ?)`, insertedID, category)
		if err != nil {
			return 0, fmt.Errorf("failed to insert product category: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return int(insertedID), nil
}

func (db *DB) CreateSourceItem(item *SourceItem, analysisResult string) error {
	query := `
        INSERT INTO source_items (source, source_item_id, title, content, author, url, score, analysis_result, source_created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.conn.Exec(query,
		item.Source,
		item.SourceItemID,
		item.Title,
		item.Content,
		item.Author,
		item.URL,
		item.Score,
		analysisResult,
		item.SourceCreatedAt.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) SourceItemExists(source string, sourceItemID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM source_items WHERE source = ? AND source_item_id = ?`
	err := db.conn.QueryRow(query, source, sourceItemID).Scan(&count)
	return count > 0, err
}

func (db *DB) GetUngroupedSourceItems() ([]*SourceItem, error) {
	rows, err := db.conn.Query(`
		SELECT rowid, source, source_item_id, title, content, author, url, score, analysis_result, created_at, source_created_at, problem_id, idea_id, product_id
		FROM source_items
		WHERE problem_id IS NULL AND idea_id IS NULL AND product_id IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*SourceItem
	for rows.Next() {
		var item SourceItem
		var problemID, ideaID, productID sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.Source,
			&item.SourceItemID,
			&item.Title,
			&item.Content,
			&item.Author,
			&item.URL,
			&item.Score,
			&item.AnalysisResult,
			&item.CreatedAt,
			&item.SourceCreatedAt,
			&problemID,
			&ideaID,
			&productID,
		); err != nil {
			return nil, err
		}
		if problemID.Valid {
			item.ProblemID = problemID.String
		}
		if ideaID.Valid {
			item.IdeaID = ideaID.String
		}
		if productID.Valid {
			item.ProductID = productID.String
		}
		items = append(items, &item)
	}
	return items, nil
}

func (db *DB) FindSimilarProblems(
	embedding []float32,
	limit int,
	threshold float32,
) ([]*Problem, error) {
	query := `
		SELECT
			id,
			title,
			description,
			pain_points,
			score,
			created_at,
			updated_at,
            vector_extract(embedding),
	        vector_distance_cos(embedding, vector32(?)) AS distance
		FROM problems
	    WHERE distance < ?
		ORDER BY distance ASC
		LIMIT ?;
	`

	rows, err := db.conn.Query(query, vector32String(embedding), threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar problems: %w", err)
	}
	defer rows.Close()

	var similarProblems []*Problem

	for rows.Next() {
		var problem Problem
		var painPointsStr string
		var distance float32
		var embedding string

		if err := rows.Scan(
			&problem.ID,
			&problem.Title,
			&problem.Description,
			&painPointsStr,
			&problem.Score,
			&problem.CreatedAt,
			&problem.UpdatedAt,
			&embedding,
			&distance,
		); err != nil {
			return nil, fmt.Errorf("failed to scan similar problem row: %w", err)
		}
		log.Printf("distance %f", distance)

		if painPointsStr != "" {
			problem.PainPoints = strings.Split(painPointsStr, ",")
		} else {
			problem.PainPoints = []string{}
		}

		similarProblems = append(similarProblems, &problem)
	}

	return similarProblems, nil
}

func (db *DB) UpdateSourceItemProblemID(ids []int, problemID int) error {
	if len(ids) == 0 {
		return nil
	}
	query := `UPDATE source_items SET problem_id = ? WHERE id IN (?` + strings.Repeat(",?", len(ids)-1) + `)`
	args := make([]interface{}, len(ids)+1)
	args[0] = problemID
	for i, id := range ids {
		args[i+1] = id
	}
	_, err := db.conn.Exec(query, args...)
	return err
}

func (db *DB) UpdateSourceItemIdeaID(ids []int, ideaID int) error {
	if len(ids) == 0 {
		return nil
	}
	query := `UPDATE source_items SET idea_id = ? WHERE id IN (?` + strings.Repeat(",?", len(ids)-1) + `)`
	args := make([]interface{}, len(ids)+1)
	args[0] = ideaID
	for i, id := range ids {
		args[i+1] = id
	}
	_, err := db.conn.Exec(query, args...)
	return err
}

func (db *DB) UpdateSourceItemProductID(ids []int, productID int) error {
	if len(ids) == 0 {
		return nil
	}
	query := `UPDATE source_items SET product_id = ? WHERE id IN (?` + strings.Repeat(",?", len(ids)-1) + `)`
	args := make([]interface{}, len(ids)+1)
	args[0] = productID
	for i, id := range ids {
		args[i+1] = id
	}
	_, err := db.conn.Exec(query, args...)
	return err
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) CreateProblemIdea(problemId, ideaId int) error {
	query := `INSERT INTO problem_idea (problem_id, idea_id)
	VALUES (?, ?)`

	_, err := db.conn.Exec(query,
		problemId,
		ideaId,
	)

	return err
}

func (db *DB) LinkProblemProduct(problemId, productId int) error {
	query := `INSERT INTO problem_product (problem_id, product_id)
	VALUES (?, ?)`

	_, err := db.conn.Exec(query,
		problemId,
		productId,
	)

	return err
}

func (db *DB) LinkIdeaProduct(ideaId, productId int) error {
	query := `INSERT INTO idea_product (idea_id, product_id)
	VALUES (?, ?)`

	_, err := db.conn.Exec(query,
		ideaId,
		productId,
	)

	return err
}

func vector32String(arr []float32) string {
	parts := make([]string, len(arr))
	for i, v := range arr {
		parts[i] = fmt.Sprintf("%.3f", v) // format to 3 decimal places
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}
