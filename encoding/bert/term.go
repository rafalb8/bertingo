package bert

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
)

type Term interface {
	Append(dst []byte) []byte
	String() string
}

type SmallInteger uint8

func (i SmallInteger) Append(dst []byte) []byte {
	return append(dst, byte(SmallIntegerExt), byte(i))
}

func (i SmallInteger) String() string {
	return strconv.FormatUint(uint64(i), 10)
}

type Integer int32

func (i Integer) Append(dst []byte) []byte {
	dst = append(dst, byte(IntegerExt))
	return binary.BigEndian.AppendUint32(dst, uint32(i))
}

func (i Integer) String() string {
	return strconv.FormatInt(int64(i), 10)
}

// Finite float stored as %.20e formatted string
type Float float64

func (f Float) Append(dst []byte) []byte {
	dst = append(dst, byte(FloatExt))
	return fmt.Appendf(dst, "%.20e", float64(f))
}

func (f Float) String() string {
	return strconv.FormatFloat(float64(f), 'g', 10, 64)
}

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

type Map []Term

func (m Map) Append(dst []byte) []byte {
	if len(m)/2 > math.MaxUint32 {
		return dst
	}
	if len(m)%2 != 0 {
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

type Nil struct{}

func (Nil) Append(dst []byte) []byte {
	return append(dst, byte(NilExt))
}

func (Nil) String() string {
	return ""
}

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

type List []Term

func (l List) Append(dst []byte) []byte {
	if len(l) == 0 {
		return Nil{}.Append(dst)
	}

	len := len(l) - 1
	if len > math.MaxUint32 {
		return dst
	}

	dst = append(dst, byte(ListExt))
	dst = binary.BigEndian.AppendUint32(dst, uint32(len))

	for _, a := range l {
		dst = a.Append(dst)
	}
	return dst
}

func (l List) String() string {
	return fmt.Sprint([]Term(l))
}

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

// Finite float stored as 8 bytes big-endian IEEE format
type NewFloat float64

func (f NewFloat) Append(dst []byte) []byte {
	dst = append(dst, byte(NewFloatExt))
	return binary.BigEndian.AppendUint64(dst, math.Float64bits(float64(f)))
}

func (f NewFloat) String() string {
	return strconv.FormatFloat(float64(f), 'g', 10, 64)
}

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
