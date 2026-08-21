package utils

import (
	"backend/src/models"
	"strings"
	"testing"
)

func TestCreateExpansionPromptIncludesContext(t *testing.T) {
	node := models.Node{Topic: "Goroutines"}
	p := CreateExpansionPrompt(node, "Go concurrency → Goroutines")

	if !strings.Contains(p.User, "Go concurrency → Goroutines") {
		t.Errorf("prompt missing learning path: %q", p.User)
	}
	if !strings.Contains(p.User, "Goroutines") {
		t.Errorf("prompt missing topic: %q", p.User)
	}
	// Tier-1 anchoring: the system prompt must name the ROOT topic and forbid drift.
	if !strings.Contains(p.System, "Go concurrency") {
		t.Errorf("system prompt missing root topic: %q", p.System)
	}
	if !strings.Contains(p.System, "tangential") {
		t.Errorf("system prompt should forbid tangential topics: %q", p.System)
	}
}

func TestCreateExpansionPromptRootNode(t *testing.T) {
	p := CreateExpansionPrompt(models.Node{Topic: "Go concurrency"}, "Go concurrency")

	if !strings.Contains(p.User, "Go concurrency") {
		t.Errorf("prompt missing topic: %q", p.User)
	}
	if !strings.Contains(p.System, "Go concurrency") {
		t.Errorf("system prompt missing root topic: %q", p.System)
	}
}

func TestCreateLeafVerificationPromptIncludesContext(t *testing.T) {
	p := CreateLeafVerificationPrompt(models.Node{Topic: "Channels"}, "Go concurrency → Goroutines → Channels")

	if !strings.Contains(p.User, "Go concurrency → Goroutines → Channels") {
		t.Errorf("prompt missing learning path: %q", p.User)
	}
	if !strings.Contains(p.System, "Go concurrency") {
		t.Errorf("system prompt missing root topic: %q", p.System)
	}
}

func TestRootTopic(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "Go concurrency", want: "Go concurrency"},
		{path: "Go concurrency → Goroutines", want: "Go concurrency"},
		{path: "ML basics → Data collection → Web scraping", want: "ML basics"},
		{path: "", want: ""},
	}
	for _, tt := range tests {
		if got := rootTopic(tt.path); got != tt.want {
			t.Errorf("rootTopic(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
