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
		if node.Data != op.OldValue {
			return fmt.Errorf("UPDATE_TEXT old value mismatch: want '%s', got '%s'", op.OldValue, node.Data)
		}
		node.Data = op.NewValue

	case OpInsertText:
		node, err := GetNode(root, op.Path)
		if err != nil {
			return err
		}
		if node.Type != html.TextNode {
			return fmt.Errorf("target node for INSERT_TEXT is not a text node (type=%d)", node.Type)
		}
		if op.Position < 0 || op.Position > len(node.Data) {
			return fmt.Errorf("INSERT_TEXT position out of bounds: pos=%d, len=%d", op.Position, len(node.Data))
		}
		// Insert
		node.Data = node.Data[:op.Position] + op.NewValue + node.Data[op.Position:]

	case OpDeleteText:
		node, err := GetNode(root, op.Path)
		if err != nil {
			return err
		}
		if node.Type != html.TextNode {
			return fmt.Errorf("target node for DELETE_TEXT is not a text node (type=%d)", node.Type)
		}
		// Verify
		deleteLen := len(op.OldValue)
		if op.Position < 0 || op.Position+deleteLen > len(node.Data) {
			return fmt.Errorf("DELETE_TEXT position out of bounds: pos=%d, len=%d, delLen=%d", op.Position, len(node.Data), deleteLen)
		}
		actual := node.Data[op.Position : op.Position+deleteLen]
		if actual != op.OldValue {
			return fmt.Errorf("DELETE_TEXT old value mismatch: want '%s', got '%s'", op.OldValue, actual)
		}
		// Delete
		node.Data = node.Data[:op.Position] + node.Data[op.Position+deleteLen:]

	case OpUpdateAttr:
		node, err := GetNode(root, op.Path)
		if err != nil {
			return err
		}
		if node.Type != html.ElementNode {
			return fmt.Errorf("target node for UPDATE_ATTR is not an element node")
		}

		// Apply new value
		setAttr(node, op.Key, op.NewValue)

	case OpInsertNode:
		// Path is Parent
		parent, err := GetNode(root, op.Path)
		if err != nil {
			return err
		}

		nodes, err := html.ParseFragment(strings.NewReader(op.NodeData), parent)
		if err != nil {
			return fmt.Errorf("failed to parse node data: %w", err)
		}
		if len(nodes) == 0 {
			return nil // No-op
		}
		newNode := nodes[0] // We assume 1 node for now.

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
