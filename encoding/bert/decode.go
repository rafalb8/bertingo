package bert

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type Decoder struct {
	buf   *bufio.Reader
	BERT2 bool
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{buf: bufio.NewReader(r)}
}

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
		return nil, errors.New("bert: invalid version")
	}

	return d.parse()
}

func (d *Decoder) parse() (Term, error) {
	t, err := d.buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("bert: decode tag: %w", err)
	}
	switch t := Tag(t); t {
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
		return nil, fmt.Errorf("bert: decode unsupported type: %s - 0x%X", t.String(), uint8(t))
	}
}

func (d *Decoder) u8() (uint8, error) {
	i, err := d.buf.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("bert: u8: %w", err)
	}
	return i, nil
}

func (d *Decoder) u16() (uint16, error) {
	var i uint16
	err := binary.Read(d.buf, binary.BigEndian, &i)
	if err != nil {
		return 0, fmt.Errorf("bert: u16: %w", err)
	}
	return i, nil
}

func (d *Decoder) u32() (uint32, error) {
	var i uint32
	err := binary.Read(d.buf, binary.BigEndian, &i)
	if err != nil {
		return 0, fmt.Errorf("bert: u32: %w", err)
	}
	return i, nil
}

func (d *Decoder) i32() (int32, error) {
	var i int32
	err := binary.Read(d.buf, binary.BigEndian, &i)
	if err != nil {
		return 0, fmt.Errorf("bert: i32: %w", err)
	}
	return i, nil
}

func (d *Decoder) f64() (float64, error) {
	var f float64
	err := binary.Read(d.buf, binary.BigEndian, &f)
	if err != nil {
		return 0, fmt.Errorf("bert: f64: %w", err)
	}
	return f, nil
}

func (d *Decoder) float() (Float, error) {
	var f float64
	_, err := fmt.Fscanf(d.buf, "%f", &f)
	if err != nil {
		return Float(0), fmt.Errorf("bert: float: %w", err)
	}
	return Float(f), nil
}

func (d *Decoder) tuple(arity uint32) (Tuple, error) {
	var err error
	t := make(Tuple, arity)
	for i := range t {
		t[i], err = d.parse()
		if err != nil {
			return nil, fmt.Errorf("bert: tuple: %w", err)
		}
	}
	return t, nil
}

func (d *Decoder) _map() (Map, error) {
	arity, err := d.u32()
	if err != nil {
		return nil, fmt.Errorf("bert: map: %w", err)
	}
	m := make(Map, arity*2)
	for i := range m {
		m[i], err = d.parse()
		if err != nil {
			return nil, fmt.Errorf("bert: map: %w", err)
		}
	}
	return m, nil
}

func (d *Decoder) list() (List, error) {
	length, err := d.u32()
	if err != nil {
		return nil, fmt.Errorf("bert: list: %w", err)
	}

	list := make(List, length, length+1)
	for i := range list {
		list[i], err = d.parse()
		if err != nil {
			return nil, fmt.Errorf("bert: list: %w", err)
		}
	}

	tail, err := d.parse()
	if err != nil {
		return nil, fmt.Errorf("bert: list: %w", err)
	}
	list = append(list, tail)
	return list, nil
}

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

func (d *Decoder) binary() (Binary, error) {
	length, err := d.u32()
	if err != nil {
		return nil, err
	}

	b := make([]byte, length)
	_, err = io.ReadFull(d.buf, b)
	if err != nil {
		return Binary{}, fmt.Errorf("bert: binary: %w", err)
	}

	return Binary(b), nil
}

func (d *Decoder) atom() (Atom, error) {
	length, err := d.u16()
	if err != nil {
		return Atom(""), fmt.Errorf("bert: atom: %w", err)
	}

	a := make([]byte, length)
	_, err = io.ReadFull(d.buf, a)
	if err != nil {
		return Atom(""), fmt.Errorf("bert: atom: %w", err)
	}

	return Atom(a), nil
}
