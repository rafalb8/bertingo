package bert

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"slices"
	"strconv"
)

// Term represents any valid Binary Erlang Term data type.
type Term interface {
	// Append serializes the term data and adds it to the destination byte slice.
	Append(dst []byte) []byte
	// String returns a clean text layout version of the inner value.
	String() string
}

// SmallInteger represents an unsigned 8-bit integer (range 0 to 255).
type SmallInteger uint8

func (i SmallInteger) Append(dst []byte) []byte {
	return append(dst, byte(SmallIntegerExt), byte(i))
}

func (i SmallInteger) String() string {
	return strconv.FormatUint(uint64(i), 10)
}

// Integer represents a signed 32-bit integer.
type Integer int32

func (i Integer) Append(dst []byte) []byte {
	dst = append(dst, byte(IntegerExt))
	return binary.BigEndian.AppendUint32(dst, uint32(i))
}

func (i Integer) String() string {
	return strconv.FormatInt(int64(i), 10)
}

// Float represents an old-style legacy float stored as a 31-byte text string.
type Float float64

func (f Float) Append(dst []byte) []byte {
	dst = append(dst, byte(FloatExt))
	return fmt.Appendf(dst, "%.20e", float64(f))
}

func (f Float) String() string {
	return strconv.FormatFloat(float64(f), 'g', 10, 64)
}

// Tuple represents a collection of terms with a fixed layout size.
type Tuple []Term

func (t Tuple) Append(dst []byte) []byte {
	switch {
	case len(t) <= math.MaxUint8:
		dst = append(dst, byte(SmallTupleExt), byte(len(t)))
	case len(t) <= math.MaxUint32:
		dst = append(dst, byte(LargeTupleExt))
		dst = binary.BigEndian.AppendUint32(dst, uint32(len(t)))
	default:
		return dst
	}

	for _, a := range t {
		dst = a.Append(dst)
	}
	return dst
}

func (t Tuple) String() string {
	return fmt.Sprint([]Term(t))
}

// Map represents Erlang key-value pair maps.
// Elements are stored sequentially in a flat list: [Key1, Val1, Key2, Val2].
type Map []Term

func (m Map) Append(dst []byte) []byte {
	if len(m)/2 > math.MaxUint32 {
		return dst
	}
	if len(m)%2 != 0 { // map requires matching pairs
		return dst
	}

	dst = append(dst, byte(MapExt))
	dst = binary.BigEndian.AppendUint32(dst, uint32(len(m)/2))

	for _, a := range m {
		dst = a.Append(dst)
	}
	return dst
}

func (m Map) String() string {
	return fmt.Sprint([]Term(m))
}

// Nil represents an empty Erlang list container structure.
type Nil struct{}

func (Nil) Append(dst []byte) []byte {
	return append(dst, byte(NilExt))
}

func (Nil) String() string {
	return "[]"
}

// String represents a flat list of small characters or bytes up to 65,535 bytes.
type String string

func (s String) Append(dst []byte) []byte {
	if len(s) == 0 {
		return Nil{}.Append(dst)
	}

	if len(s) > math.MaxUint16 {
		return dst
	}

	dst = append(dst, byte(StringExt))
	dst = binary.BigEndian.AppendUint16(dst, uint16(len(s)))
	return append(dst, s...)
}

func (s String) String() string {
	return strconv.QuoteToGraphic(string(s))
}

// List represents an Erlang list container structure.
type List []Term

func (l List) Append(dst []byte) []byte {
	if len(l) == 0 {
		return Nil{}.Append(dst)
	}

	length := len(l)
	if length > math.MaxUint32 {
		return dst
	}

	dst = append(dst, byte(ListExt))
	dst = binary.BigEndian.AppendUint32(dst, uint32(length))

	for _, a := range l {
		dst = a.Append(dst)
	}

	// standard list must be closed with a trailing empty list marker
	return Nil{}.Append(dst)
}

func (l List) String() string {
	return fmt.Sprint([]Term(l))
}

// Binary represents a raw, unformatted sequence chunk of bytes.
type Binary []byte

