# vchtml

`vchtml` is a structure-aware version control library for HTML, written in Go. Unlike traditional line-based diff tools, `vchtml` understands the Document Object Model (DOM), enabling semantic diffing, patching, and 3-way merging of HTML documents.

It is designed for collaborative editing scenarios where multiple users might be modifying the same HTML structure concurrently.

## Features

- **DOM-Aware Diffing**: Generates a minimal set of atomic operations (`Delta`) to transform one HTML tree into another.
- **Accurate Patching**: Applies changes safely to a base document, verifying integrity with hashes.
- **3-Way Merge**: Intelligent merging of concurrent edits from two users against a common base.
- **Conflict Detection**: Automatically identifies logical conflicts (e.g., conflicting attribute updates, overlapping text edits).
- **Granular Operations**: Supports node insertion/deletion, attribute updates, and fine-grained text editing.

## Installation

```bash
go get github.com/dannyswat/vchtml
```

## Usage

### Basic Diff and Patch

```go
package main

import (
    "fmt"
    "log"
    "github.com/dannyswat/vchtml"
)

func main() {
    original := `<div><p>Hello World</p></div>`
    modified := `<div><p>Hello Go!</p></div>`
    
    // Calculate the difference
    delta, err := vchtml.Diff(original, modified, "alice")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Generated %d operations\n", len(delta.Operations))
    
    // Apply the delta back to the original
    result, err := vchtml.Patch(original, delta)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(result)
    // Output: <html><head></head><body><div><p>Hello Go!</p></div></body></html>
}
```

### Merging Concurrent Changes

```go
package main

import (
    "fmt"
    "github.com/dannyswat/vchtml"
)

func main() {
    base := `<div><p>Hello</p></div>`
    
    // User A changes text
    deltaA, _ := vchtml.Diff(base, `<div><p>Hi</p></div>`, "Alice")
    
    // User B adds a class attribute
    deltaB, _ := vchtml.Diff(base, `<div class="greeting"><p>Hello</p></div>`, "Bob")
    
    // Merge both changes onto base
    mergedHTML, _, conflicts, err := vchtml.Merge(base, deltaA, deltaB)
    if err != nil {
        panic(err)
    }
    
    if len(conflicts) > 0 {
        fmt.Println("Conflicts detected:", conflicts)
    } else {
        fmt.Println("Merged Result:")
        fmt.Println(mergedHTML)
    }
}
```

## Core API

### `Diff(oldHTML, newHTML, author string) (*Delta, error)`
Compares two HTML strings and returns a `Delta` containing the sequence of operations required to transform `oldHTML` to `newHTML`.

### `Patch(baseHTML string, delta *Delta) (string, error)`
Applies a `Delta` to a base HTML string. It validates the base document hash before applying changes to ensure consistency.

### `Merge(baseHTML string, deltaA, deltaB *Delta) (string, *Delta, []Conflict, error)`
Combines two concurrent deltas (`deltaA` and `deltaB`) that both originated from `baseHTML`. It returns:
- The merged HTML string.
- A consolidated `Delta` representing the combined changes.
- A list of `Conflict`s if the changes are incompatible.

## Operations

The library uses a set of atomic operations to represent changes:

- `INSERT_NODE`: Adds a new HTML element.
- `DELETE_NODE`: Removes an existing element.
- `MOVE_NODE`: Reparents or reorders a node.
- `UPDATE_ATTR`: Adds, removes, or modifies an attribute.
- `UPDATE_TEXT`: Replaces the entire content of a text node.
- `INSERT_TEXT`: Inserts a string into a text node at a specific offset.
- `DELETE_TEXT`: Removes a string from a text node at a specific offset.

## Testing

Run the test suite:

```bash
go test -v
```

## License

MIT
