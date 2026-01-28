package vchtml

// NodePath represents the traversal steps from the root to a target node.
// Example: [0, 1, 3] means root -> child[0] -> child[1] -> child[3]
type NodePath []int

type OpType string

const (
	OpInsertNode OpType = "INSERT_NODE" // Insert a new node
	OpDeleteNode OpType = "DELETE_NODE" // Remove a node
	OpMoveNode   OpType = "MOVE_NODE"   // Reparent or reorder a node
	OpUpdateAttr OpType = "UPDATE_ATTR" // Change/Add/Remove an attribute
	OpUpdateText OpType = "UPDATE_TEXT" // Replace full text (Atomic)
	OpInsertText OpType = "INSERT_TEXT" // Insert text at position
	OpDeleteText OpType = "DELETE_TEXT" // Delete text at position
)

// Operation represents an atomic change to the HTML structure.
type Operation struct {
	Type     OpType   `json:"type"`
	Path     NodePath `json:"path"`
	Key      string   `json:"key,omitempty"`       // For Attributes (name of the attribute)
	OldValue string   `json:"old_value,omitempty"` // Previous value (for verification/conflict check)
	NewValue string   `json:"new_value,omitempty"` // New value/Content. For InsertText: text to insert.
	NodeData string   `json:"node_data,omitempty"` // For Insert: The HTML string of the node
	Position int      `json:"position,omitempty"`  // For InsertNode/MoveNode: child index. For InsertText/DeleteText: char offset.
}

// Delta represents a set of changes applied to a base document.
type Delta struct {
	BaseHash   string      `json:"base_hash"` // Hash of the original document to ensure validity
	Operations []Operation `json:"operations"`
	Timestamp  int64       `json:"timestamp"`
	Author     string      `json:"author"`
}

// Conflict represents a detected conflict between two operations.
type Conflict struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Path        NodePath    `json:"path"`
	Ops         []Operation `json:"ops"`
}
