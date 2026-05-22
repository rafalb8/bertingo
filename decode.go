package bert

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
)

var ErrVersion = errors.New("bert: invalid version")

type reader interface {
	io.Reader
	io.ByteReader
	Peek(n int) ([]byte, error)
	Discard(n int) (int, error)
}

// Decoder parses a Binary Erlang Term stream into abstract representative Go Terms.
type Decoder struct {
	buf reader

	BERT2 bool
}

// NewDecoder creates an BERT data decoder
func NewDecoder(r io.Reader) *Decoder {
	if br, ok := r.(reader); ok {
		return &Decoder{buf: br}
	}
	return &Decoder{buf: bufio.NewReader(r)}
}

// Decode reads the stream and extracts the next Erlang data term.
// Returns an error if structural markers or version headers fail validation.
func (d *Decoder) Decode() (Term, error) {
	if d.BERT2 {
		_, err := binary.ReadUvarint(d.buf)
		if err != nil {
			if err == io.EOF {
				return nil, err
			}
			return nil, fmt.Errorf("bert: decode BERT2: %w", err)
		}
	}

	magic, err := d.buf.ReadByte()
	if err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, fmt.Errorf("bert: decode magic: %w", err)
	}

	if magic != byte(Version) {
		return nil, ErrVersion
	}

	return d.parse()
}

func (d *Decoder) parse() (Term, error) {
	t, err := d.buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("bert: decode tag: %w", err)
	}
	switch Tag(t) {
	case SmallIntegerExt:
		u, err := d.u8()
		return SmallInteger(u), err

	case IntegerExt:
		i, err := d.i32()
		return Integer(i), err

	case FloatExt:
		return d.float()

	case SmallTupleExt:
		arity, err := d.u8()
		if err != nil {
			return nil, err
		}
		return d.tuple(uint32(arity))

	case LargeTupleExt:
		arity, err := d.u32()
		if err != nil {
			return nil, err
		}
		return d.tuple(arity)

	case MapExt:
		return d._map()

	case NilExt:
		return Nil{}, nil

	case StringExt:
		return d.string()

	case ListExt:
		return d.list()

	case BinaryExt:
		return d.binary()

	case NewFloatExt:
		f, err := d.f64()
		return NewFloat(f), err

	case AtomExt:
		return d.atom()

	default:
		return nil, fmt.Errorf("bert: decode unsupported type: 0x%X", uint8(t))
	}
}

// -- Fixed-Size Primitives --

// u8 pulls a single raw byte straight from the buffer stream.
func (d *Decoder) u8() (uint8, error) {
	i, err := d.buf.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("bert: u8: %w", err)
	}
	return i, nil
}

// u16 parses a big-endian 16-bit unsigned integer.
func (d *Decoder) u16() (uint16, error) {
	b, err := d.buf.Peek(2)
	if err != nil {
		return 0, fmt.Errorf("bert: u16: %w", err)
	}
	i := binary.BigEndian.Uint16(b)
	_, _ = d.buf.Discard(2)
	return i, nil
}

// u32 parses a big-endian 32-bit unsigned integer.
func (d *Decoder) u32() (uint32, error) {
	b, err := d.buf.Peek(4)
	if err != nil {
		return 0, fmt.Errorf("bert: u32: %w", err)
	}
	i := binary.BigEndian.Uint32(b)
	_, _ = d.buf.Discard(4)
	return i, nil
}

// i32 parses a big-endian signed 32-bit integer.
func (d *Decoder) i32() (int32, error) {
	val, err := d.u32()
	return int32(val), err
}

// f64 parses an IEEE 754 double-precision float.
func (d *Decoder) f64() (float64, error) {
	b, err := d.buf.Peek(8)
	if err != nil {
		return 0, fmt.Errorf("bert: f64: %w", err)
	}
	bits := binary.BigEndian.Uint64(b)
	_, _ = d.buf.Discard(8)
	return math.Float64frombits(bits), nil
}

