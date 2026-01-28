package vchtml

import (
	"testing"
)

func TestPatchTextOps(t *testing.T) {
	// Explicitly construct ops to test Patch logic in isolation
	// Assuming Path for text node is [0, 0] (Root -> p -> Text)
	// Actually ParseHTML usually returns a Doc -> Html -> Body -> p -> Text?
	// Wait, ParseHTML in this lib uses html.Parse which returns a DocumentNode.
	// <html><head></head><body><p>...</p></body></html>
	// So Path is deeper.
	// To reliably get path, we should run Diff first.

	// Let's use RoundTrip testing which is easier given Path complexity.

	tests := []struct {
		name    string
		oldHTML string
		newHTML string
	}{
		{
			name:    "Insert Text",
			oldHTML: "<p>Hello</p>",
			newHTML: "<p>Hello World</p>",
		},
		{
			name:    "Delete Text",
			oldHTML: "<p>Hello World</p>",
			newHTML: "<p>Hello</p>",
		},
		{
			name:    "Insert Middle",
			oldHTML: "<p>ABC</p>",
			newHTML: "<p>A B C</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta, err := Diff(tt.oldHTML, tt.newHTML, "tester")
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}

			patched, err := Patch(tt.oldHTML, delta)
			if err != nil {
				t.Fatalf("Patch() error = %v", err)
			}

			pDoc, _ := ParseHTML(patched)
			nDoc, _ := ParseHTML(tt.newHTML)
			pStr, _ := RenderNode(pDoc)
			nStr, _ := RenderNode(nDoc)

			if pStr != nStr {
				t.Errorf("Patch mismatch.\nWant: %s\nGot:  %s", nStr, pStr)
			}
		})
	}
}
