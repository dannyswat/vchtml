# vchtml

`vchtml` is a Go library for HTML-specific version control. It supports semantic diffing, patching, and 3-way merging of HTML documents.

## Features

- **Diff**: Generates a minimal set of operations (`Delta`) to transform one HTML document into another.
- **Patch**: Applies a `Delta` to a base HTML document.
- **Merge**: Combines concurrent changes from two users, handling index shifts and detecting conflicts.
- **Conflict Detection**: Identifies colliding edits (e.g., same attribute modified differently).

## Usage

```go
package main

import (
    "fmt"
    "vchtml"
)

func main() {
    base := `<div><p>Hello</p></div>`
    
    // User A changes text
    deltaA, _ := vchtml.Diff(base, `<div><p>Hi</p></div>`, "Alice")
    
    // User B adds an attribute
    deltaB, _ := vchtml.Diff(base, `<div class="main"><p>Hello</p></div>`, "Bob")
    
    // Merge
    merged, _, conflicts, _ := vchtml.Merge(base, deltaA, deltaB)
    
    if len(conflicts) > 0 {
        fmt.Println("Conflicts detected!")
    } else {
        fmt.Println("Merged HTML:", merged)
        // Output: <div class="main"><p>Hi</p></div>
    }
}
```

## Operations

The library uses the following atomic operations:
- `INSERT_NODE`
- `DELETE_NODE`
- `UPDATE_ATTR`
- `UPDATE_TEXT`

## Testing

Run the test suite:

```bash
go test -v
```
