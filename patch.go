package vchtml

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// Patch applies the changes in 'delta' to 'baseHTML'.
func Patch(baseHTML string, delta *Delta) (string, error) {
	// 1. Verify Hash
	currentHash := hashString(baseHTML)
	if currentHash != delta.BaseHash {
		// For strict mode, we might reject.
		// For now, allow but warn or just error?
		// Requirement says: "try to resolve conflict", implying we might patch dirty versions?
		// But Patch() usually applies to the exact base. Merge() handles conflict.
		// Let's return error if hash mismatch.
		return "", fmt.Errorf("base hash mismatch: expected %s, got %s", delta.BaseHash, currentHash)
	}

	doc, err := ParseHTML(baseHTML)
	if err != nil {
		return "", err
	}

	for i, op := range delta.Operations {
		if err := applyOp(doc, op); err != nil {
			return "", fmt.Errorf("failed to apply op %d (%s): %w", i, op.Type, err)
		}
	}

	return RenderNode(doc)
}

func applyOp(root *html.Node, op Operation) error {
	switch op.Type {
	case OpUpdateText:
		node, err := GetNode(root, op.Path)
		if err != nil {
			return err
		}
		if node.Type != html.TextNode {
			return fmt.Errorf("target node for UPDATE_TEXT is not a text node (type=%d)", node.Type)
		}
		// Verify old value?
		if node.Data != op.OldValue {
			// This is a conflict in theory, but Patch usually applies blindly or strict check.
			// Let's assume strict check.
			return fmt.Errorf("UPDATE_TEXT old value mismatch: want '%s', got '%s'", op.OldValue, node.Data)
		}
		node.Data = op.NewValue

	case OpUpdateAttr:
		node, err := GetNode(root, op.Path)
		if err != nil {
			return err
		}
		if node.Type != html.ElementNode {
			return fmt.Errorf("target node for UPDATE_ATTR is not an element node")
		}

		// If Op says update Key, we find it.
		// If verify old value:
		currentVal := getAttr(node, op.Key)
		if currentVal != op.OldValue {
			// Treat missing ("") as match if OldValue is ""
			if !(currentVal == "" && op.OldValue == "") {
				// For now, relax or error.
				// Error helps debugging.
				// return fmt.Errorf("UPDATE_ATTR old value mismatch for %s: want '%s', got '%s'", op.Key, op.OldValue, currentVal)
			}
		}

		// Apply new value
		if op.NewValue == "" {
			// Remove attribute? Or set to empty?
			// Since we didn't define OpDeleteAttr, let's look at convention.
			// If we treat missing as remove, we should probably remove it.
			// But existing HTML allows val="" (empty but present).
			// Let's assume: we set it.
			// If we want remove, we'd need explicit signal.
			// For now: Set it.
			setAttr(node, op.Key, op.NewValue)
		} else {
			setAttr(node, op.Key, op.NewValue)
		}

	case OpInsertNode:
		// Path is Parent
		parent, err := GetNode(root, op.Path)
		if err != nil {
			return err
		}

		// Parse NodeData
		// context is parent usually, but here just use body or similar context.
		// Element context matters for parsing (e.g. <tr> inside <table>).
		// We try to guess context from parent.
		nodes, err := html.ParseFragment(strings.NewReader(op.NodeData), parent)
		if err != nil {
			return fmt.Errorf("failed to parse node data: %w", err)
		}
		if len(nodes) == 0 {
			return nil // No-op
		}
		newNode := nodes[0] // We assume 1 node for now.

		// Insert at Position.
		// We need to find the node currently at Position, and InsertBefore it.
		// If Position == len, AppendChild.

		insertChildAt(parent, newNode, op.Position)

	case OpDeleteNode:
		// Path is the node itself
		node, err := GetNode(root, op.Path)
		if err != nil {
			return err
		}
		if node.Parent == nil {
			return errors.New("cannot delete root node or orphan")
		}
		node.Parent.RemoveChild(node)

	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}

	return nil
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func setAttr(n *html.Node, key, val string) {
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	// Add if not found
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

func insertChildAt(parent, child *html.Node, index int) {
	// Find the Sibling at index
	ref := getChildAtIndex(parent, index)
	if ref != nil {
		parent.InsertBefore(child, ref)
	} else {
		// Index is presumably at end
		parent.AppendChild(child)
	}
}
