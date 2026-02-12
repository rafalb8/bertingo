package bert

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
)

type Term interface {
	Append(dst []byte) ([]byte, error)
	String() string
}

type SmallInteger uint8

func (i SmallInteger) Append(dst []byte) ([]byte, error) {
	return append(dst, byte(SmallIntegerExt), byte(i)), nil
}

func (i SmallInteger) String() string {
	return strconv.FormatUint(uint64(i), 10)
}

type Integer int32

func (i Integer) Append(dst []byte) ([]byte, error) {
	dst = append(dst, byte(IntegerExt))
	return binary.BigEndian.AppendUint32(dst, uint32(i)), nil
}

func (i Integer) String() string {
	return strconv.FormatInt(int64(i), 10)
}

// Finite float stored as %.20e formatted string
type Float float64

func (f Float) Append(dst []byte) ([]byte, error) {
	dst = append(dst, byte(FloatExt))
	return fmt.Appendf(dst, "%.20e", float64(f)), nil
}

func (f Float) String() string {
	return strconv.FormatFloat(float64(f), 'g', 10, 64)
}

type Tuple []Term

func (t Tuple) Append(dst []byte) ([]byte, error) {
	var err error
	switch {
	case len(t) <= math.MaxUint8:
		dst = append(dst, byte(SmallTupleExt), byte(len(t)))
	case len(t) <= math.MaxUint32:
		dst = append(dst, byte(LargeTupleExt))
		dst = binary.BigEndian.AppendUint32(dst, uint32(len(t)))
	default:
		return dst, errors.New("bert: tuple too large")
	}

	for _, a := range t {
		dst, err = a.Append(dst)
		if err != nil {
			return dst, err
		}
	}
	return dst, nil
}

func (t Tuple) String() string {
	return fmt.Sprint([]Term(t))
}

type Map []Term

func (m Map) Append(dst []byte) ([]byte, error) {
	if len(m)/2 > math.MaxUint32 {
		return dst, errors.New("bert: map too large")
	}
	if len(m)%2 != 0 {
		return dst, errors.New("bert: map must have even number of elements")
	}

	dst = append(dst, byte(MapExt))
	dst = binary.BigEndian.AppendUint32(dst, uint32(len(m)/2))

	for _, a := range m {
		dst, err := a.Append(dst)
		if err != nil {
			return dst, err
		}
	}
	return dst, nil
}

func (m Map) String() string {
	return fmt.Sprint([]Term(m))
}

type Nil struct{}

func (Nil) Append(dst []byte) ([]byte, error) {
	return append(dst, byte(NilExt)), nil
}

func (Nil) String() string {
	return ""
}

type String string

func (s String) Append(dst []byte) ([]byte, error) {
	if len(s) == 0 {
		return Nil{}.Append(dst)
	}

	if len(s) > math.MaxUint16 {
		return dst, errors.New("bert: string too long")
	}

	dst = append(dst, byte(StringExt))
	dst = binary.BigEndian.AppendUint16(dst, uint16(len(s)))
	return append(dst, s...), nil
}

func (s String) String() string {
	return strconv.QuoteToGraphic(string(s))
}

type List []Term

func (l List) Append(dst []byte) ([]byte, error) {
	if len(l) == 0 {
		return Nil{}.Append(dst)
	}

	len := len(l) - 1
	if len > math.MaxUint32 {
		return dst, errors.New("bert: list too large")
	}

	dst = append(dst, byte(ListExt))
	dst = binary.BigEndian.AppendUint32(dst, uint32(len))

	for _, a := range l {
		dst, err := a.Append(dst)
		if err != nil {
			return dst, err
		}
	}
	return dst, nil
}

func (l List) String() string {
	return fmt.Sprint([]Term(l))
}

type Binary []byte

func (b Binary) Append(dst []byte) ([]byte, error) {
	if len(b) > math.MaxUint16 {
		return dst, errors.New("bert: binary too long")
	}

	dst = append(dst, byte(BinaryExt))
	dst = binary.BigEndian.AppendUint32(dst, uint32(len(b)))
	return append(dst, b...), nil
}

func (b Binary) String() string {
	return strconv.QuoteToASCII(string(b))
}

// Finite float stored as 8 bytes big-endian IEEE format
type NewFloat float64

func (f NewFloat) Append(dst []byte) ([]byte, error) {
	dst = append(dst, byte(NewFloatExt))
	return binary.BigEndian.AppendUint64(dst, math.Float64bits(float64(f))), nil
}

func (f NewFloat) String() string {
	return strconv.FormatFloat(float64(f), 'g', 10, 64)
}

type Atom string

func (a Atom) Append(dst []byte) ([]byte, error) {
	if len(a) > 255 {
		return dst, errors.New("bert: atom too long")
	}

	dst = append(dst, byte(AtomExt))
	dst = binary.BigEndian.AppendUint16(dst, uint16(len(a)))
	return append(dst, a...), nil
}

func (a Atom) String() string {
	return string(a)
}
