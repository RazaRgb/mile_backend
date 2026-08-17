package db

import (
	"backend/src/models"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UpsertArticle stores or replaces the article content for a node.
func UpsertArticle(article models.Article, tx ...pgx.Tx) error {
	query := `
	INSERT INTO articles (node_id, content)
	VALUES ($1, $2)
	ON CONFLICT (node_id) DO UPDATE SET content = EXCLUDED.content
	`
	if _, err := execOrTx(tx, query, article.NodeID, article.Content); err != nil {
		return fmt.Errorf("upsert article: %w", err)
	}
	return nil
}

// GetArticleByNodeID returns the article for a node.
func GetArticleByNodeID(nodeID uuid.UUID, tx ...pgx.Tx) (models.Article, error) {
	query := `
	SELECT node_id, content, created_at
	FROM articles
	WHERE node_id = $1
	`
	var a models.Article
	err := queryRowOrTx(tx, query, nodeID).Scan(&a.NodeID, &a.Content, &a.CreatedAt)
	if err != nil {
		return models.Article{}, fmt.Errorf("get article: %w", err)
	}
	return a, nil
}

// UpsertArticleMetadata stores or replaces telemetry/metadata for a node's article.
func UpsertArticleMetadata(meta models.ArticleMetadata, tx ...pgx.Tx) error {
	query := `
	INSERT INTO article_metadata (node_id, status, comment, score, seconds_watched)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (node_id) DO UPDATE SET
		status = EXCLUDED.status,
		comment = EXCLUDED.comment,
		score = EXCLUDED.score,
		seconds_watched = EXCLUDED.seconds_watched
	`
	if _, err := execOrTx(tx, query, meta.NodeID, meta.Status, meta.Comment, meta.Score, meta.SecondsWatched); err != nil {
		return fmt.Errorf("upsert article metadata: %w", err)
	}
	return nil
}

// GetAvailableFeedArticles returns up to limit UNWATCHED articles across the
// user's streams, in random order. Watched and skipped reels are excluded
// (skipped ones stay in the DB for later reference by the improve loop).
func GetAvailableFeedArticles(userID uuid.UUID, limit int, tx ...pgx.Tx) ([]models.FeedItem, error) {
	query := `
	SELECT a.node_id, n.topic, n.path, a.content, a.created_at
	FROM articles a
	JOIN nodes n ON n.id = a.node_id
	JOIN streams s ON s.id = n.stream_id
	JOIN article_metadata m ON m.node_id = a.node_id
	WHERE s.user_id = $1 AND m.status = $2
	ORDER BY RANDOM()
	LIMIT $3
	`
	rows, err := queryOrTx(tx, query, userID, models.StatusUnwatched, limit)
	if err != nil {
		return nil, fmt.Errorf("query feed articles: %w", err)
	}
	defer rows.Close()

	items := make([]models.FeedItem, 0)
	for rows.Next() {
		var it models.FeedItem
		if err := rows.Scan(&it.NodeID, &it.Topic, &it.Path, &it.Content, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan feed item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feed items: %w", err)
	}
	return items, nil
}

// UpdateArticleStatus sets the metadata status for a node's article, but only
// if the article belongs to the given user. Returns whether anything was
// updated (false = article missing or not owned by the user).
func UpdateArticleStatus(nodeID, userID uuid.UUID, status string, tx ...pgx.Tx) (bool, error) {
	query := `
	UPDATE article_metadata m
	SET status = $3
	FROM articles a
	JOIN nodes n ON n.id = a.node_id
	JOIN streams s ON s.id = n.stream_id
	WHERE a.node_id = $1 AND s.user_id = $2 AND m.node_id = a.node_id
	`
	tag, err := execOrTx(tx, query, nodeID, userID, status)
	if err != nil {
		return false, fmt.Errorf("update article status: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
