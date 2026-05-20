package bert

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

var ErrNotImplemented = errors.New("bert: not implemented")

func Marshal(v any) (Term, error) {
	return marshal(reflect.Indirect(reflect.ValueOf(v)))
}

func marshal(v reflect.Value) (Term, error) {
	switch k := v.Kind(); k {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return Tuple{Atom("bert"), Atom("nil")}, nil
		}
		return marshal(v.Elem())

	case reflect.Bool:
		return Atom(strconv.FormatBool(v.Bool())), nil

	case reflect.String:
		return String(v.String()), nil

	case reflect.Uint:
		switch i := v.Uint(); {
		case i <= math.MaxUint8:
			return SmallInteger(i), nil
		case i <= math.MaxInt32:
			return Integer(i), nil
		default:
			// TODO: BigInt
			return nil, ErrNotImplemented
		}

	case reflect.Uint8:
		return SmallInteger(v.Uint()), nil

	case reflect.Uint16:
		return Integer(v.Uint()), nil

	case reflect.Uint32, reflect.Uint64:
		// TODO: BigInt
		return nil, ErrNotImplemented

	case reflect.Int:
		switch i := v.Int(); {
		case i >= 0 && i <= math.MaxUint8:
			return SmallInteger(i), nil
		case i >= math.MinInt32 && i <= math.MaxInt32:
			return Integer(i), nil
		default:
			// TODO: BigInt
			return nil, ErrNotImplemented
		}

	case reflect.Int8, reflect.Int16, reflect.Int32:
		return Integer(v.Int()), nil

	case reflect.Int64:
		// TODO: BigInt
		return nil, ErrNotImplemented

	case reflect.Float32, reflect.Float64:
		return NewFloat(v.Float()), nil

	case reflect.Array, reflect.Slice:
		// []uint8/byte encode as binary
		if v.Type().Elem().Kind() == reflect.Uint8 {
			if v.Kind() == reflect.Slice || v.CanAddr() {
				return Binary(v.Bytes()), nil
			}

			// fallback for arrays passed by value
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
		l := make(List, 0, length+1) // v.len + nil

		for i := range length {
			x, err := marshal(v.Index(i))
			if err != nil {
				return nil, err
			}
			l = append(l, x)

			// check if x's are SmallIntgers => can be optimized into string
			if optimize {
				_, optimize = x.(SmallInteger)
			}
		}

		// if only SmallIntegers, flatten it to a String
		if optimize {
			buf := make([]byte, len(l))
			for i, x := range l {
				buf[i] = byte(x.(SmallInteger))
			}
			return String(unsafe.String(unsafe.SliceData(buf), len(buf))), nil
		}

		return append(l, Nil{}), nil

	case reflect.Struct:
		return marshalStruct(v, false)

	default:
		return nil, fmt.Errorf("bert: unsupported type: %s", k.String())
	}
}

type flags struct {
	omitempty bool
	binary    bool
	list      bool
	atom      bool
}

func parseTag(field reflect.StructField) (name string, flag flags) {
	tag, found := field.Tag.Lookup("bert")
	if !found {
		return field.Name, flag
	}

	tag, options, found := strings.Cut(tag, ",")
	if !found {
		return cmp.Or(tag, field.Name), flag
	}

	for opt := range strings.SplitSeq(options, ",") {
		switch strings.TrimSpace(opt) {
		case "omitempty":
			flag.omitempty = true
		case "binary":
			flag.binary = true
		case "list":
			flag.list = true
		case "atom":
			flag.atom = true
		}
	}
	return cmp.Or(tag, field.Name), flag
}

func marshalStruct(v reflect.Value, list bool) (Term, error) {
	termNum := v.NumField()
	if list {
		termNum += 1 // NumField + nil
	} else {
		termNum *= 2 // NumField * (Name + Value)
	}

	out := make([]Term, 0, termNum)

	for f, v := range v.Fields() {
		name, flag := parseTag(f)

		if name == "-" {
			continue
		}

		if flag.omitempty && v.IsZero() {
			continue
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
			val, err = marshalStruct(v, true)

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
			val, err = marshal(v)
		}
		if err != nil {
			return nil, fmt.Errorf("bert: marshalStruct: %w", err)
		}

		tuple = append(tuple, val)

		if list {
			// in list mode, pair name atom with value in tuple
			out = append(out, tuple)
		} else {
			out = append(out, tuple...)
		}
	}

	if list {
		// return proper list
		return List(append(out, Nil{})), nil
	}
	return Tuple(out), nil
}
