package db

import (
	"backend/src/models"
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// execOrTx runs query against the pool, or against tx[0] when a transaction is provided.
func execOrTx(tx []pgx.Tx, query string, args ...any) (pgconn.CommandTag, error) {
	if len(tx) == 0 {
		return DB.Exec(context.Background(), query, args...)
	}
	return tx[0].Exec(context.Background(), query, args...)
}

// queryRowOrTx runs query against the pool, or against tx[0] when a
// transaction is provided.
func queryRowOrTx(tx []pgx.Tx, query string, args ...any) pgx.Row {
	if len(tx) == 0 {
		return DB.QueryRow(context.Background(), query, args...)
	}
	return tx[0].QueryRow(context.Background(), query, args...)
}

// queryOrTx runs a multi-row query against the pool, or against tx[0] when a
// transaction is provided.
func queryOrTx(tx []pgx.Tx, query string, args ...any) (pgx.Rows, error) {
	if len(tx) == 0 {
		return DB.Query(context.Background(), query, args...)
	}
	return tx[0].Query(context.Background(), query, args...)
}

// InsertNode persists a node.
func InsertNode(node models.Node, tx ...pgx.Tx) error {
	query := `
	INSERT INTO nodes (id, stream_id, topic, path, is_leaf, generated)
	VALUES ($1, $2, $3, $4, $5, $6)
	`
	if _, err := execOrTx(tx, query, node.ID, node.StreamID, node.Topic, node.Path, node.IsLeaf, node.Generated); err != nil {
		return fmt.Errorf("insert node: %w", err)
	}
	return nil
}

// UpdateNodeIsLeaf flips a node's leaf status (used when a node is expanded).
func UpdateNodeIsLeaf(nodeID uuid.UUID, isLeaf bool, tx ...pgx.Tx) error {
	query := `UPDATE nodes SET is_leaf = $2 WHERE id = $1`
	if _, err := execOrTx(tx, query, nodeID, isLeaf); err != nil {
		return fmt.Errorf("update node leaf status: %w", err)
	}
	return nil
}

// UpdateNodeGenerated marks whether a node's article has been generated.
func UpdateNodeGenerated(nodeID uuid.UUID, generated bool, tx ...pgx.Tx) error {
	query := `UPDATE nodes SET generated = $2 WHERE id = $1`
	if _, err := execOrTx(tx, query, nodeID, generated); err != nil {
		return fmt.Errorf("update node generated: %w", err)
	}
	return nil
}

// GetLeafNodesByUserAndStream returns all leaf nodes of a stream that belongs
// to the given user, ordered by path lexicographically (paths are zero-padded,
// so this yields depth-first in-order traversal).
func GetLeafNodesByUserAndStream(userID, streamID uuid.UUID, tx ...pgx.Tx) ([]models.Node, error) {
	query := `
	SELECT n.id, n.stream_id, n.topic, n.path, n.is_leaf, n.generated, n.created_at
	FROM nodes n
	JOIN streams s ON s.id = n.stream_id
	WHERE s.user_id = $1 AND n.stream_id = $2 AND n.is_leaf = true
	ORDER BY n.path
	`
	rows, err := queryOrTx(tx, query, userID, streamID)
	if err != nil {
		return nil, fmt.Errorf("query leaf nodes: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.Node, 0)
	for rows.Next() {
		var n models.Node
		if err := rows.Scan(&n.ID, &n.StreamID, &n.Topic, &n.Path, &n.IsLeaf, &n.Generated, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan leaf node: %w", err)
		}
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate leaf nodes: %w", err)
	}
	return nodes, nil
}

// GetAncestorNodes returns every ancestor of the node at path within a stream,
// ordered shallowest-first. Since paths are dot-joined ancestor ids, the
// ancestors are exactly the proper prefixes of the path, e.g. for
// "0001.0001.0005" the ancestors are "0001" and "0001.0001".
func GetAncestorNodes(streamID uuid.UUID, path string, tx ...pgx.Tx) ([]models.Node, error) {
	prefixes := pathPrefixes(path)
	if len(prefixes) == 0 {
		return []models.Node{}, nil // root node has no ancestors
	}

	query := `
	SELECT id, stream_id, topic, path, is_leaf, generated, created_at
	FROM nodes
	WHERE stream_id = $1 AND path = ANY($2::text[])
	ORDER BY path
	`
	rows, err := queryOrTx(tx, query, streamID, prefixes)
	if err != nil {
		return nil, fmt.Errorf("query ancestor nodes: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.Node, 0, len(prefixes))
	for rows.Next() {
		var n models.Node
		if err := rows.Scan(&n.ID, &n.StreamID, &n.Topic, &n.Path, &n.IsLeaf, &n.Generated, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan ancestor node: %w", err)
		}
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ancestor nodes: %w", err)
	}
	return nodes, nil
}

// pathPrefixes returns all proper prefixes of a dot-joined path:
// "0001.0001.0005" → ["0001", "0001.0001"]. A root path (no dots) → nil.
func pathPrefixes(path string) []string {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return nil
	}
	prefixes := make([]string, 0, len(parts)-1)
	cur := parts[0]
	for i := 1; i < len(parts); i++ {
		prefixes = append(prefixes, cur)
		cur += "." + parts[i]
	}
	return prefixes
}

// GetNodesByStream returns every node of a stream, ordered by path.
func GetNodesByStream(streamID uuid.UUID, tx ...pgx.Tx) ([]models.Node, error) {
	query := `
	SELECT id, stream_id, topic, path, is_leaf, generated, created_at
	FROM nodes
	WHERE stream_id = $1
	ORDER BY path
	`
	rows, err := queryOrTx(tx, query, streamID)
	if err != nil {
		return nil, fmt.Errorf("query stream nodes: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.Node, 0)
	for rows.Next() {
		var n models.Node
		if err := rows.Scan(&n.ID, &n.StreamID, &n.Topic, &n.Path, &n.IsLeaf, &n.Generated, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan stream node: %w", err)
		}
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stream nodes: %w", err)
	}
	return nodes, nil
}

// ClaimNodeGeneration atomically reserves a leaf for generation, so concurrent
// feed fills never expand/generate the same node twice. Returns whether this
// caller won the claim (false = already claimed or generated).
func ClaimNodeGeneration(nodeID uuid.UUID, tx ...pgx.Tx) (bool, error) {
	query := `UPDATE nodes SET generated = true WHERE id = $1 AND generated = false`
	tag, err := execOrTx(tx, query, nodeID)
	if err != nil {
		return false, fmt.Errorf("claim node generation: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ReleaseNodeGeneration reverts a claim after a failed generation attempt.
func ReleaseNodeGeneration(nodeID uuid.UUID, tx ...pgx.Tx) error {
	query := `UPDATE nodes SET generated = false WHERE id = $1 AND generated = true`
	if _, err := execOrTx(tx, query, nodeID); err != nil {
		return fmt.Errorf("release node generation: %w", err)
	}
	return nil
}

// GetStreamTree returns every node of a stream belonging to the user, with its
// article status (nil when no article exists yet), ordered by path. The join on
// streams enforces ownership — an empty result means the stream is missing or
// not the user's.
func GetStreamTree(userID, streamID uuid.UUID, tx ...pgx.Tx) ([]models.TreeNode, error) {
	query := `
	SELECT n.id, n.topic, n.path, n.is_leaf, n.generated, LOWER(m.status)
	FROM nodes n
	JOIN streams s ON s.id = n.stream_id
	LEFT JOIN article_metadata m ON m.node_id = n.id
	WHERE s.user_id = $1 AND n.stream_id = $2
	ORDER BY n.path
	`
	rows, err := queryOrTx(tx, query, userID, streamID)
	if err != nil {
		return nil, fmt.Errorf("query stream tree: %w", err)
	}
	defer rows.Close()

	tree := make([]models.TreeNode, 0)
	for rows.Next() {
		var t models.TreeNode
		if err := rows.Scan(&t.NodeID, &t.Topic, &t.Path, &t.IsLeaf, &t.Generated, &t.Status); err != nil {
			return nil, fmt.Errorf("scan stream tree node: %w", err)
		}
		tree = append(tree, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stream tree: %w", err)
	}
	return tree, nil
}

// GetUngeneratedLeaves returns up to limit leaf nodes across the user's
// streams that do not yet have an article, ordered by path.
func GetUngeneratedLeaves(userID uuid.UUID, limit int, tx ...pgx.Tx) ([]models.Node, error) {
	query := `
	SELECT n.id, n.stream_id, n.topic, n.path, n.is_leaf, n.generated, n.created_at
	FROM nodes n
	JOIN streams s ON s.id = n.stream_id
	WHERE s.user_id = $1 AND n.is_leaf = true AND n.generated = false
	ORDER BY n.path
	LIMIT $2
	`
	rows, err := queryOrTx(tx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query ungenerated leaves: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.Node, 0, limit)
	for rows.Next() {
		var n models.Node
		if err := rows.Scan(&n.ID, &n.StreamID, &n.Topic, &n.Path, &n.IsLeaf, &n.Generated, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan ungenerated leaf: %w", err)
		}
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ungenerated leaves: %w", err)
	}
	return nodes, nil
}

// GetNodeByID returns a single node.
func GetNodeByID(nodeID uuid.UUID, tx ...pgx.Tx) (models.Node, error) {
	query := `
	SELECT id, stream_id, topic, path, is_leaf, generated, created_at
	FROM nodes
	WHERE id = $1
	`
	var n models.Node
	err := queryRowOrTx(tx, query, nodeID).Scan(
		&n.ID, &n.StreamID, &n.Topic, &n.Path, &n.IsLeaf, &n.Generated, &n.CreatedAt,
	)
	if err != nil {
		return models.Node{}, fmt.Errorf("get node: %w", err)
	}
	return n, nil
}
