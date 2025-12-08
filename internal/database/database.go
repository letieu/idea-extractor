package database

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"strings"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
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
	// For SQLite, just use the database path
	connStr := cfg.Database.DBName // e.g., "ideas.db"

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

func (db *DB) CreateIdea(idea *Idea) error {
	query := `INSERT INTO ideas (title, content, score, categories) VALUES (?, ?, ?, ?) RETURNING id, created_at, updated_at`
	return db.conn.QueryRow(query, idea.Title, idea.Content, idea.Score, idea.Categories).Scan(&idea.ID, &idea.CreatedAt, &idea.UpdatedAt)
}

func (db *DB) CreateIdeaItem(item *IdeaItem, embedding []float32) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
        INSERT INTO idea_items (idea_id, source, source_item_id, title, content, author, url, score, categories, source_created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := tx.Exec(query,
		item.IdeaID,
		item.Source,
		item.SourceItemID,
		item.Title,
		item.Content,
		item.Author,
		item.URL,
		item.Score,
		item.Categories,
		item.SourceCreatedAt.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	vecQuery := `INSERT INTO vec_idea_items(rowid, embedding) VALUES (?, ?)`
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

func (db *DB) IdeaItemExists(source string, sourceItemID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM idea_items WHERE source = ? AND source_item_id = ?`
	err := db.conn.QueryRow(query, source, sourceItemID).Scan(&count)
	return count > 0, err
}

func (db *DB) GetUngroupedIdeaItems() ([]*IdeaItem, error) {
	rows, err := db.conn.Query(`
		SELECT id, idea_id, source, source_item_id, title, content, author, url, score, categories, created_at, source_created_at
		FROM idea_items
		WHERE idea_id IS NULL OR idea_id = 0
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*IdeaItem
	for rows.Next() {
		var item IdeaItem
		if err := rows.Scan(
			&item.ID,
			&item.IdeaID,
			&item.Source,
			&item.SourceItemID,
			&item.Title,
			&item.Content,
			&item.Author,
			&item.URL,
			&item.Score,
			&item.Categories,
			&item.CreatedAt,
			&item.SourceCreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, nil
}

func (db *DB) GetEmbeddings(ids []int) (map[int][]float32, error) {
	if len(ids) == 0 {
		return make(map[int][]float32), nil
	}

	query := `SELECT rowid, embedding FROM vec_idea_items WHERE rowid IN (?` + strings.Repeat(",?", len(ids)-1) + `)`
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

func (db *DB) FindSimilarItems(embedding []float32, limit int) ([]Neighbor, error) {
	query := `
		SELECT
			rowid,
			distance
		FROM vec_idea_items
		WHERE embedding MATCH ?
		ORDER BY distance
		LIMIT ?
	`

	v, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return nil, err
	}

	rows, err := db.conn.Query(query, v, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var neighbors []Neighbor
	for rows.Next() {
		var neighbor Neighbor
		if err := rows.Scan(&neighbor.ID, &neighbor.Distance); err != nil {
			return nil, err
		}
		neighbors = append(neighbors, neighbor)
	}
	return neighbors, nil
}

func (db *DB) UpdateIdeaIDForItems(ids []int, ideaID int) error {
	if len(ids) == 0 {
		return nil
	}

	query := `UPDATE idea_items SET idea_id = ? WHERE id IN (?` + strings.Repeat(",?", len(ids)-1) + `)`
	args := make([]any, len(ids)+1)
	args[0] = ideaID
	for i, id := range ids {
		args[i+1] = id
	}
	_, err := db.conn.Exec(query, args...)
	return err
}

func (db *DB) Close() error {
	return db.conn.Close()
}
