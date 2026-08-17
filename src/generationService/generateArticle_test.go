package generationservice

import (
	"backend/src/models"
	"testing"
)

func TestContextChain(t *testing.T) {
	node := models.Node{Topic: "Introduction to Goroutines", Path: "0001.0001.0001"}
	ancestors := []models.Node{
		{Topic: "Go concurrency", Path: "0001"},
		{Topic: "Goroutines", Path: "0001.0001"},
	}

	want := "Go concurrency → Goroutines → Introduction to Goroutines"
	if got := contextChain(ancestors, node); got != want {
		t.Errorf("contextChain = %q, want %q", got, want)
	}
}

func TestContextChainNoAncestors(t *testing.T) {
	node := models.Node{Topic: "Go concurrency", Path: "0001"}
	if got := contextChain(nil, node); got != "Go concurrency" {
		t.Errorf("contextChain = %q, want %q", got, "Go concurrency")
	}
}

func TestParseLeafVerification(t *testing.T) {
	tests := []struct {
		name    string
		resp    string
		want    bool
		wantErr bool
	}{
		{name: "true verdict", resp: `{"can_explain_under_100_words": true}`, want: true},
		{name: "false verdict", resp: `{"can_explain_under_100_words": false}`, want: false},
		{name: "missing field", resp: `{"something_else": 1}`, wantErr: true},
		{name: "non-boolean verdict", resp: `{"can_explain_under_100_words": "yes"}`, wantErr: true},
		{name: "garbage", resp: `not json at all`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLeafVerification(tt.resp)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
