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
}

func TestCreateExpansionPromptRootNode(t *testing.T) {
	p := CreateExpansionPrompt(models.Node{Topic: "Go concurrency"}, "Go concurrency")

	if !strings.Contains(p.User, "Go concurrency") {
		t.Errorf("prompt missing topic: %q", p.User)
	}
}

func TestCreateLeafVerificationPromptIncludesContext(t *testing.T) {
	p := CreateLeafVerificationPrompt(models.Node{Topic: "Channels"}, "Go concurrency → Goroutines → Channels")

	if !strings.Contains(p.User, "Go concurrency → Goroutines → Channels") {
		t.Errorf("prompt missing learning path: %q", p.User)
	}
	if !strings.Contains(p.System, "learning path") {
		t.Errorf("system prompt should reference the learning path: %q", p.System)
	}
}
