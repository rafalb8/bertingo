# BERTinGo

Go library for encoding data into the BERT (Binary Erlang Term) format.
It allows your Go applications to speak directly to Erlang and Elixir nodes.

## Features
* **Struct Tag Customization**: Easily map Go structs to Erlang Tuples, Lists, Atoms, or Binaries using the `bert` tag.
* **BERT2 Support**: Includes optional support for BERT2 length-prefixed packet framing.

## Installation
```bash
go get -u github.com/rafalb8/bertingo
```

## Quick Start

### 1. Simple Marshalling (To Bytes)
If you just need a raw `[]byte` slice to send over a network socket or write to a file, use `Marshal`:

```go
package main

import (
	"encoding/hex"
	"fmt"

	bert "github.com/rafalb8/bertingo"
)

func main() {
	m := map[string]any{
		"name": "Alice",
		"age":  30,
	}

	// Turn it into BERT binary format
	data, err := bert.Marshal(m)
	if err != nil {
		panic(err)
	}

	fmt.Println(hex.Dump(data))
}
```

### 2. Stream Encoding
```go
package main

import (
	"bytes"
	"encoding/hex"
	"fmt"

	bert "github.com/rafalb8/bertingo"
)

type User struct {
	Name string `bert:"name,atom"`   // Encodes as an Erlang Atom symbol
	Role string `bert:"role,binary"` // Encodes as a raw Erlang Binary block
	Age  int    `bert:"age"`
}

func main() {
	// Prepare our stream destination (like a network connection or file)
	buf := bytes.Buffer{}
	encoder := bert.NewEncoder(&buf)

	user := User{
		Name: "bob",
		Role: "admin",
		Age:  25,
	}

	// Convert Go struct to generic BERT terms
	term, err := bert.ToTerm(user)
	if err != nil {
		panic(err)
	}

	// Serialize straight to the output stream
	err = encoder.Encode(term)
	if err != nil {
		panic(err)
	}

	fmt.Println(hex.Dump(buf.Bytes()))
}
```

## How Types Map Together
| Go Type | Erlang Type | Notes |
| --- | --- | --- |
| `bool` | `Atom` | Encodes as `true` or `false` tokens |
| `string` | `String` | Lists of characters |
| `int32` | `Integer` | Signed 32-bit values |
| `uint8` | `SmallInteger` | Fast 8-bit values (0-255) |
| `float64` | `NewFloat` | 8-byte IEEE 754 precision |
| `[]byte` | `Binary` | Raw data chunks |
| `struct` | `Tuple` | Key-value pairs matching your fields |

## Struct Tags Reference
Use the `bert:"..."` key to change how fields behave:

* `bert:"name"` — Changes the Erlang key name to "name" instead of the Go struct field name.
* `bert:"name,atom"` — Forces a string to become an Erlang Atom token.
* `bert:"name,binary"` — Forces a string to become a raw Erlang Binary object.
* `bert:"name,omitempty"` — Skips writing this field entirely if it holds its default Go zero value.
* `bert:"-"` — Tells the encoder to always ignore this field.
