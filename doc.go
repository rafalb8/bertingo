/*
Package bert provides encoding and decoding for the Binary Erlang Term format.

# Type Mapping

The package translates Go data types to Erlang external term formats via runtime
reflection using [ToTerm]. The following mappings are applied:

  - bool                     -> Atom ("true" or "false")
  - string                   -> StringExt (or optimized to Binary/Atom via tags)
  - uint8, int8, int16, int32-> SmallIntegerExt or IntegerExt
  - float32, float64         -> NewFloatExt (IEEE 754 double precision)
  - slices/arrays of uint8   -> BinaryExt
  - slices/arrays of other   -> ListExt (optimized to StringExt if all elements are bytes)
  - structs                  -> TupleExt (or ListExt of paired tuples via tags)
  - nil pointers/interfaces  -> TupleExt representation: {bert, nil}

# Struct Tags

You can control how Go struct fields are translated into Erlang terms using the
`bert` struct tag.

	type User struct {
		ID    int32    `bert:"id"`
		Name  string   `bert:"name,atom"`       // Encoded as an Erlang Atom
		Bio   string   `bert:"bio,binary"`      // Encoded as a raw Erlang Binary
		Roles []string `bert:"roles,omitempty"` // Skipped if empty
	}

Options available:
  - omitempty : Skips the field if it holds its Go zero-value.
  - atom      : Forces a Go string to encode as an Erlang Atom token.
  - binary    : Forces a Go string to encode as a raw Erlang Binary chunk.
  - list      : Forces a nested struct to serialize as a sequential List.
*/
package bert
