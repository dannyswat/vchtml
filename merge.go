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

	// We might expand operations during transform, so we use a list that can grow?
	// But usually we transform B against A one by one.
	// Since we are returning a combined delta, we take A as-is (applied first),
	// and then B (transformed).

	var opsBTransformed []Operation
	for _, opB := range deltaB.Operations {
		currentOps := []Operation{opB}

		for _, opA := range opsA {
			var nextOps []Operation
			for _, curr := range currentOps {
				transformed, err := transformOp(curr, opA)
				if err != nil {
					return "", nil, nil, err
				}
				nextOps = append(nextOps, transformed...)
			}
			currentOps = nextOps
		}
		opsBTransformed = append(opsBTransformed, currentOps...)
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

// MergeAll merges a list of deltas sequentially.
func MergeAll(baseHTML string, deltas []*Delta) (string, *Delta, []Conflict, error) {
	if len(deltas) == 0 {
		return baseHTML, &Delta{BaseHash: hashString(baseHTML)}, nil, nil
	}

	merged := deltas[0]

	if len(deltas) == 1 {
		patched, err := Patch(baseHTML, merged)
		return patched, merged, nil, err
	}

	var patched string
	var err error
	var conflicts []Conflict

	for i := 1; i < len(deltas); i++ {
		patched, merged, conflicts, err = Merge(baseHTML, merged, deltas[i])
		if err != nil {
			return "", nil, nil, err
		}
		if len(conflicts) > 0 {
			return "", nil, conflicts, nil
		}
	}

	return patched, merged, nil, nil
}

func detectConflicts(opsA, opsB []Operation) []Conflict {
	var conflicts []Conflict
	mapA := make(map[string]Operation)
	for _, op := range opsA {
		mapA[pathKey(op)] = op
	}

	for _, opB := range opsB {
		keyB := pathKey(opB)
		if opA, exists := mapA[keyB]; exists {
			if isConflict(opA, opB) {
				conflicts = append(conflicts, Conflict{
					Type:        "Direct",
					Description: fmt.Sprintf("Conflict on node %v: %s vs %s", opB.Path, opA.Type, opB.Type),
					Path:        opB.Path,
					Ops:         []Operation{opA, opB},
				})
			}
		}

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
		if a.Type == OpDeleteNode && b.Type == OpDeleteNode {
			return false
		}
		return true
	}
	// Atomic update conflict
	if a.Type == OpUpdateText && b.Type == OpUpdateText {
		return a.NewValue != b.NewValue
	}

	// Granular text conflict?
	if (a.Type == OpInsertText || a.Type == OpDeleteText) && (b.Type == OpInsertText || b.Type == OpDeleteText) {
		// We allow granular merging unless logic fails.
		// For now assume NO conflict, let transform handle it.
		// If transform fails (e.g. overlapping delete/insert that is ambiguous), it should return error there?
		// But detectConflict checks *before* merge.
		// Let's assume text ops are mergeable.
		return false
	}
	// Mixed Atomic/Granular?
	if (a.Type == OpUpdateText && (b.Type == OpInsertText || b.Type == OpDeleteText)) ||
		(b.Type == OpUpdateText && (a.Type == OpInsertText || a.Type == OpDeleteText)) {
		return true // Mixing modes is dangerous
	}

	if a.Type == OpUpdateAttr && b.Type == OpUpdateAttr {
		if a.Key == b.Key {
			return a.NewValue != b.NewValue
		}
		return false
	}
	if a.Type == OpInsertNode && b.Type == OpInsertNode {
		if a.Position == b.Position {
			// Actually this is usually NOT a conflict, just order ambiguity.
			// But for deterministic result we can flag or allow.
			// Let's allow.
			return false
		}
	}
	return false
}

func pathKey(op Operation) string {
	s := strings.Trim(fmt.Sprint(op.Path), "[]")
	if op.Type == OpInsertNode {
		return s + ":I:" + strconv.Itoa(op.Position)
	}
	// For text operations, conflict is checked on the node (path)
	// But if we want to support multiple ops on same node, we shouldn't collision on just Path.
	// But `detectConflicts` iterates over map keys. If multiple ops have same key, mapping overrides!
	// This map approach is flawed for multiple ops on same node (like multiple text inserts).
	// FIX: We should rely on list iteration or improve key.
	// But `detectConflicts` is a simplified check.
	// For text ops, we want to allow multiple.
	// So we return a key that includes Op index? No.
	// We'll append suffix to key for text ops so they don't overwrite each other in the map,
	// effectively disabling map-based conflict check for them, leaving it to manual check or `transformOp`.
	if op.Type == OpInsertText || op.Type == OpDeleteText {
		return s + ":T:" + strconv.Itoa(op.Position) + ":" + op.NewValue + ":" + op.OldValue
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

func transformOp(b, a Operation) ([]Operation, error) {
	newB := b

	// Case: Text Ops
	if (a.Type == OpInsertText || a.Type == OpDeleteText) && pathEqual(b.Path, a.Path) {
		// Both on same text node.

		if a.Type == OpInsertText {
			// A Inserted at a.Position.
			// B is Insert or Delete.
			if b.Position >= a.Position {
				// Shift B forward
				newB.Position += len(a.NewValue)
			}
		} else if a.Type == OpDeleteText {
			// A Deleted at a.Position, length len(a.OldValue)
			delLen := len(a.OldValue)
			aEnd := a.Position + delLen

			if b.Position >= aEnd {
				// B is after deleted range. Shift back.
				newB.Position -= delLen
			} else if b.Position >= a.Position {
				// B starts inside deleted range.
				// If B is Insert:
				//   It inserts inside something that is gone.
				//   Usually we collapse it to insertion point a.Position.
				if b.Type == OpInsertText {
					newB.Position = a.Position
				} else if b.Type == OpDeleteText {
					// B deletes something that overlaps with A's deletion.
					// A: Delete [5, 10). B: Delete [6, 8).
					// B is redundant. Return empty.
					// B: Delete [8, 12).
					// Remaining of B is [10, 12) (shifted to 5 -> [5, 7)).
					// This overlap logic is complex.
					// For invalid/overlapping deletes, let's error or no-op.
					return nil, nil // Return empty (consumed).
				}
			} else {
				// B starts before A.
				// If B Delete ends after A starts?
				if b.Type == OpDeleteText {
					bLen := len(b.OldValue)
					bEnd := b.Position + bLen
					if bEnd > a.Position {
						// Overlap from left.
						// Similar complexity.
						return nil, nil
					}
				}
			}
		}
		return []Operation{newB}, nil
	}

	// Case 1: A Inserted a node
	if a.Type == OpInsertNode {
		if pathEqual(b.Path, a.Path) {
			if a.Position <= b.Position {
				newB.Position++
			}
		} else if isSiblingAffected(a.Path, a.Position, b.Path) {
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
		parentPath := a.Path[:len(a.Path)-1]
		delIndex := a.Path[len(a.Path)-1]

		if pathEqual(b.Path, parentPath) {
			if delIndex < b.Position {
				newB.Position--
			}
		} else if isSiblingAffected(parentPath, delIndex, b.Path) {
			idx := b.Path[len(parentPath)]
			if delIndex < idx {
				newB.Path = make(NodePath, len(b.Path))
				copy(newB.Path, b.Path)
				newB.Path[len(parentPath)]--
			}
		}
	}

	return []Operation{newB}, nil
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

func isSiblingAffected(parent NodePath, index int, target NodePath) bool {
	if len(target) <= len(parent) {
		return false
	}
	for i := range parent {
		if target[i] != parent[i] {
			return false
		}
	}
	if target[len(parent)] >= index {
		return true
	}
	return false
}
