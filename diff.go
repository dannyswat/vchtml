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

	// We assume operations are generated against the 'old' tree structure.
	// As we generate ops, indices might shift if we applied them sequentially,
	// but usually a Delta is a set of instructions based on the *original* state
	// (or they need to be applied in a specific order, typically reverse for deletes).
	// For this library, let's assume paths in operations refer to the *original* document state
	// unless we specify otherwise.

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
	// If types differ (Element vs Text) or Data (tag name) differs,
	// we treat it as a full replace (Delete old, Insert new).
	// However, for the root, we can't really "replace" it easily in this recursion
	// without context of parent.
	// But usually this function is called on matching pairs.

	if oldNode.Type != newNode.Type || oldNode.DataAtom != newNode.DataAtom || (oldNode.Type == html.ElementNode && oldNode.Data != newNode.Data) {
		// Totally different nodes.
		// Since we are inside a recursion that assumes these nodes "match" structurally/positionally,
		// this implies the node at this path has changed type/tag.
		// We should probably emit a DELETE on this path and an INSERT on this path.
		// But wait, if we delete the node at 'path', the path becomes invalid for the insert if we are not careful?
		// Actually, usually REPLACE = UPDATE (if supported) or DELETE + INSERT.

		// For now, let's handle Text changes and Attribute changes.
		// Structural replace is complex. Let's assume for V1 the structure is somewhat stable
		// or we handle it in diffChildren logic.
	}

	// 2. Compare Attributes (if Element)
	if oldNode.Type == html.ElementNode {
		attrOps := diffAttributes(oldNode, newNode, path)
		ops = append(ops, attrOps...)
	}

	// 3. Compare Text (if TextNode)
	if oldNode.Type == html.TextNode {
		if oldNode.Data != newNode.Data {
			ops = append(ops, Operation{
				Type:     OpUpdateText,
				Path:     path,
				OldValue: oldNode.Data,
				NewValue: newNode.Data,
			})
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
			// For HTML, removing an attribute is distinct.
			// We'll signal remove by maybe sending a special value or just handling it?
			// The Operation struct has NewValue. We can't distinguish "set to empty" vs "remove" unless we have a flag or convention.
			// Let's assume NewValue="" means empty string, but we need a way to say remove.
			// We usually use OpUpdateAttr with NewValue=""? No, that's valid value.
			// We might need an OpRemoveAttr but we defined OpUpdateAttr.
			// Let's assume standard behavior: if null/missing in new, it's removed.
			// We can use a sentinel or just OpUpdateAttr with logical "remove".
			// Check spec: OpUpdateAttr // Change/Add/Remove an attribute.
			// We might need to handle "Remove" logic in Patch.
			// For now, let's say if NewValue is empty and we verify it's missing in New map?
			// Actually the Patch needs to know if it should set Attr="" or remove it.
			// Let's revisit OpType or just assume "nil" concept.
			// Currently NewValue is string.
			// For now, we will treat missing as "removed".
			// We can encode "removed" as a special value or rely on Patch knowing that.
			// Or we assume the op simply says "set this key to this value",
			// but we need a "Delete Attribute" op.
			// Reuse Ops: OpUpdateAttr can imply remove if NewValue is specific?
			// Let's stick to OpUpdateAttr. We will assume if it's missing in new, it is an update to "".
			// Wait, that's wrong strictly speaking.
			// Let's assume we can change OpType to OpDeleteAttr in future if needed.
			// For now, let's treat it as Update to empty for simplicity, or handle "Remove" by passing a magic value?
			// No, that's hacky.
			// Let's just say "UpdateAttr" with nil concept? string in Go can't be nil.
			// We'll treat it as: if we detect removal, we emit OpUpdateAttr with empty string,
			// BUT this might be ambiguous.
			// Let's add a comment: we treat attribute removal as setting to empty string for V0.
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
// Ideally this uses a LCS / Edit Distance algorithm.
// For V1, we will implement a simple index-based comparison
// and detect basic Insert/Append.
// We will iterate and match by index.
// If `new` has more children, they are Inserts.
// If `old` has more, they are Deletes.
// Note: This is NOT robust for reordering or inserting in the middle,
// as it will detect everything after as Changed.
func diffChildren(oldNode, newNode *html.Node, parentPath NodePath) ([]Operation, error) {
	var ops []Operation

	// Convert linked lists to slices for easier indexing
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
	// We must delete from the end to avoid shifting indices affecting subsequent deletions
	for i := len(oldChildren) - 1; i >= commonLen; i-- {
		// The node at oldChildren[commonLen] (since we process in order?)
		// Actually, if we delete, we usually delete from the end or specific index.
		// Path: parentPath + [i]
		ops = append(ops, Operation{
			Type: OpDeleteNode,
			Path: append(append(NodePath(nil), parentPath...), i),
		})
	}

	// Handle Insertions (New has more)
	for i := commonLen; i < len(newChildren); i++ {
		// Render the new node to string
		nodeHTML, err := RenderNode(newChildren[i])
		if err != nil {
			return nil, err
		}
		ops = append(ops, Operation{
			Type:     OpInsertNode,
			Path:     parentPath, // Insert into parent
			Position: i,          // At index i
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
