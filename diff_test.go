package vchtml

import (
	"testing"
)

func TestDiffTextGranularity(t *testing.T) {
	tests := []struct {
		name      string
		oldHTML   string
		newHTML   string
		expectOps []OpType // Expected sequence of types (simplified)
	}{
		{
			name:      "Append Text",
			oldHTML:   "<p>Hello</p>",
			newHTML:   "<p>Hello World</p>",
			expectOps: []OpType{OpInsertText},
		},
		{
			name:      "Prepend Text",
			oldHTML:   "<p>World</p>",
			newHTML:   "<p>Hello World</p>",
			expectOps: []OpType{OpInsertText},
		},
		{
			name:      "Insert Middle",
			oldHTML:   "<p>Hello World</p>",
			newHTML:   "<p>Hello Go World</p>",
			expectOps: []OpType{OpInsertText},
		},
		{
			name:      "Delete End",
			oldHTML:   "<p>Hello World</p>",
			newHTML:   "<p>Hello</p>",
			expectOps: []OpType{OpDeleteText},
		},
		{
			name:      "Delete Middle",
			oldHTML:   "<p>Hello Go World</p>",
			newHTML:   "<p>Hello World</p>",
			expectOps: []OpType{OpDeleteText},
		},
		{
			name:      "Replace Middle/Part",
			oldHTML:   "<p>Hello Old World</p>",
			newHTML:   "<p>Hello New World</p>",
			expectOps: []OpType{OpDeleteText, OpInsertText}, // Depending on implementation order
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta, err := Diff(tt.oldHTML, tt.newHTML, "test")
			if err != nil {
				t.Fatalf("Diff failed: %v", err)
			}

			if len(delta.Operations) != len(tt.expectOps) {
				t.Errorf("Ops count mismatch. Want %d, Got %d", len(tt.expectOps), len(delta.Operations))
				for i, op := range delta.Operations {
					t.Logf("Op[%d]: %v", i, op)
				}
				return
			}

			// Verify types roughly (order might vary if multiple on same node, but here simple)
			for i, op := range delta.Operations {
				// We expect specific types.
				// Since map iteration order in diffAttributes is random, attr ops might vary,
				// but here we deal with text.
				// For text diff, order is deterministic (Delete then Insert usually).
				// We check if type matches one of expected or exact sequence.
				if op.Type != tt.expectOps[i] {
					t.Errorf("Op[%d] type mismatch. Want %s, Got %s", i, tt.expectOps[i], op.Type)
				}
			}
		})
	}
}

func TestDiffSimple(t *testing.T) {
	// Keep original basic tests
	tests := []struct {
		name    string
		oldHTML string
		newHTML string
		wantOps int
	}{
		{
			name:    "No changes",
			oldHTML: "<div><p>Hello</p></div>",
			newHTML: "<div><p>Hello</p></div>",
			wantOps: 0,
		},
		{
			name:    "Attribute change",
			oldHTML: `<div class="a"></div>`,
			newHTML: `<div class="b"></div>`,
			wantOps: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta, err := Diff(tt.oldHTML, tt.newHTML, "tester")
			if err != nil {
				t.Fatalf("Diff error: %v", err)
			}
			if len(delta.Operations) != tt.wantOps {
				t.Errorf("Want %d ops, got %d", tt.wantOps, len(delta.Operations))
			}
		})
	}
}
