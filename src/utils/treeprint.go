package utils

import (
	"backend/src/models"
	"fmt"
	"sort"
	"strings"
)

// treeNode is a node in the display tree built from flat paths.
type treeNode struct {
	node     models.Node
	children []*treeNode
}

// BuildTreeText renders nodes (with dot-joined paths like "0001.0001.0005")
// as a clean ASCII tree. Input order does not matter — nodes are sorted by
// path first, so parents always come before children.
//
// Markers: ● = article generated, ○ = leaf with no article yet.
func BuildTreeText(nodes []models.Node) string {
	if len(nodes) == 0 {
		return "(no nodes)"
	}

	// Sort so parents (path prefixes) are processed before their children.
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Path < nodes[j].Path })

	byPath := make(map[string]*treeNode, len(nodes))
	var roots []*treeNode
	for _, n := range nodes {
		tn := &treeNode{node: n}
		byPath[n.Path] = tn

		if parentKey, ok := parentPath(n.Path); ok {
			if parent, found := byPath[parentKey]; found {
				parent.children = append(parent.children, tn)
				continue
			}
		}
		roots = append(roots, tn)
	}

	var sb strings.Builder
	for i, r := range roots {
		sb.WriteString(nodeLabel(r.node) + "\n")
		for j, c := range r.children {
			renderBranch(&sb, c, "", j == len(r.children)-1)
		}
		if i < len(roots)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// parentPath returns the parent's path and true, or ("", false) for a root.
func parentPath(path string) (string, bool) {
	idx := strings.LastIndex(path, ".")
	if idx < 0 {
		return "", false
	}
	return path[:idx], true
}

// nodeLabel formats one node: padded topic, path, and a state marker.
func nodeLabel(n models.Node) string {
	marker := " "
	switch {
	case n.Generated:
		marker = "●"
	case n.IsLeaf:
		marker = "○"
	}
	return fmt.Sprintf("%-48s %-18s %s", n.Topic, n.Path, marker)
}

// renderBranch draws a node and its subtree with box-drawing connectors.
func renderBranch(sb *strings.Builder, tn *treeNode, prefix string, isLast bool) {
	connector := "├── "
	childPrefix := prefix + "│   "
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}
	sb.WriteString(prefix + connector + nodeLabel(tn.node) + "\n")
	for i, c := range tn.children {
		renderBranch(sb, c, childPrefix, i == len(tn.children)-1)
	}
}
