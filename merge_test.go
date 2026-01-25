package vchtml

import (
	"testing"
)

func TestMerge(t *testing.T) {
	baseHTML := `<ul><li>A</li><li>B</li></ul>`

	// Delta A: Insert X at 0
	deltaA, _ := Diff(baseHTML, `<ul><li>X</li><li>A</li><li>B</li></ul>`, "A")

	// Delta B: Insert Y at 2 (Append)
	deltaB, _ := Diff(baseHTML, `<ul><li>A</li><li>B</li><li>Y</li></ul>`, "B")

	mergedHTML, _, conflicts, err := Merge(baseHTML, deltaA, deltaB)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}
	if len(conflicts) > 0 {
		t.Fatalf("Unexpected conflicts: %v", conflicts)
	}

	// Expected: X, A, B, Y
	// But note: order of X depending on exact implementation.
	// A inserted X at 0.
	// B inserted Y at 2 (end).
	// Result structure: <ul><li>X</li><li>A</li><li>B</li><li>Y</li></ul>

	wanted := `<ul><li>X</li><li>A</li><li>B</li><li>Y</li></ul>`

	// Normalize (Parse/Render) to avoid string diff issues
	wantDoc, _ := ParseHTML(wanted)
	wantStr, _ := RenderNode(wantDoc)

	gotDoc, _ := ParseHTML(mergedHTML)
	gotStr, _ := RenderNode(gotDoc)

	if gotStr != wantStr {
		t.Errorf("Merge mismatch.\nWant: %s\nGot:  %s", wantStr, gotStr)
	}
}

func TestMergeAll(t *testing.T) {
	baseHTML := `<div><p>Start</p><p>Line 1</p></div>`

	// Delta 1: Change text to "First"
	delta1, _ := Diff(baseHTML, `<div><p>Start</p><p>Line 2</p></div>`, "User1")

	// Delta 2: Add attribute to div
	delta2, _ := Diff(baseHTML, `<div><p>Starts</p><p>Line 1</p></div>`, "User2")

	mergedHTML, _, conflicts, err := MergeAll(baseHTML, []*Delta{delta1, delta2})
	if err != nil {
		t.Fatalf("MergeAll failed: %v", err)
	}
	if len(conflicts) > 0 {
		t.Fatalf("Unexpected conflicts: %v", conflicts)
	}

	// Expected: <div><p>Starts</p><p>Line 2</p></div>
	wanted := `<div><p>Starts</p><p>Line 2</p></div>`

	// Normalize (Parse/Render) to avoid string diff issues
	wantDoc, _ := ParseHTML(wanted)
	wantStr, _ := RenderNode(wantDoc)

	gotDoc, _ := ParseHTML(mergedHTML)
	gotStr, _ := RenderNode(gotDoc)

	if gotStr != wantStr {
		t.Errorf("MergeAll mismatch.\nWant: %s\nGot:  %s", wantStr, gotStr)
	}
}

func TestConflict(t *testing.T) {
	baseHTML := `<div>Text</div>`

	// Delta A: Change to "A"
	deltaA, _ := Diff(baseHTML, `<div>A</div>`, "A")

	// Delta B: Change to "B"
	deltaB, _ := Diff(baseHTML, `<div>B</div>`, "B")

	_, _, conflicts, _ := Merge(baseHTML, deltaA, deltaB)
	if len(conflicts) == 0 {
		t.Fatal("Expected conflict, got none")
	}
	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(conflicts))
	}
}
