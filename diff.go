package vchtml

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/net/html"
)

// Diff calculates the operations needed to transform 'oldHTML' into 'newHTML'.
func Diff(oldHTML, newHTML, author string) (*Delta, error) {
	oldDoc, err := ParseHTML(oldHTML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse old HTML: %w", err)
	}
	newDoc, err := ParseHTML(newHTML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse new HTML: %w", err)
	}

	delta := &Delta{
		BaseHash:  hashString(oldHTML),
		Timestamp: time.Now().Unix(),
		Author:    author,
	}

	ops, err := diffNodes(oldDoc, newDoc, NodePath{})
	if err != nil {
		return nil, err
	}
	delta.Operations = ops

	return delta, nil
}

func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// diffNodes compares two nodes and returns a list of operations.
// It assumes oldNode and newNode represent the "same" node in position.
func diffNodes(oldNode, newNode *html.Node, path NodePath) ([]Operation, error) {
	var ops []Operation

	// 1. Check if nodes are inherently different (e.g. different tag).
	if oldNode.Type != newNode.Type || oldNode.DataAtom != newNode.DataAtom || (oldNode.Type == html.ElementNode && oldNode.Data != newNode.Data) {
		// Structural replacement not implemented fully in this snippet, assumes structure matches.
	}

	// 2. Compare Attributes (if Element)
	if oldNode.Type == html.ElementNode {
		attrOps := diffAttributes(oldNode, newNode, path)
		ops = append(ops, attrOps...)
	}

	// 3. Compare Text (if TextNode)
	if oldNode.Type == html.TextNode {
		if oldNode.Data != newNode.Data {
			textOps := diffText(oldNode.Data, newNode.Data, path)
			ops = append(ops, textOps...)
		}
	}

	// 4. Compare Children
	childOps, err := diffChildren(oldNode, newNode, path)
	if err != nil {
		return nil, err
	}
	ops = append(ops, childOps...)

	return ops, nil
}

func diffAttributes(oldNode, newNode *html.Node, path NodePath) []Operation {
	var ops []Operation
	oldAttrs := make(map[string]string)
	for _, a := range oldNode.Attr {
		oldAttrs[a.Key] = a.Val
	}

	newAttrs := make(map[string]string)
	for _, a := range newNode.Attr {
		newAttrs[a.Key] = a.Val
	}

	// Check for updates or deletions
	for k, vOld := range oldAttrs {
		vNew, exists := newAttrs[k]
		if !exists {
			// Attribute deleted (or set to empty if we handle it that way, but explicit delete is better)
		} else if vOld != vNew {
			ops = append(ops, Operation{
				Type:     OpUpdateAttr,
				Path:     path,
				Key:      k,
				OldValue: vOld,
				NewValue: vNew,
			})
		}
	}

	// Check for additions
	for k, vNew := range newAttrs {
		if _, exists := oldAttrs[k]; !exists {
			ops = append(ops, Operation{
				Type:     OpUpdateAttr,
				Path:     path,
				Key:      k,
				NewValue: vNew,
			})
		}
	}

	return ops
}

// diffChildren compares lists of children.
func diffChildren(oldNode, newNode *html.Node, parentPath NodePath) ([]Operation, error) {
	var ops []Operation

	oldChildren := getChildrenList(oldNode)
	newChildren := getChildrenList(newNode)

	// Simple loop over matching indices
	commonLen := len(oldChildren)
	if len(newChildren) < commonLen {
		commonLen = len(newChildren)
	}

	for i := 0; i < commonLen; i++ {
		// New Path for this child
		childPath := append(NodePath(nil), parentPath...)
		childPath = append(childPath, i)

		// Recursively diff
		childOps, err := diffNodes(oldChildren[i], newChildren[i], childPath)
		if err != nil {
			return nil, err
		}
		ops = append(ops, childOps...)
	}

	// Handle Deletions (Old has more)
	for i := len(oldChildren) - 1; i >= commonLen; i-- {
		ops = append(ops, Operation{
			Type: OpDeleteNode,
			Path: append(append(NodePath(nil), parentPath...), i),
		})
	}

	// Handle Insertions (New has more)
	for i := commonLen; i < len(newChildren); i++ {
		nodeHTML, err := RenderNode(newChildren[i])
		if err != nil {
			return nil, err
		}
		ops = append(ops, Operation{
			Type:     OpInsertNode,
			Path:     parentPath,
			Position: i,
			NodeData: nodeHTML,
		})
	}

	return ops, nil
}

func getChildrenList(n *html.Node) []*html.Node {
	var children []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		children = append(children, c)
	}
	return children
}

func diffText(oldText, newText string, path NodePath) []Operation {
	// Find common prefix length
	prefixLen := 0
	minLen := len(oldText)
	if len(newText) < minLen {
		minLen = len(newText)
	}
	for prefixLen < minLen && oldText[prefixLen] == newText[prefixLen] {
		prefixLen++
	}

	// Find common suffix length, constrained by prefixLen
	suffixLen := 0
	maxSuffix := minLen - prefixLen
	for suffixLen < maxSuffix {
		if oldText[len(oldText)-1-suffixLen] == newText[len(newText)-1-suffixLen] {
			suffixLen++
		} else {
			break
		}
	}

	var ops []Operation

	// Middle part of oldText is deleted
	deleteCount := len(oldText) - prefixLen - suffixLen
	if deleteCount > 0 {
		deletedText := oldText[prefixLen : len(oldText)-suffixLen]
		ops = append(ops, Operation{
			Type:     OpDeleteText,
			Path:     path,
			Position: prefixLen,
			OldValue: deletedText,
		})
	}

	// Middle part of newText is inserted
	insertCount := len(newText) - prefixLen - suffixLen
	if insertCount > 0 {
		insertedText := newText[prefixLen : len(newText)-suffixLen]
		ops = append(ops, Operation{
			Type:     OpInsertText,
			Path:     path,
			Position: prefixLen,
			NewValue: insertedText,
		})
	}

	return ops
}
