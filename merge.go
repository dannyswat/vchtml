package vchtml

import (
	"fmt"
	"strconv"
	"strings"
)

// Merge combines two concurrent deltas.
func Merge(baseHTML string, deltaA, deltaB *Delta) (string, *Delta, []Conflict, error) {
	// Verify base
	baseHash := hashString(baseHTML)
	if deltaA.BaseHash != baseHash || deltaB.BaseHash != baseHash {
		return "", nil, nil, fmt.Errorf("base hash mismatch")
	}

	conflicts := detectConflicts(deltaA.Operations, deltaB.Operations)
	if len(conflicts) > 0 {
		return "", nil, conflicts, nil
	}

	// Transform B against A
	opsA := deltaA.Operations
	opsBTransformed := make([]Operation, len(deltaB.Operations))
	copy(opsBTransformed, deltaB.Operations)

	// In a full implementation, we'd transform each opB against all opsA.
	// Since opsA are executed sequentially, the document state changes.
	// We need to adjust B's paths/indices to look like they are applied AFTER A.

	for i := range opsBTransformed {
		for _, opA := range opsA {
			var err error
			opsBTransformed[i], err = transformOp(opsBTransformed[i], opA)
			if err != nil {
				return "", nil, nil, err
			}
		}
	}

	mergedOps := append(opsA, opsBTransformed...)

	mergedDelta := &Delta{
		BaseHash:   baseHash,
		Operations: mergedOps,
		Author:     "system-merge",
		Timestamp:  deltaA.Timestamp, // or current
	}

	// Apply
	patched, err := Patch(baseHTML, mergedDelta)
	return patched, mergedDelta, nil, err
}

func detectConflicts(opsA, opsB []Operation) []Conflict {
	var conflicts []Conflict
	// We use string representation of Path for map keys
	mapA := make(map[string]Operation)
	for _, op := range opsA {
		mapA[pathKey(op)] = op
	}

	for _, opB := range opsB {
		// Check direct path conflicts
		keyB := pathKey(opB)
		if opA, exists := mapA[keyB]; exists {
			// Same node conflict?
			if isConflict(opA, opB) {
				conflicts = append(conflicts, Conflict{
					Type:        "Direct",
					Description: fmt.Sprintf("Conflict on node %v: %s vs %s", opB.Path, opA.Type, opB.Type),
					Path:        opB.Path,
					Ops:         []Operation{opA, opB},
				})
			}
		}

		// Check Ancestry conflicts (Delete vs Edit)
		// If A deletes a node, and B edits a child of that node.
		for _, opA := range opsA {
			if opA.Type == OpDeleteNode {
				if isDescendant(opA.Path, opB.Path) {
					conflicts = append(conflicts, Conflict{
						Type:        "Structure",
						Description: "Modification of deleted node",
						Path:        opB.Path,
						Ops:         []Operation{opA, opB},
					})
				}
			}
			if opB.Type == OpDeleteNode {
				if isDescendant(opB.Path, opA.Path) {
					conflicts = append(conflicts, Conflict{
						Type:        "Structure",
						Description: "Modification of deleted node",
						Path:        opA.Path,
						Ops:         []Operation{opA, opB},
					})
				}
			}
		}
	}
	return conflicts
}

func isConflict(a, b Operation) bool {
	if a.Type == OpDeleteNode || b.Type == OpDeleteNode {
		// Delete vs Any is conflict (unless both delete, which we might support as no-op)
		if a.Type == OpDeleteNode && b.Type == OpDeleteNode {
			return false // Idempotent
		}
		return true
	}
	if a.Type == OpUpdateText && b.Type == OpUpdateText {
		return a.NewValue != b.NewValue
	}
	if a.Type == OpUpdateAttr && b.Type == OpUpdateAttr {
		if a.Key == b.Key {
			return a.NewValue != b.NewValue
		}
		// Different keys are fine
		return false
	}
	// Insert vs Insert at same parent?
	// Path for Insert is Parent.
	// We allow concurrent inserts at the same position. Determining order is done by specific merge strategy (e.g. A before B).
	// So we return false.
	if a.Type == OpInsertNode && b.Type == OpInsertNode {
		if a.Position == b.Position {
			return false
		}
	}
	return false
}

