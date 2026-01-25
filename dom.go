package vchtml

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// ParseHTML parses a string into an HTML node tree.
func ParseHTML(content string) (*html.Node, error) {
	// ParseFragment allows us to parse parts of HTML without enforcing <html><body> structure
	// if the input is partial, but for a full doc, Parse is better.
	// Let's assume we are dealing with a full document context or fragment.
	// For general purpose, Parse is safer as it normalizes the tree.
	// However, Parse wraps everything effectively in html/head/body.
	// Let's try to detect if it's a fragment or doc.
	// For simplicity in v1, we use Parse.
	return html.Parse(strings.NewReader(content))
}

// RenderNode converts a node tree back to a string.
func RenderNode(n *html.Node) (string, error) {
	var buf bytes.Buffer
	if err := html.Render(&buf, n); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// GetNode traverses the tree using the provided path to find a specific node.
// The path indices generally refer to element/text nodes in the Child traversal.
func GetNode(root *html.Node, path NodePath) (*html.Node, error) {
	current := root
	for i, index := range path {
		// Find the child at 'index'
		child := getChildAtIndex(current, index)
		if child == nil {
			return nil, fmt.Errorf("node not found at path %v (failed at index %d, step %d)", path, index, i)
		}
		current = child
	}
	return current, nil
}

// getChildAtIndex finds the Nth child of a node.
// Note: html.Node's children are a linked list (FirstChild, NextSibling).
func getChildAtIndex(parent *html.Node, index int) *html.Node {
	count := 0
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if count == index {
			return c
		}
		count++
	}
	return nil
}

// GetPath finds the path from root to the target node.
func GetPath(root, target *html.Node) (NodePath, error) {
	var path NodePath

	// We build the path backwards from target to root
	current := target
	for current != root {
		parent := current.Parent
		if parent == nil {
			return nil, errors.New("target node is not a descendant of root")
		}

		index := getChildIndex(parent, current)
		if index == -1 {
			return nil, errors.New("integrity error: child not found in parent's list")
		}

		// Prepend index
		path = append(NodePath{index}, path...)
		current = parent
	}
	return path, nil
}

// getChildIndex returns the index of child within parent.
func getChildIndex(parent, child *html.Node) int {
	count := 0
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if c == child {
			return count
		}
		count++
	}
	return -1
}
