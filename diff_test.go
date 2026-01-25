package vchtml

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestDiffSimple(t *testing.T) {
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
			name:    "Text change",
			oldHTML: "<div><p>Hello</p></div>",
			newHTML: "<div><p>World</p></div>",
			wantOps: 1,
		},
		{
			name:    "Attribute change",
			oldHTML: `<div class="a"></div>`,
			newHTML: `<div class="b"></div>`,
			wantOps: 1,
		},
		{
			name:    "Attribute add",
			oldHTML: `<div></div>`,
			newHTML: `<div id="TEST"></div>`,
			wantOps: 1,
		},
		{
			name:    "Insert node",
			oldHTML: `<ul><li>A</li></ul>`,
			newHTML: `<ul><li>A</li><li>B</li></ul>`,
			wantOps: 1,
		},
		{
			name:    "Delete node",
			oldHTML: `<ul><li>A</li><li>B</li></ul>`,
			newHTML: `<ul><li>A</li></ul>`,
			wantOps: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta, err := Diff(tt.oldHTML, tt.newHTML, "tester")
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}
			if len(delta.Operations) != tt.wantOps {
				printJSON(delta.Operations)
				t.Errorf("Diff() generated %d ops, want %d", len(delta.Operations), tt.wantOps)
			}
		})
	}
}

func printJSON(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}
