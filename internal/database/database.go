package database

import (
	"database/sql"

	"github.com/letieu/idea-extractor/config"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

func NewDB(cfg *config.Config) (*DB, error) {
	// For SQLite, just use the database path
	connStr := cfg.Database.DBName // e.g., "ideas.db"

	conn, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) SaveIdea(idea *Idea) error {
	query := `
        INSERT INTO ideas (reddit_id, title, content, author, subreddit, url, score, 
                          reddit_created_at, categories)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT (reddit_id) 
        DO UPDATE SET 
            title = excluded.title,
            content = excluded.content,
            updated_at = CURRENT_TIMESTAMP
        RETURNING id`

	return db.conn.QueryRow(query,
		idea.RedditID, idea.Title, idea.Content, idea.Author,
		idea.Subreddit, idea.URL, idea.Score,
		idea.RedditCreatedAt.Format("2006-01-02 15:04:05"),
		idea.Categories,
	).Scan(&idea.ID)
}

func (db *DB) IdeaExists(redditID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM ideas WHERE reddit_id = ?`
	err := db.conn.QueryRow(query, redditID).Scan(&count)
	return count > 0, err
}

func (db *DB) Close() error {
	return db.conn.Close()
}
