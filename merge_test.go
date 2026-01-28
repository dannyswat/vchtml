package vchtml

import (
	"testing"
)

func TestMergeTextConflict(t *testing.T) {
	baseHTML := `<p>Hello World</p>`

	// Delta A: Insert "Go " at index 6 (after "Hello ")
	// Result A: Hello Go World
	deltaA, err := Diff(baseHTML, `<p>Hello Go World</p>`, "A")
	if err != nil {
		t.Fatal(err)
	}

	// Delta B: Insert "!" at index 11 (End)
	// Result B: Hello World!
	deltaB, err := Diff(baseHTML, `<p>Hello World!</p>`, "B")
	if err != nil {
		t.Fatal(err)
	}

	mergedHTML, _, conflicts, err := Merge(baseHTML, deltaA, deltaB)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}
	if len(conflicts) > 0 {
		t.Fatalf("Unexpected conflicts: %v", conflicts)
	}

	// Expected: Hello Go World!
	// Position of "!" might shift if A inserted before it?
	// Hello (5) + ' ' (1) = Index 6.
	// A inserts at 6.
	// B inserts at 11 (End).
	// Since 11 > 6, B should start after A.
	// A adds 3 chars ("Go "). So B should be at 11 + 3 = 14.
	// "Hello Go World" is 14 chars. So "!" at 14 is correct.

	wanted := `<p>Hello Go World!</p>`

	// Comparison
	if !compareHTML(t, mergedHTML, wanted) {
		t.Errorf("Merge incorrect.")
	}
}

func TestMergeTextConflictInterleaved(t *testing.T) {
	baseHTML := `<p>ABCD</p>`

	// A: Insert X after B (Pos 2) -> ABXCD
	deltaA, _ := Diff(baseHTML, `<p>ABXCD</p>`, "A")

	// B: Insert Y after C (Pos 3) -> ABCYD
	deltaB, _ := Diff(baseHTML, `<p>ABCYD</p>`, "B")

	merged, _, conflicts, err := Merge(baseHTML, deltaA, deltaB)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) > 0 {
		t.Fatalf("Unexpected conflicts: %v", conflicts)
	}

	// Expected: ABXCYD
	// A inserts X at 2.
	// B inserts Y at 3.
	// 3 >= 2. So B shifts by 1 (len X). New B pos = 4.
	// Apply A: A B X C D
	//          0 1 2 3 4
	// Apply B at 4: A B X C Y D

	want := `<p>ABXCYD</p>`
	compareHTML(t, merged, want)
}

func compareHTML(t *testing.T, got, want string) bool {
	gDoc, _ := ParseHTML(got)
	wDoc, _ := ParseHTML(want)
	gStr, _ := RenderNode(gDoc)
	wStr, _ := RenderNode(wDoc)
	if gStr != wStr {
		t.Logf("Want: %s", wStr)
		t.Logf("Got:  %s", gStr)
		return false
	}
	return true
}