// --- Struct & Collection Parsers ---

// float reads obsolete legacy Erlang string floats, which are 31-byte text blocks.
func (d *Decoder) float() (Float, error) {
	// Erlang FloatExt is exactly 31 bytes, null-padded.
	var buf [31]byte
	if _, err := io.ReadFull(d.buf, buf[:]); err != nil {
		return Float(0), fmt.Errorf("bert: float: %w", err)
	}

	// find the null terminator or space truncation boundary
	end := 0
	for end < len(buf) && buf[end] != 0 {
		end++
	}

	f, err := strconv.ParseFloat(string(buf[:end]), 64)
	if err != nil {
		return Float(0), fmt.Errorf("bert: float parse: %w", err)
	}
	return Float(f), nil
}

// tuple deserializes sequential structures matching a strict, expected layout dimension.
func (d *Decoder) tuple(arity uint32) (Tuple, error) {
	// guard against un-allocated memory explosions
	if arity > 1024 {
		return nil, fmt.Errorf("bert: tuple arity %d exceeds safety limit", arity)
	}

	var err error
	t := make(Tuple, arity)
	for i := range t {
		t[i], err = d.parse()
		if err != nil {
			return nil, fmt.Errorf("bert: tuple step: %w", err)
		}
	}
	return t, nil
}

// _map decodes Erlang map key/value sequence pairings sequentially into flat array slices.
func (d *Decoder) _map() (Map, error) {
	arity, err := d.u32()
	if err != nil {
		return nil, fmt.Errorf("bert: map size: %w", err)
	}
	if arity > 1024 {
		return nil, fmt.Errorf("bert: map size %d exceeds safety limit", arity)
	}

	m := make(Map, arity*2)
	for i := range m {
		m[i], err = d.parse()
		if err != nil {
			return nil, fmt.Errorf("bert: map step: %w", err)
		}
	}
	return m, nil
}

// list handles nested list data and drops the trailing structural Nil element
// if it describes a proper, standard list layout.
func (d *Decoder) list() (List, error) {
	length, err := d.u32()
	if err != nil {
		return nil, fmt.Errorf("bert: list size: %w", err)
	}
	if length > 4096 {
		return nil, fmt.Errorf("bert: list size %d exceeds safety limit", length)
	}

	list := make(List, length, length+1)
	for i := range list {
		list[i], err = d.parse()
		if err != nil {
			return nil, fmt.Errorf("bert: list step: %w", err)
		}
	}

	tail, err := d.parse()
	if err != nil {
		return nil, fmt.Errorf("bert: list tail: %w", err)
	}

	// Erlang lists end with an empty list/nil if proper, or an arbitrary term if improper.
	if _, ok := tail.(Nil); !ok {
		list = append(list, tail)
	}
	return list, nil
}

// string decodes an Erlang list of small byte sequences representation.
func (d *Decoder) string() (String, error) {
	length, err := d.u16()
	if err != nil {
		return String(""), err
	}

	s := make([]byte, length)
	_, err = io.ReadFull(d.buf, s)
	if err != nil {
		return String(""), fmt.Errorf("bert: string: %w", err)
	}

	return String(s), nil
}

// binary pulls an arbitrary raw byte chunk.
func (d *Decoder) binary() (Binary, error) {
	length, err := d.u32()
	if err != nil {
		return nil, err
	}

	b := make([]byte, length)
	_, err = io.ReadFull(d.buf, b)
	if err != nil {
		return nil, fmt.Errorf("bert: binary read: %w", err)
	}

	return Binary(b), nil
}

// atom parses string token identifiers.
func (d *Decoder) atom() (Atom, error) {
	length, err := d.u16()
	if err != nil {
		return Atom(""), fmt.Errorf("bert: atom size: %w", err)
	}

	a := make([]byte, length)
	_, err = io.ReadFull(d.buf, a)
	if err != nil {
		return Atom(""), fmt.Errorf("bert: atom read: %w", err)
	}

	return Atom(a), nil
}
