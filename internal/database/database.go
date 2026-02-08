package database

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"strings"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	"github.com/google/uuid"
	"github.com/letieu/idea-extractor/config"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

type Neighbor struct {
	ID       int
	Distance float32
}

// toFloat32Slice converts a slice of bytes to a slice of float32.
func toFloat32Slice(data []byte) ([]float32, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("invalid byte slice length for float32 conversion")
	}
	floats := make([]float32, len(data)/4)
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, &floats)
	return floats, err
}

func NewDB(cfg *config.Config) (*DB, error) {
	sqlite_vec.Auto()
	connStr := cfg.Database.DBName

	conn, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	db := &DB{conn: conn}

	return db, nil
}

func generateUUID() string {
	return uuid.New().String()
}

func (db *DB) CreateProblem(problem *Problem) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on error

	problem.ID = generateUUID()
	query := `INSERT INTO problems (id, title, description, pain_points, score, created_at, updated_at, slug)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	painPointsStr := strings.Join(problem.PainPoints, ",")

	_, err = tx.Exec(query,
		problem.ID,
		problem.Title,
		problem.Description,
		painPointsStr,
		problem.Score,
		problem.CreatedAt,
		problem.UpdatedAt,
		problem.Slug,
	)
	if err != nil {
		return fmt.Errorf("failed to insert problem: %w", err)
	}

	// Retrieve the auto-generated IntID
	var intID int64
	err = tx.QueryRow("SELECT int_id FROM problems WHERE id = ?", problem.ID).Scan(&intID)
	if err != nil {
		return fmt.Errorf("failed to retrieve problem IntID: %w", err)
	}
	problem.IntID = int(intID)

	// Store embedding in vec_problems table
	if len(problem.Embedding) > 0 {
		vecQuery := `INSERT INTO vec_problems(rowid, embedding) VALUES (?, ?)`
		v, err := sqlite_vec.SerializeFloat32(problem.Embedding)
		if err != nil {
			return fmt.Errorf("failed to serialize problem embedding: %w", err)
		}

		_, err = tx.Exec(vecQuery, problem.IntID, v)
		if err != nil {
			return fmt.Errorf("failed to insert problem embedding: %w", err)
		}
	}

	// create categories
	for _, category := range problem.Categories {
		_, err := tx.Exec(`INSERT INTO problem_categories (problem_id, category_slug) VALUES (?, ?)`, problem.ID, category)
		if err != nil {
			return fmt.Errorf("failed to insert problem category: %w", err)
		}
	}

	return tx.Commit()
}

func (db *DB) CreateIdea(idea *Idea) error {
	idea.ID = generateUUID()
	query := `INSERT INTO ideas (id, title, description, features, score, created_at, updated_at, slug)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	featuresStr := strings.Join(idea.Features, ",")

	_, err := db.conn.Exec(query,
		idea.ID,
		idea.Title,
		idea.Description,
		featuresStr,
		idea.Score,
		idea.CreatedAt,
		idea.UpdatedAt,
		idea.Slug,
	)

	// create categories
	for _, category := range idea.Categories {
		_, err := db.conn.Exec(`INSERT INTO idea_categories (idea_id, category_slug) VALUES (?, ?)`, idea.ID, category)
		if err != nil {
			return fmt.Errorf("failed to insert idea category: %w", err)
		}
	}
	return err
}

func (db *DB) CreateProduct(product *Product) error {
	product.ID = generateUUID()
	query := `INSERT INTO products (id, name, description, url, created_at, updated_at, slug)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := db.conn.Exec(query,
		product.ID,
		product.Name,
		product.Description,
		product.URL,
		product.CreatedAt,
		product.UpdatedAt,
		product.Slug,
	)

	// create categories
	for _, category := range product.Categories {
		_, err := db.conn.Exec(`INSERT INTO product_categories (product_id, category_slug) VALUES (?, ?)`, product.ID, category)
		if err != nil {
			return fmt.Errorf("failed to insert product category: %w", err)
		}
	}

	return err
}

func (db *DB) CreateSourceItem(item *SourceItem, embedding []float32, analysisResult string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
        INSERT INTO source_items (source, source_item_id, title, content, author, url, score, analysis_result, source_created_at, problem_id, idea_id, product_id)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := tx.Exec(query,
		item.Source,
		item.SourceItemID,
		item.Title,
		item.Content,
		item.Author,
		item.URL,
		item.Score,
		analysisResult,
		item.SourceCreatedAt.Format("2006-01-02 15:04:05"),
		sql.NullString{String: item.ProblemID, Valid: item.ProblemID != ""},
		sql.NullString{String: item.IdeaID, Valid: item.IdeaID != ""},
		sql.NullString{String: item.ProductID, Valid: item.ProductID != ""},
	)
	if err != nil {
		return err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	item.ID = int(lastID)

	vecQuery := `INSERT INTO vec_source_items(rowid, embedding) VALUES (?, ?)`
	v, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return err
	}

	_, err = tx.Exec(vecQuery, lastID, v)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) SourceItemExists(source string, sourceItemID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM source_items WHERE source = ? AND source_item_id = ?`
	err := db.conn.QueryRow(query, source, sourceItemID).Scan(&count)
	return count > 0, err
}

