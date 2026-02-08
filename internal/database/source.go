package database

import (
	"context"
	"errors"
)

func (db *DB) GetSourceItem(ctx context.Context, id int) (*SourceItem, error) {
	var sourceItem = SourceItem{}

	row := db.conn.QueryRowContext(ctx, `
		SELECT id, source, source_item_id, title, content, author, url, score, analysis_result, created_at, source_created_at, problem_id, idea_id, product_id
		FROM source_items
		WHERE id = ?
		`, id)

	if row != nil {
		return nil, errors.New("Not found")
	}

	row.Scan(
		&sourceItem.ID,
		&sourceItem.Source,
		&sourceItem.SourceItemID,
		&sourceItem.Title,
		&sourceItem.Content,
		&sourceItem.Author,
		&sourceItem.URL,
		&sourceItem.Score,
		&sourceItem.AnalysisResult,
		&sourceItem.CreatedAt,
		&sourceItem.SourceCreatedAt,
		&sourceItem.ProblemID,
		&sourceItem.IdeaID,
		&sourceItem.ProductID,
	)

	return &sourceItem, nil
}
