package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID `json:"id"`
	Email          string    `json:"email"`
	Username       string    `json:"username"`
	HashedPassword string    `json:"pass_hash"`
	CreatedAt      time.Time `json:"created_at"`
}

type Stream struct {
	ID        uuid.UUID `json:"id"`
	Topic     string    `json:"topic"`
	UserID    uuid.UUID `json:"user_id"`
	Feedback  string    `json:"feedback"`
	CreatedAt time.Time `json:"created_at"`
}

type Article struct {
	NodeID    uuid.UUID `json:"node_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type ArticleMetadata struct {
	NodeID         uuid.UUID `json:"node_id"`
	Status         string    `json:"status"`
	Comment        string    `json:"comment"`
	Score          float32   `json:"score"`
	SecondsWatched int       `json:"seconds_watched"`
	CreatedAt      time.Time `json:"created_at"`
}

type Node struct {
	ID        uuid.UUID `json:"id"`
	StreamID  uuid.UUID `json:"stream_id"`
	Topic     string    `json:"topic"`
	Path      string    `json:"path"`
	IsLeaf    bool      `json:"is_leaf"`
	Generated bool      `json:"generated"`
	CreatedAt time.Time `json:"created_at"`
}

// Article status values stored in article_metadata.status.
const (
	StatusUnwatched = "UNWATCHED"
	StatusWatched   = "WATCHED"
	StatusSkipped   = "SKIPPED"
)

// FeedItem is an article plus its node info, as served to the feed.
type FeedItem struct {
	NodeID    uuid.UUID `json:"node_id"`
	Topic     string    `json:"topic"`
	Path      string    `json:"path"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