func (db *DB) GetUngroupedSourceItems() ([]*SourceItem, error) {
	rows, err := db.conn.Query(`
		SELECT id, source, source_item_id, title, content, author, url, score, analysis_result, created_at, source_created_at, problem_id, idea_id, product_id
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

func (db *DB) GetEmbeddings(ids []int) (map[int][]float32, error) {
	if len(ids) == 0 {
		return make(map[int][]float32), nil
	}

	query := `SELECT rowid, embedding FROM vec_source_items WHERE rowid IN (?` + strings.Repeat(",?", len(ids)-1) + `)`
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	embeddings := make(map[int][]float32)
	for rows.Next() {
		var id int
		var embeddingBytes []byte
		if err := rows.Scan(&id, &embeddingBytes); err != nil {
			return nil, err
		}
		embedding, err := toFloat32Slice(embeddingBytes)
		if err != nil {
			return nil, err
		}
		embeddings[id] = embedding
	}
	return embeddings, nil
}

func (db *DB) FindSimilarProblems(embedding []float32, limit int, threshold float32) ([]*Problem, error) {
	query := `
		SELECT
			p.int_id,
			p.id,
			p.title,
			p.description,
			p.pain_points,
			p.score,
			p.created_at,
			p.updated_at,
			v.distance
		FROM vec_problems AS v
		JOIN problems AS p ON v.rowid = p.int_id
		WHERE v.embedding MATCH ?
		  AND k = ?
		  AND v.distance <= ?
		ORDER BY v.distance;
	`

	v, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize embedding: %w", err)
	}

	rows, err := db.conn.Query(query, v, limit, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar problems: %w", err)
	}
	defer rows.Close()

	var similarProblems []*Problem
	for rows.Next() {
		var problem Problem
		var painPointsStr string
		var distance float32 // Although distance is returned, it's not stored in Problem struct directly.

		if err := rows.Scan(
			&problem.IntID,
			&problem.ID,
			&problem.Title,
			&problem.Description,
			&painPointsStr,
			&problem.Score,
			&problem.CreatedAt,
			&problem.UpdatedAt,
			&distance,
		); err != nil {
			return nil, fmt.Errorf("failed to scan similar problem row: %w", err)
		}
		problem.PainPoints = strings.Split(painPointsStr, ",")
		if len(problem.PainPoints) == 1 && problem.PainPoints[0] == "" {
			problem.PainPoints = []string{}
		}

		// Note: The embedding itself is not retrieved here, only the distance.
		similarProblems = append(similarProblems, &problem)
	}
	return similarProblems, nil
}

func (db *DB) UpdateSourceItemProblemID(ids []int, problemID string) error {
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

func (db *DB) UpdateSourceItemIdeaID(ids []int, ideaID string) error {
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

func (db *DB) UpdateSourceItemProductID(ids []int, productID string) error {
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

func (db *DB) CreateProblemIdea(problemId, ideaId string) error {
	query := `INSERT INTO problem_idea (problem_id, idea_id)
	VALUES (?, ?)`

	_, err := db.conn.Exec(query,
		problemId,
		ideaId,
	)

	return err
}

func (db *DB) CreateProblemProduct(problemId, productId string) error {
	query := `INSERT INTO problem_product (problem_id, product_id)
	VALUES (?, ?)`

	_, err := db.conn.Exec(query,
		problemId,
		productId,
	)

	return err
}
