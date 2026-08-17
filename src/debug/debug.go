package debug

import (
	"backend/src/db"
	"backend/src/utils"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DumpDir is where stream tree dumps are written (relative to the working dir).
const DumpDir = "tree_dumps"

// DumpStreamTrees writes one text file per stream into dir, containing a clean
// render of that stream's node tree. Each call overwrites the files, so they
// always reflect the current structure. Returns the paths written.
func DumpStreamTrees(dir string) ([]string, error) {
	streams, err := db.GetAllStreams()
	if err != nil {
		return nil, fmt.Errorf("list streams: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create dump dir: %w", err)
	}

	files := make([]string, 0, len(streams))
	for _, s := range streams {
		nodes, err := db.GetNodesByStream(s.ID)
		if err != nil {
			return nil, fmt.Errorf("get nodes for stream %s: %w", s.ID, err)
		}

		body := fmt.Sprintf(
			"Stream: %s\nID:     %s\nDumped: %s\nLegend: ● generated   ○ leaf, no article yet\n\n%s",
			s.Topic, s.ID, time.Now().Format(time.RFC3339), utils.BuildTreeText(nodes),
		)

		path := filepath.Join(dir, fmt.Sprintf("tree_%s_%s.txt", s.ID.String()[:8], slugify(s.Topic)))
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", path, err)
		}
		files = append(files, path)
	}
	return files, nil
}

// slugify turns a stream topic into a filesystem-safe filename slug.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
