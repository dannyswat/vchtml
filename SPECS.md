# vchtml Specification

## Overview
`vchtml` is a Go library designed to provide version control semantics specifically for HTML documents. Unlike line-based diff tools (like standard `diff` or `git`), `vchtml` understands the hierarchical structure of the Document Object Model (DOM). This allows for semantic diffing, patching, and merging, which is essential for collaborative editing workflows where changes need to be reviewed and approved.

## Goals
1.  **Structure Awareness**: Operations should respect HTML tags, attributes, and nesting.
2.  **Minimality**: Deltas should represent the smallest set of changes required.
3.  **Concurrency Support**: Ability to handle multiple users editing the same base document.
4.  **Conflict Safety**: Detect and report when concurrent edits collide logically (e.g., two users changing the same attribute to different values).

## Architecture

The library relies on `golang.org/x/net/html` for parsing and rendering.

### 1. Data Structures

#### 1.1. Pathing
To reliably address specific nodes within the HTML tree, we utilize a pathing mechanism (similar to a simplified XPath or a slice of child indices).

```go
// NodePath represents the traversal steps from the root to a target node.
// Example: [0, 1, 3] means root -> child[0] -> child[1] -> child[3]
type NodePath []int
```

#### 1.2. Operations
A `Delta` consists of a sequence of atomic `Operation`s.

```go
type OpType string

const (
    OpInsertNode     OpType = "INSERT_NODE"     // Insert a new node
    OpDeleteNode     OpType = "DELETE_NODE"     // Remove a node
    OpMoveNode       OpType = "MOVE_NODE"       // Reparent or reorder a node
    OpUpdateAttr     OpType = "UPDATE_ATTR"     // Change/Add/Remove an attribute
    OpUpdateText     OpType = "UPDATE_TEXT"     // Change text content of a text node
)

type Operation struct {
    Type      OpType
    Path      NodePath    // Location where the operation applies
    Key       string      // For Attributes (name of the attribute)
    OldValue  string      // Previous value (for verification/conflict check)
    NewValue  string      // New value/Content
    NodeData  string      // For Insert: The HTML string of the node
    Position  int         // For Insert/Move: The index in the parent's child list
}
```

#### 1.3. Delta
```go
type Delta struct {
    BaseHash   string      // Hash of the original document to ensure validity
    Operations []Operation
    Timestamp  int64
    Author     string
}
```

### 2. Core API

#### 2.1. Diff
Compares two HTML strings and produces a Delta.

```go
// Diff calculates the operations needed to transform 'oldHTML' into 'newHTML'.
func Diff(oldHTML, newHTML string) (*Delta, error)
```
*   **Algorithm**: Tree-edit distance algorithm (e.g., Zhang-Shasha or a heuristic-based approach optimized for HTML).
*   **Heuristics**: Match nodes by ID if available, otherwise use tag name + class/content similarity.

#### 2.2. Patch (Apply)
Applies a Delta to a base HTML string.

```go
// Patch applies the changes in 'delta' to 'baseHTML'.
// Returns error if BaseHash doesn't match or paths are invalid.
func Patch(baseHTML string, delta *Delta) (string, error)
```

#### 2.3. Merge
The core complexity handler. It takes a base document and two concurrent deltas (Alice's and Bob's) and attempts to combine them.

```go
type ResolutionStrategy int

const (
    StrategyFail OnConflict // Return error on conflict
    StrategyKeepYours       // Prefer delta A
    StrategyKeepTheirs      // Prefer delta B
)

// Merge combines two concurrent deltas based on a common ancestor.
// It returns a new merged HTML string, a consolidated Delta, and a list of conflicts if any.
func Merge(baseHTML string, deltaA, deltaB *Delta) (string, *Delta, []Conflict, error)
```

### 3. Conflict Detection logic

Conflicts arise when `deltaA` and `deltaB` modify the same or dependent parts of the tree. Use 3-way merge logic.

**Scenario 1: Attribute Conflict**
*   Base: `<div class="a">`
*   Delta A: Update `class` to `"a b"`
*   Delta B: Update `class` to `"a c"`
*   **Result**: Conflict. Manual resolution needed (or specific heuristics like tokenizing class lists).

**Scenario 2: Text Conflict**
*   Base: `<p>Hello World</p>`
*   Delta A: `<p>Hello Go</p>`
*   Delta B: `<p>Hello Future</p>`
*   **Result**: Conflict.

**Scenario 3: Structural Conflict (Delete/Edit)**
*   Base: `<ul><li>Item 1</li></ul>`
*   Delta A: Edits text of `<li>` to "Item 1 Modified"
*   Delta B: Deletes the `<ul>` (and thus the `<li>`)
*   **Result**: Conflict (Ghost Edit). Usually, deletion wins, or it is flagged.

**Scenario 4: Sibling Insertions (Non-conflicting)**
*   Base: `<ul><li>A</li></ul>`
*   Delta A: Inserts `<li>B</li>` at index 1.
*   Delta B: Inserts `<li>C</li>` at index 1.
*   **Result**: Both inserted. Order depends on determinism rule (e.g., timestamp or hash).

## Implementation Roadmap

1.  **Phase 1: DOM Traversal & Pathing**
    *   Implement parser wrapper using `golang.org/x/net/html`.
    *   Implement `GetNode(root, path)` and `GetPath(root, node)`.

2.  **Phase 2: Diff Engine**
    *   Implement basic tree comparison.
    *   Generate `INSERT`, `DELETE`, `UPDATE` operations.

3.  **Phase 3: Patch Engine**
    *   Apply operations to the DOM tree.
    *   Render back to string.

4.  **Phase 4: Merge & Conflict**
    *   Implement conflict detection logic.
    *   Implement the `Merge` function.

## Dependencies

*   `golang.org/x/net/html`: robust, standards-compliant HTML parsing.