func pathKey(op Operation) string {
	// For Insert, path is Parent. But conflict logic might need to distinguish position?
	// If Insert, unique key includes position?
	// Or we use Path to Node.
	// Ideally we key by "Target Node Identity".
	// For Delete/Update, Path is the node.
	// For Insert, Path is Parent. The "Target" is (Parent, Position).
	// But Position is dynamic.
	// Let's simplified key: Path array.
	s := strings.Trim(fmt.Sprint(op.Path), "[]")
	if op.Type == OpInsertNode {
		return s + ":I:" + strconv.Itoa(op.Position)
	}
	return s
}

func isDescendant(ancestor, child NodePath) bool {
	if len(child) <= len(ancestor) {
		return false
	}
	for i := range ancestor {
		if child[i] != ancestor[i] {
			return false
		}
	}
	return true
}

// transformOp adjusts opB (which assumes State 0) to be valid on State 1 (after opA).
func transformOp(b, a Operation) (Operation, error) {
	newB := b

	// Case 1: A Inserted a node
	if a.Type == OpInsertNode {
		// Does A's insertion affect B's Path?
		// A inserted at `a.Path` (Parent), index `a.Position`.

		// If B is in same Parent (Sibling)
		// Check if B.Path == A.Path (Parent Match for Insert Ops) or B.Path.Parent == A.Path (for other ops)

		// 1. B is InsertNode (Path is Parent)
		if pathEqual(b.Path, a.Path) {
			// Same parent.
			if a.Position <= b.Position {
				// A inserted before B. Shift B.
				newB.Position++
			}
		} else if isSiblingAffected(a.Path, a.Position, b.Path) {
			// B targets a node that is a Sibling of the inserted node (or descendant of a sibling).
			// Adjust the index in B.Path.
			// Path is `[... ParentIdx, ChildIdx, ...]`
			// A.Path is `[... ParentIdx]`
			// We check the index at `len(a.Path)`.
			idx := b.Path[len(a.Path)]
			if a.Position <= idx {
				newB.Path = make(NodePath, len(b.Path))
				copy(newB.Path, b.Path)
				newB.Path[len(a.Path)]++
			}
		}
	}

	// Case 2: A Deleted a node
	if a.Type == OpDeleteNode {
		// A deleted at `a.Path`.
		// `a.Path` ends with the Index being deleted.
		parentPath := a.Path[:len(a.Path)-1]
		delIndex := a.Path[len(a.Path)-1]

		// 1. B is InsertNode (Path is Parent)
		if pathEqual(b.Path, parentPath) {
			// Insert into same parent as deleted node.
			if delIndex < b.Position {
				newB.Position--
			}
			// If delIndex == b.Position?
			// Insert at 5. Delete 5.
			// Insert happens "Before 5". Delete "Removes 5".
			// If we do A (Delete 5) then B (Insert 5). B inserts at the *new* 5.
			// Original 5 is gone. Original 6 is at 5. B inserts before 6.
			// Seems correct to use Position 5.
		} else if isSiblingAffected(parentPath, delIndex, b.Path) {
			// Descendant of sibling.
			idx := b.Path[len(parentPath)]
			if delIndex < idx {
				newB.Path = make(NodePath, len(b.Path))
				copy(newB.Path, b.Path)
				newB.Path[len(parentPath)]--
			} else if delIndex == idx {
				// B targets the Deleted Node!
				// This should have been caught by Conflict Detection.
				// But strict Transform might say "Do nothing" or "Error".
				// We assume conflict detector caught it.
			}
		}
	}

	return newB, nil
}

func pathEqual(a, b NodePath) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Check if `path` passes through a sibling of `insertParent` at `insertIdx` that needs shifting.
// Condition: `path` has `insertParent` as prefix, and `path` longer.
// And `path[len(insertParent)]` >= `insertIdx`.
func isSiblingAffected(parent NodePath, index int, target NodePath) bool {
	if len(target) <= len(parent) {
		return false
	}
	// Prefix match
	for i := range parent {
		if target[i] != parent[i] {
			return false
		}
	}
	// Check sibling index
	if target[len(parent)] >= index {
		return true
	}
	return false
}
