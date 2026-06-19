package bert

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

// Marshal converts a Go value into a slice of bytes formatted as a Binary Erlang Term.
// If `bert2` is true, it will add a BERT2 length prefix.
func Marshal(v any, bert2 bool) ([]byte, error) {
	term, err := AsTerm(v)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	enc := NewEncoder(buf)
	enc.BERT2 = bert2

	err = enc.Encode(term)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type AsTermer interface {
	AsTerm() (Term, error)
}

// AsTerm converts any Go value into an Erlang data Term object.
func AsTerm(v any) (Term, error) {
	return toTerm(reflect.ValueOf(v))
}

func toTerm(v reflect.Value) (Term, error) {
	if v.CanInterface() {
		if v, ok := v.Interface().(AsTermer); ok {
			return v.AsTerm()
		}
	}

	switch k := v.Kind(); k {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			// map nil items to a generic Erlang standard null representation tuple: {bert, nil}
			return Tuple{Atom("bert"), Atom("nil")}, nil
		}
		return toTerm(v.Elem())

	case reflect.Bool:
		return Atom(strconv.FormatBool(v.Bool())), nil

	case reflect.String:
		return String(v.String()), nil

	case reflect.Uint:
		switch x := v.Uint(); {
		case x <= math.MaxUint8:
			return SmallInteger(x), nil
		case x <= math.MaxInt32:
			return Integer(x), nil
		default:
			b := SmallBigInt{}
			b.Int.SetUint64(x)
			return b, nil
		}

	case reflect.Uint8:
		return SmallInteger(v.Uint()), nil

	case reflect.Uint16:
		return Integer(v.Uint()), nil

	case reflect.Uint32, reflect.Uint64:
		b := SmallBigInt{}
		b.Int.SetUint64(v.Uint())
		return b, nil

	case reflect.Int:
		switch x := v.Int(); {
		case x >= 0 && x <= math.MaxUint8:
			return SmallInteger(x), nil
		case x >= math.MinInt32 && x <= math.MaxInt32:
			return Integer(x), nil
		default:
			b := SmallBigInt{}
			b.Int.SetInt64(x)
			return b, nil
		}

	case reflect.Int8, reflect.Int16, reflect.Int32:
		return Integer(v.Int()), nil

	case reflect.Int64:
		b := SmallBigInt{}
		b.Int.SetInt64(v.Int())
		return b, nil

	case reflect.Float32, reflect.Float64:
		return NewFloat(v.Float()), nil

	case reflect.Array, reflect.Slice:
		// Check if it is a list of raw bytes or uint8 characters
		if v.Type().Elem().Kind() == reflect.Uint8 {
			if v.Kind() == reflect.Slice || v.CanAddr() {
				return Binary(v.Bytes()), nil
			}

			// Copy standard arrays passed by value safely into a separate byte buffer slice
			res := make([]byte, v.Len())
			for i := range res {
				res[i] = byte(v.Index(i).Uint())
			}
			return Binary(res), nil
		}

		length := v.Len()
		if length == 0 {
			return Nil{}, nil
		}

		optimize := true
		l := make(List, 0, length)

		for i := range length {
			x, err := toTerm(v.Index(i))
			if err != nil {
				return nil, err
			}
			l = append(l, x)

			// check if every individual component inside is a SmallInteger
			if optimize {
				_, optimize = x.(SmallInteger)
			}
		}

		// if the elements are all SmallInteger, flatten them into a single String representation
		if optimize {
			buf := make([]byte, len(l))
			for i, x := range l {
				buf[i] = byte(x.(SmallInteger))
			}
			return String(unsafe.String(unsafe.SliceData(buf), len(buf))), nil
		}

		return l, nil

	case reflect.Map:
		m := make(Map, 0, v.Len()*2)
		it := v.MapRange()
		for it.Next() {
			k, err := toTerm(it.Key())
			if err != nil {
				return nil, fmt.Errorf("bert: map key: %w", err)
			}

			v, err := toTerm(it.Value())
			if err != nil {
				return nil, fmt.Errorf("bert: map value: %w", err)
			}
			m = append(m, k, v)
		}
		return m, nil

	case reflect.Struct:
		return structToTerm(v, false)

	default:
		return nil, fmt.Errorf("bert: unsupported type: %s", k.String())
	}
}

// flags keeps track of custom configuration tags attached to struct fields.
type flags struct {
	omitempty bool // Skip if the field has its "empty" value
	omitzero  bool // Skip if the field has its zero value
	binary    bool // Force a Go string to encode as a raw Erlang Binary block
	list      bool // Force a child struct to layout components like an Erlang List array
	atom      bool // Force a Go string to convert into an Erlang token Atom symbol
}

// parseTag inspects a struct field's `bert:"..."` tag values.
func parseTag(field reflect.StructField) (name string, flag flags) {
	tag, found := field.Tag.Lookup("bert")
	if !found {
		return field.Name, flag
	}

	tag, options, found := strings.Cut(tag, ",")
	if !found {
		return tag, flag
	}

	for opt := range strings.SplitSeq(options, ",") {
		switch strings.TrimSpace(opt) {
		case "omitempty":
			flag.omitempty = true
		case "omitzero":
			flag.omitzero = true
		case "binary":
			flag.binary = true
		case "list":
			flag.list = true
		case "atom":
			flag.atom = true
		}
	}
	return tag, flag
}

// structToTerm transforms standard Go structures into nested Erlang key-value fields.
func structToTerm(v reflect.Value, list bool) (Term, error) {
	termNum := v.NumField()
	if list {
		termNum += 1 // NumField + nil
	} else {
		termNum *= 2 // NumField * (Name + Value)
	}

	out := make([]Term, 0, termNum)

	for f, v := range v.Fields() {
		name, flag := parseTag(f)

		if name == "-" { // Skip fields explicitly ignored with a `-` tag
			continue
		}

		if flag.omitempty && v.IsZero() {
			continue
		}

		if flag.omitzero {
			hasIsZero := false
			if v.CanInterface() {
				if z, ok := v.Interface().(interface{ IsZero() bool }); ok {
					hasIsZero = true
					if z.IsZero() {
						continue
					}
				}
			}

			if !hasIsZero && v.IsZero() {
				continue
			}
		}

		tuple := make(Tuple, 0, 2)
		if name != "" {
			tuple = append(tuple, Atom(name))
		}

		var val Term
		var err error
		switch {
		case flag.list:
			v = reflect.Indirect(v)
			if v.Kind() == reflect.Interface {
				v = v.Elem()
			}
			val, err = structToTerm(v, true)

		case flag.atom:
			if v.Kind() != reflect.String {
				return nil, fmt.Errorf("bert: atom tag can only be used with string: %s", f.Name)
			}
			val = Atom(v.String())

		case flag.binary:
			if v.Kind() != reflect.String {
				return nil, fmt.Errorf("bert: binary tag can only be used with string: %s", f.Name)
			}
			val = Binary(v.String())

		default:
			val, err = toTerm(v)
		}
		if err != nil {
			return nil, fmt.Errorf("bert: structToTerm: %w", err)
		}

		tuple = append(tuple, val)

		if list {
			out = append(out, tuple)
		} else {
			out = append(out, tuple...)
		}
	}

	if list {
		return List(out), nil
	}
	return Tuple(out), nil
}
