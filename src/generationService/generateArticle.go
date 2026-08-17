package generationservice

import (
	"backend/src/db"
	"backend/src/debug"
	"backend/src/models"
	"backend/src/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrNoArticleAvailable is returned by GetNextArticle when every leaf node in
// the stream already has a generated article.
var ErrNoArticleAvailable = errors.New("no ungenerated leaf nodes in stream")

// GenerateArticle writes article content for a node's topic via the LLM and
// stores it (upserting by node_id). The node's ancestor topics are included in
// the prompt so the model knows the surrounding learning context.
func GenerateArticle(node models.Node) (models.Article, error) {
	ancestors, err := db.GetAncestorNodes(node.StreamID, node.Path)
	if err != nil {
		return models.Article{}, fmt.Errorf("get ancestor nodes: %w", err)
	}

	p := utils.CreateGenerationPrompt(node, contextChain(ancestors, node))

	content, err := utils.CallLLM(utils.ArticleLLM, p.User, p.System)
	if err != nil {
		return models.Article{}, fmt.Errorf("generate article: %w", err)
	}

	article := models.Article{
		NodeID:    node.ID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	// Persist article + fresh metadata atomically; a new article starts
	// unwatched and its node is marked as generated.
	if err := db.RunInTransaction(func(tx pgx.Tx) error {
		if err := db.UpsertArticle(article, tx); err != nil {
			return err
		}
		if err := db.UpsertArticleMetadata(models.ArticleMetadata{
			NodeID: node.ID,
			Status: models.StatusUnwatched,
		}, tx); err != nil {
			return err
		}
		return db.UpdateNodeGenerated(node.ID, true, tx)
	}); err != nil {
		return models.Article{}, fmt.Errorf("store article: %w", err)
	}
	return article, nil
}

// SeedStream creates the stream and its level-1 roadmap atomically: the
// expansion LLM splits the stream topic into top-level subtopics, which become
// root nodes with paths "0001", "0002", ... All writes happen in one
// transaction, so a failed LLM call leaves no orphan stream behind.
func SeedStream(stream models.Stream) ([]models.Node, error) {
	// Roots have no ancestors, so the learning path is just the topic itself.
	p := utils.CreateExpansionPrompt(models.Node{Topic: stream.Topic}, stream.Topic)

	resp, err := utils.CallLLMStructured(utils.ExpandLLM, p, utils.ExpansionSchema)
	if err != nil {
		return nil, fmt.Errorf("seed stream: %w", err)
	}

	subtopics, err := parseSubtopics(resp)
	if err != nil {
		return nil, fmt.Errorf("seed stream: %w", err)
	}

	nodes := make([]models.Node, 0, len(subtopics))
	for i, topic := range subtopics {
		nodes = append(nodes, models.Node{
			ID:        uuid.New(),
			StreamID:  stream.ID,
			Topic:     topic,
			Path:      fmt.Sprintf("%04d", i+1),
			IsLeaf:    true,
			CreatedAt: time.Now(),
		})
	}

	err = db.RunInTransaction(func(tx pgx.Tx) error {
		if err := db.InsertStream(stream, tx); err != nil {
			return err
		}
		for i := range nodes {
			if err := db.InsertNode(nodes[i], tx); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("persist seeded stream: %w", err)
	}
	dumpTrees() // new roadmap nodes changed the tree
	return nodes, nil
}

// dumpTrees refreshes the per-stream tree dump files after the content engine
// changed the tree. Failures are logged, never fatal to generation.
func dumpTrees() {
	if _, err := debug.DumpStreamTrees(debug.DumpDir); err != nil {
		slog.Warn("tree dump after generation failed", "error", err)
	}
}

// ExpandNode splits a node's topic into child subtopics via structured LLM
// output and persists the children atomically (children + parent leaf flag).
// The node's ancestor chain is included in the prompt so the LLM knows the
// context it is expanding within.
func ExpandNode(node models.Node) ([]models.Node, error) {
	ancestors, err := db.GetAncestorNodes(node.StreamID, node.Path)
	if err != nil {
		return nil, fmt.Errorf("get ancestor nodes: %w", err)
	}

	resp, err := utils.CallLLMStructured(utils.ExpandLLM, utils.CreateExpansionPrompt(node, contextChain(ancestors, node)), utils.ExpansionSchema)
	if err != nil {
		return nil, fmt.Errorf("expand node: %w", err)
	}

	subtopics, err := parseSubtopics(resp)
	if err != nil {
		return nil, fmt.Errorf("expand node: %w", err)
	}

	children := make([]models.Node, 0, len(subtopics))
	for i, topic := range subtopics {
		children = append(children, models.Node{
			ID:        uuid.New(),
			StreamID:  node.StreamID,
			Topic:     topic,
			Path:      childPath(node, i),
			IsLeaf:    true,
			CreatedAt: time.Now(),
		})
	}

	// Persist all children and flip the parent's leaf flag in one transaction.
	result, err := db.RunInTransactionWithReturn(func(tx pgx.Tx) (any, error) {
		for i := range children {
			if err := db.InsertNode(children[i], tx); err != nil {
				return nil, err
			}
		}
		if err := db.UpdateNodeIsLeaf(node.ID, false, tx); err != nil {
			return nil, err
		}
		// The node was claimed (generated=true) before the walk but has no
		// article; now that it is internal, clear the flag so the tree dumps
		// don't show a false ●. (Nodes with real articles are never expanded.)
		if err := db.UpdateNodeGenerated(node.ID, false, tx); err != nil {
			return nil, err
		}
		return children, nil
	})
	if err != nil {
		return nil, fmt.Errorf("persist expanded nodes: %w", err)
	}

	return result.([]models.Node), nil
}

func VerifyLeaf(node models.Node) (bool, error) {
	ancestors, err := db.GetAncestorNodes(node.StreamID, node.Path)
	if err != nil {
		return false, fmt.Errorf("get ancestor nodes: %w", err)
	}

	resp, err := utils.CallLLMStructured(utils.ExpandLLM, utils.CreateLeafVerificationPrompt(node, contextChain(ancestors, node)), utils.LeafVerificationSchema)
	if err != nil {
		return false, fmt.Errorf("verify leaf: %w", err)
	}

	verdict, err := parseLeafVerification(resp)
	if err != nil {
		return false, fmt.Errorf("verify leaf: %w", err)
	}
	return verdict, nil
}

// parseLeafVerification extracts the boolean verdict from the LLM's JSON reply
// and errors out if the required field is missing or not a boolean.
func parseLeafVerification(resp string) (bool, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(resp), &raw); err != nil {
		return false, fmt.Errorf("parse verification response: %w", err)
	}

	val, ok := raw["can_explain_under_100_words"]
	if !ok {
		return false, fmt.Errorf("verification response missing can_explain_under_100_words")
	}

	var verdict bool
	if err := json.Unmarshal(val, &verdict); err != nil {
		return false, fmt.Errorf("verification response has non-boolean verdict: %w", err)
	}
	return verdict, nil
}

// parseSubtopics extracts the subtopics array from the LLM's JSON reply.
func parseSubtopics(resp string) ([]string, error) {
	var parsed struct {
		Subtopics []string `json:"subtopics"`
	}
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		return nil, fmt.Errorf("parse expansion response: %w", err)
	}
	if len(parsed.Subtopics) == 0 {
		return nil, fmt.Errorf("llm returned no subtopics")
	}
	return parsed.Subtopics, nil
}

const maxExpandDepth = 10

// GetNextArticle walks down the stream's smallest-path ungenerated leaf,
// expanding it until it can be explained in under 100 words, then generates
// and stores the article for that node.
//
// After expanding the smallest leaf, its first child is guaranteed to be the
// new smallest leaf (paths are zero-padded numeric segments), so following
// children[0] is equivalent to re-querying the leaf set each round.
func GetNextArticle(userID, streamID uuid.UUID) (models.FeedItem, error) {
	nodes, err := db.GetLeafNodesByUserAndStream(userID, streamID)
	if err != nil {
		return models.FeedItem{}, fmt.Errorf("get leaf nodes: %w", err)
	}

	// Smallest-path leaf that does not already have an article.
	node := models.Node{}
	found := false
	for _, n := range nodes { // ordered by path, so first ungenerated wins
		if !n.Generated {
			node = n
			found = true
			break
		}
	}
	if !found {
		return models.FeedItem{}, ErrNoArticleAvailable
	}
	item, err := nextArticleForLeaf(node)
	if err != nil {
		return models.FeedItem{}, err
	}
	dumpTrees() // expansions + generated article changed the tree
	return item, nil
}

// nextArticleForLeaf walks a leaf node (expanding it until the topic is small
// enough to be explained in under 100 words), then generates and stores its
// article.
func nextArticleForLeaf(node models.Node) (models.FeedItem, error) {
	for depth := 0; depth < maxExpandDepth; depth++ {
		leafWorthy, err := VerifyLeaf(node) // true = explainable in <100 words
		if err != nil {
			return models.FeedItem{}, fmt.Errorf("verify leaf: %w", err)
		}
		if leafWorthy {
			break
		}

		children, err := ExpandNode(node)
		if err != nil {
			return models.FeedItem{}, fmt.Errorf("expand node: %w", err)
		}
		node = children[0] // lowest-path child becomes the new candidate
	}

	article, err := GenerateArticle(node)
	if err != nil {
		return models.FeedItem{}, fmt.Errorf("generate article: %w", err)
	}
	return models.FeedItem{
		NodeID:    node.ID,
		Topic:     node.Topic,
		Path:      node.Path,
		Content:   article.Content,
		CreatedAt: article.CreatedAt,
	}, nil
}

// FillFeed returns up to count unwatched reels for the user. Existing unwatched
// articles are served first; if there are fewer than count, new articles are
// generated in parallel (up to maxConcurrentGenerations at once) to top up.
// Watched and skipped reels are excluded — skipped ones remain stored in the
// DB for later reference by the improve loop.
func FillFeed(userID uuid.UUID, count int) ([]models.FeedItem, error) {
	existing, err := db.GetAvailableFeedArticles(userID, count)
	if err != nil {
		return nil, fmt.Errorf("get available articles: %w", err)
	}

	if needed := count - len(existing); needed > 0 {
		leaves, err := db.GetUngeneratedLeaves(userID, needed)
		if err != nil {
			return nil, fmt.Errorf("get ungenerated leaves: %w", err)
		}
		claimed := claimLeaves(leaves)
		existing = append(existing, generateArticlesParallel(claimed)...)
		if len(claimed) > 0 {
			dumpTrees() // expansions + generated articles changed the tree
		}
	}
	return existing, nil
}

// claimLeaves atomically reserves the given leaves so concurrent feed fills
// never expand or generate the same node twice. Unclaimed leaves are dropped.
func claimLeaves(leaves []models.Node) []models.Node {
	claimed := make([]models.Node, 0, len(leaves))
	for _, leaf := range leaves {
		ok, err := db.ClaimNodeGeneration(leaf.ID)
		if err != nil {
			slog.Warn("claim leaf failed", "node_id", leaf.ID, "error", err)
			continue
		}
		if ok {
			claimed = append(claimed, leaf)
		}
	}
	return claimed
}

// maxConcurrentGenerations caps how many LLM generations run at once.
const maxConcurrentGenerations = 5

// generateArticlesParallel generates an article for each leaf concurrently and
// returns the successes (individual failures are logged, not fatal).
func generateArticlesParallel(leaves []models.Node) []models.FeedItem {
	items := make([]models.FeedItem, 0, len(leaves))
	var (
		itemsMu sync.Mutex
		wg      sync.WaitGroup
		sem     = make(chan struct{}, maxConcurrentGenerations)
	)

	for _, leaf := range leaves {
		wg.Add(1)
		go func(leaf models.Node) {
			defer wg.Done()
			sem <- struct{}{} // acquire a generation slot
			defer func() { <-sem }()

			item, err := nextArticleForLeaf(leaf)
			if err != nil {
				slog.Warn("feed generation failed", "node_id", leaf.ID, "error", err)
				if relErr := db.ReleaseNodeGeneration(leaf.ID); relErr != nil {
					slog.Warn("release leaf claim failed", "node_id", leaf.ID, "error", relErr)
				}
				return
			}
			itemsMu.Lock()
			items = append(items, item)
			itemsMu.Unlock()
		}(leaf)
	}
	wg.Wait()
	return items
}

// contextChain renders the learning path "root → … → parent → node" using the
// ancestors (ordered shallowest-first) plus the node itself.
func contextChain(ancestors []models.Node, node models.Node) string {
	parts := make([]string, 0, len(ancestors)+1)
	for _, a := range ancestors {
		parts = append(parts, a.Topic)
	}
	parts = append(parts, node.Topic)
	return strings.Join(parts, " → ")
}

func childPath(parent models.Node, index int) string {
	return fmt.Sprintf("%s.%04d", parent.Path, index+1)
}
