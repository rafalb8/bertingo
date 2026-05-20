// https://www.erlang.org/doc/apps/erts/erl_ext_dist.html
package bert

//go:generate go tool stringer -type=Tag
type Tag byte

// Header tags
const (
	Version        Tag = 131
	CompressedZlib Tag = 80
)

const (
	AtomCacheRef Tag = 82

	SmallIntegerExt Tag = 97
	IntegerExt      Tag = 98

	FloatExt Tag = 99

	PortExt    Tag = 102
	NewPortExt Tag = 89
	V4PortExt  Tag = 120

	PidExt    Tag = 103
	NewPidExt Tag = 88

	SmallTupleExt Tag = 104
	LargeTupleExt Tag = 105

	MapExt Tag = 116

	NilExt Tag = 106

	StringExt Tag = 107
	ListExt   Tag = 108

	BinaryExt Tag = 109

	SmallBigExt Tag = 110
	LargeBigExt Tag = 111

	ReferenceExt      Tag = 101
	NewReferenceExt   Tag = 114
	NewerReferenceExt Tag = 90

	FunExt    Tag = 117
	NewFunExt Tag = 112

	ExportExt    Tag = 113
	BitBinaryExt Tag = 77
	NewFloatExt  Tag = 70

	AtomUtf8Ext      Tag = 118
	SmallAtomUtf8Ext Tag = 119
	AtomExt          Tag = 100
	SmallAtomExt     Tag = 115

	LocalExt Tag = 121
)
