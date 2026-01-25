package vchtml

import (
	"testing"
)

func TestPatchRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		oldHTML string
		newHTML string
	}{
		{
			name:    "Text change",
			oldHTML: "<div><p>Hello</p></div>",
			newHTML: "<div><p>World</p></div>",
		},
		{
			name:    "Attribute change",
			oldHTML: `<div class="a"></div>`,
			newHTML: `<div class="b"></div>`,
		},
		{
			name:    "Insert node",
			oldHTML: `<ul><li>A</li></ul>`,
			newHTML: `<ul><li>A</li><li>B</li></ul>`,
		},
		{
			name:    "Delete node",
			oldHTML: `<ul><li>A</li><li>B</li></ul>`,
			newHTML: `<ul><li>A</li></ul>`,
		},
		{
			name:    "Complex structural change",
			oldHTML: `<div id="main"><h1>Title</h1><p>Text</p></div>`,
			newHTML: `<div id="main"><h1>New Title</h1><p>Text</p><p>Footer</p></div>`,
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

			// We need to compare semantic equivalence, as Patch might introduce slight normalization diffs
			// (e.g. whitespace, quote style).
			// So we re-parse and re-render both expected and actual.

			wantDoc, _ := ParseHTML(tt.newHTML)
			wantStr, _ := RenderNode(wantDoc)

			doc2, _ := ParseHTML(patched)
			gotStr, _ := RenderNode(doc2)

			if gotStr != wantStr {
				t.Errorf("RoundTrip failed.\nWant: %s\nGot:  %s", wantStr, gotStr)
				// Print Ops for debug
				printJSON(delta.Operations)
			}
		})
	}
}
