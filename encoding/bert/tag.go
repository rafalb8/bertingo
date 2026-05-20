// https://www.erlang.org/doc/apps/erts/erl_ext_dist.html
package bert

//go:generate go tool stringer -type=Tag

// Tag represents a single byte marker that identifies the data type in Erlang binary format.
type Tag byte

// Format header versions
const (
	Version        Tag = 131 // The standard Erlang binary format version marker
	CompressedZlib Tag = 80  // Signals that the data inside is compressed with Zlib
)

// Erlang data type tags
const (
	AtomCacheRef Tag = 82 // Reference to an atom stored in the connection cache

	SmallIntegerExt Tag = 97 // Unsigned 8-bit integer (0 to 255)
	IntegerExt      Tag = 98 // Signed 32-bit integer

	FloatExt Tag = 99 // Old 31-byte string representation of a float

	PortExt    Tag = 102 // Old Erlang port identification marker
	NewPortExt Tag = 89  // Standard Erlang port identification marker
	V4PortExt  Tag = 120 // Newest Erlang port format (supports large node names)

	PidExt    Tag = 103 // Old Erlang process identifier format
	NewPidExt Tag = 88  // Standard Erlang process identifier format

	SmallTupleExt Tag = 104 // A tuple containing 255 or fewer elements
	LargeTupleExt Tag = 105 // A tuple containing more than 255 elements

	MapExt Tag = 116 // Erlang map type (key-value pairs)

	NilExt Tag = 106 // An empty list

	StringExt Tag = 107 // A list of bytes (characters) up to 65,535 bytes long
	ListExt   Tag = 108 // A standard Erlang list with elements and a tail

	BinaryExt Tag = 109 // A raw chunk of binary byte data

	SmallBigExt Tag = 110 // Large integer (bignum) up to 255 bytes long
	LargeBigExt Tag = 111 // Huge integer (bignum) up to 4 gigabytes long

	ReferenceExt      Tag = 101 // Old Erlang internal reference format
	NewReferenceExt   Tag = 114 // Standard Erlang reference format
	NewerReferenceExt Tag = 90  // Modern reference format with larger data sizes

	FunExt    Tag = 117 // Old Erlang anonymous function (lambda) format
	NewFunExt Tag = 112 // Standard Erlang anonymous function (lambda) format

	ExportExt    Tag = 113 // A remote function reference (Module:Function/Arity)
	BitBinaryExt Tag = 77  // A binary chunk whose length is not a multiple of 8 bits
	NewFloatExt  Tag = 70  // Modern 8-byte IEEE 754 float format

	AtomUtf8Ext      Tag = 118 // An atom name encoded in UTF-8 (up to 65,535 bytes)
	SmallAtomUtf8Ext Tag = 119 // An atom name encoded in UTF-8 (up to 255 bytes)
	AtomExt          Tag = 100 // Old Latin-1 encoded atom name (up to 65,535 bytes)
	SmallAtomExt     Tag = 115 // Old Latin-1 encoded atom name (up to 255 bytes)

	LocalExt Tag = 121 // Internal engine optimization token marker
)