func (b Binary) Append(dst []byte) []byte {
	if len(b) > math.MaxUint32 {
		return dst
	}

	dst = append(dst, byte(BinaryExt))
	dst = binary.BigEndian.AppendUint32(dst, uint32(len(b)))
	return append(dst, b...)
}

func (b Binary) String() string {
	return strconv.QuoteToASCII(string(b))
}

// SmallBigInt represents an Erlang (SMALL_BIG_EXT).
type SmallBigInt struct {
	Int big.Int
}

func (b SmallBigInt) Append(dst []byte) []byte {
	sign := b.Int.Sign()
	if sign == 0 {
		return append(dst, byte(SmallBigExt), 0, 0)
	}

	// calculate bytes required (round up)
	n := (b.Int.BitLen() + 7) / 8
	if n > math.MaxUint8 {
		return dst
	}

	// sign byte (1 - negative)
	var s byte = 0
	if sign < 0 {
		s = 1
	}

	words := b.Int.Bits()

	// grow slice to fit all the data
	// tag(1) + n(1) + s(1) + data
	dst = slices.Grow(dst, 3+len(words)*(bits.UintSize/8))
	dst = append(dst, byte(SmallBigExt), byte(n), s)

	// calculate where data should end
	end := len(dst) + n

	// append payload chunks in Little-Endian order
	switch bits.UintSize {
	case 64:
		for _, w := range words {
			dst = binary.LittleEndian.AppendUint64(dst, uint64(w))
		}
	case 32:
		for _, w := range words {
			dst = binary.LittleEndian.AppendUint32(dst, uint32(w))
		}
	}

	// trim trailing whole-word padding bytes
	return dst[:end]
}

func (b SmallBigInt) String() string {
	return b.Int.String()
}

// LargeBigInt represents an Erlang (LARGE_BIG_EXT).
type LargeBigInt struct {
	Int big.Int
}

func (b LargeBigInt) Append(dst []byte) []byte {
	sign := b.Int.Sign()
	if sign == 0 {
		return append(dst, byte(LargeBigExt), 0, 0, 0, 0, 0)
	}

	// calculate bytes required (round up)
	n := (b.Int.BitLen() + 7) / 8
	if n > math.MaxUint32 {
		return dst
	}

	// sign byte (1 - negative)
	var s byte = 0
	if sign < 0 {
		s = 1
	}

	words := b.Int.Bits()

	// grow slice to fit all the data
	// tag(1) + n(4) + s(1) + data
	dst = slices.Grow(dst, 6+len(words)*(bits.UintSize/8))

	dst = append(dst, byte(LargeBigExt))
	dst = binary.BigEndian.AppendUint32(dst, uint32(n))
	dst = append(dst, s)

	// calculate where data should end
	end := len(dst) + n

	// append payload chunks in Little-Endian order
	switch bits.UintSize {
	case 64:
		for _, w := range words {
			dst = binary.LittleEndian.AppendUint64(dst, uint64(w))
		}
	case 32:
		for _, w := range words {
			dst = binary.LittleEndian.AppendUint32(dst, uint32(w))
		}
	}

	// trim trailing whole-word padding bytes
	return dst[:end]
}

func (b LargeBigInt) String() string {
	return b.Int.String()
}

// NewFloat represents a modern 8-byte double precision floating point number (IEEE 754).
type NewFloat float64

func (f NewFloat) Append(dst []byte) []byte {
	dst = append(dst, byte(NewFloatExt))
	return binary.BigEndian.AppendUint64(dst, math.Float64bits(float64(f)))
}

func (f NewFloat) String() string {
	return strconv.FormatFloat(float64(f), 'g', 10, 64)
}

// Atom represents an Erlang named string token symbol.
type Atom string

func (a Atom) Append(dst []byte) []byte {
	if len(a) > math.MaxUint16 {
		return dst
	}

	dst = append(dst, byte(AtomExt))
	dst = binary.BigEndian.AppendUint16(dst, uint16(len(a)))
	return append(dst, a...)
}

func (a Atom) String() string {
	return string(a)
}
