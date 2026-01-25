package vchtml

import (
	"testing"

	"golang.org/x/net/html"
)

func TestPathing(t *testing.T) {
	htmlStr := `<html><head></head><body><div><p>Hello</p></div></body></html>`
	doc, err := ParseHTML(htmlStr)
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// The structure should be roughly:
	// root -> html (0) -> body (1) -> div (0) -> p (0) -> text "Hello" (0)
	// Target: the text node "Hello"
	// Path should be: [0, 1, 0, 0, 0]
	targetPath := NodePath{0, 1, 0, 0, 0}

	node, err := GetNode(doc, targetPath)
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}

	if node.Type != html.TextNode {
		t.Errorf("Expected TextNode, got %d", node.Type)
	}
	if node.Data != "Hello" {
		t.Errorf("Expected node data 'Hello', got '%s'", node.Data)
	}

	// Now test reverse: GetPath
	path, err := GetPath(doc, node)
	if err != nil {
		t.Fatalf("GetPath failed: %v", err)
	}

	if len(path) != len(targetPath) {
		t.Fatalf("Path length mismatch. Got %v, want %v", path, targetPath)
	}

	for i := range path {
		if path[i] != targetPath[i] {
			t.Errorf("Path mismatch at index %d. Got %d, want %d", i, path[i], targetPath[i])
		}
	}
}
