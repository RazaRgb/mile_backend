package db

import (
	"backend/src/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// InsertStream persists a new stream for a user.
func InsertStream(stream models.Stream, tx ...pgx.Tx) error {
	query := `
	INSERT INTO streams (id, user_id, topic, feedback, created_at)
	VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := execOrTx(tx, query, stream.ID, stream.UserID, stream.Topic, stream.Feedback, stream.CreatedAt); err != nil {
		return fmt.Errorf("insert stream: %w", err)
	}
	return nil
}

// GetUserStreams returns all streams belonging to a user, newest first.
func GetUserStreams(userID uuid.UUID, tx ...pgx.Tx) ([]models.Stream, error) {
	query := `
	SELECT id, user_id, topic, feedback, created_at
	FROM streams
	WHERE user_id = $1
	ORDER BY created_at DESC
	`
	rows, err := queryOrTx(tx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user streams: %w", err)
	}
	defer rows.Close()

	streams := make([]models.Stream, 0)
	for rows.Next() {
		var s models.Stream
		if err := rows.Scan(&s.ID, &s.UserID, &s.Topic, &s.Feedback, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan stream: %w", err)
		}
		streams = append(streams, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate streams: %w", err)
	}
	return streams, nil
}

// GetAllStreams returns every stream in the database (used by debug tooling).
func GetAllStreams(tx ...pgx.Tx) ([]models.Stream, error) {
	query := `
	SELECT id, user_id, topic, feedback, created_at
	FROM streams
	ORDER BY created_at
	`
	rows, err := queryOrTx(tx, query)
	if err != nil {
		return nil, fmt.Errorf("query all streams: %w", err)
	}
	defer rows.Close()

	streams := make([]models.Stream, 0)
	for rows.Next() {
		var s models.Stream
		if err := rows.Scan(&s.ID, &s.UserID, &s.Topic, &s.Feedback, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan stream: %w", err)
		}
		streams = append(streams, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate streams: %w", err)
	}
	return streams, nil
}

// DeleteStream removes a stream owned by the given user. Deleting a stream
// cascades to its nodes, articles, and article metadata (ON DELETE CASCADE).
// Returns whether anything was actually deleted (false = missing or not owned).
func DeleteStream(streamID, userID uuid.UUID, tx ...pgx.Tx) (bool, error) {
	query := `DELETE FROM streams WHERE id = $1 AND user_id = $2`
	tag, err := execOrTx(tx, query, streamID, userID)
	if err != nil {
		return false, fmt.Errorf("delete stream: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// GetStream returns the stream with the given ID (WIP — implementation will
// be filled in when the streams API is wired up).
func GetStream(streamID uuid.UUID) (models.Stream, error) {
	query := `
	SELECT id, user_id, topic, feedback, created_at
	FROM streams
	WHERE id = $1
	`
	var s models.Stream
	err := DB.QueryRow(context.Background(), query, streamID).Scan(
		&s.ID,
		&s.UserID,
		&s.Topic,
		&s.Feedback,
		&s.CreatedAt,
	)
	if err != nil {
		return models.Stream{}, fmt.Errorf("get stream: %w", err)
	}
	return s, nil
}
