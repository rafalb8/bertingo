# BERTinGo

Go library for encoding data into the BERT (Binary Erlang Term) format.
It allows your Go applications to speak directly to Erlang and Elixir nodes.

## Features
* **Struct Tag Customization**: Easily map Go structs to Erlang Tuples, Lists, Atoms, or Binaries using the `bert` tag.
* **Custom Serialization**: Implement the `ToTermer` interface on your own types for full control over how they convert to BERT terms.
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
	data, err := bert.Marshal(m, true) // true - add BERT2 length prefix
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

### 3. Decoding to Terms
```go
package main

import (
	"bytes"
	"fmt"

	bert "github.com/rafalb8/bertingo"
)

func main() {
	// Example BERT file
	buf := bytes.NewBuffer([]byte{
		131, 104, 3, 100, 0, 3, 98, 111,
		98, 109, 0, 0, 0, 5, 97, 100,
		109, 105, 110, 97, 25,
	})

	term, err := bert.NewDecoder(buf).Decode()
	if err != nil {
		panic(err)
	}

	// Outputs:
	// Tuple: {
	//   Atom: bob
	//   Binary: <<"admin">>
	//   SmallInteger: 25
	// }
	fmt.Println(bert.Tree(term))
}
```

## Advanced Usage
### Custom Term Conversion (`ToTermer`)
You can intercept the conversion pipeline for any custom Go type by implementing the `ToTermer` interface. If a type satisfies this interface, `bert.ToTerm()` will bypass reflection and defer directly to your custom implementation:

```go
type ToTermer interface {
    ToTerm() (Term, error)
}
```

#### Example:
```go
package main

import (
	bert "github.com/rafalb8/bertingo"
)

type CustomSecret string

// ToTerm ensures the secret is always obfuscated or tagged safely when encoded
func (s CustomSecret) ToTerm() (bert.Term, error) {
	return bert.ToTerm(map[string]string{
		"status": "encrypted",
		"data":   "******",
	})
}
```

---

## How Types Map Together
| Go Type | Erlang Type | Notes |
| --- | --- | --- |
| Type matching `ToTermer` | *Variable* | Uses custom `ToTerm()` output directly |
| bool | Atom | Encodes as true or false tokens |
| []byte | Binary | Raw data chunks |
| string | String | Lists of characters |
| uint8 | SmallInteger | Fast 8-bit values (0-255) |
| uint16/int8/int16/int32 | Integer | Signed 32-bit values |
| uint32/uint64/int64 | SmallBigInt | Large precision integers that fit into small bignum tags |
| uint/int | SmallInteger/Integer/SmallBigInt | Dynamic mapping based on the host architecture size and runtime value |
| float32/float64 | NewFloat | 8-byte IEEE 754 precision |
| map | Map | Native associative arrays with key-value pairs |
| struct | Tuple | Key-value pairs matching your fields (or tagged tuples for records) |

---

## Struct Tags Reference
Use the `bert:"..."` key to change how fields behave:

* `bert:""` — Drops the Erlang key entirely.
* `bert:"-"` — Tells the encoder to always ignore this field.
* `bert:"name"` — Changes the Erlang key name to "name" instead of the Go struct field name.
* `bert:"name,atom"` — Forces a string to become an Erlang Atom token.
* `bert:"name,binary"` — Forces a string to become a raw Erlang Binary object.
* `bert:"name,omitempty"` — Skips writing this field entirely if it holds its default Go zero value.
* `bert:"name,omitzero"` — Skips writing this field entirely if it returns `true` for `IsZero() bool` or holds its default Go zero value.
