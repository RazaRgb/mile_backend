package utils

import (
	"backend/src/models"
	"strings"
	"testing"
)

func TestBuildTreeText(t *testing.T) {
	nodes := []models.Node{
		{Path: "0001.0001", Topic: "Goroutines", IsLeaf: false},
		{Path: "0001.0001.0001", Topic: "Introduction to Goroutines", IsLeaf: true, Generated: true},
		{Path: "0001.0002", Topic: "Channels", IsLeaf: true},
		{Path: "0001", Topic: "Go concurrency", IsLeaf: false},
		{Path: "0002", Topic: "Memory management", IsLeaf: true, Generated: true},
	}

	got := BuildTreeText(nodes)

	for _, want := range []string{
		"Go concurrency",
		"├── Goroutines",
		"│   └── Introduction to Goroutines",
		"└── Channels",
		"●", // generated markers present
		"○", // ungenerated leaf marker present
	} {
		if !strings.Contains(got, want) {
			t.Errorf("tree missing %q\n---\n%s", want, got)
		}
	}
}

func TestBuildTreeTextEmpty(t *testing.T) {
	if got := BuildTreeText(nil); got != "(no nodes)" {
		t.Errorf("empty tree = %q, want %q", got, "(no nodes)")
	}
}

func TestParentPath(t *testing.T) {
	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{path: "0001", want: "", ok: false},
		{path: "0001.0001", want: "0001", ok: true},
		{path: "0001.0001.0005", want: "0001.0001", ok: true},
	}
	for _, tt := range tests {
		got, ok := parentPath(tt.path)
		if got != tt.want || ok != tt.ok {
			t.Errorf("parentPath(%q) = (%q, %v), want (%q, %v)", tt.path, got, ok, tt.want, tt.ok)
		}
	}
}
