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
			for i := range v.Len() {
				res[i] = byte(v.Index(i).Uint())
			}
			return Binary(res), nil
		}

		if v.Len() == 0 {
			return Nil{}, nil
		}

		optimize := true
		l := make(List, 0, v.Len()+1) // v.len + nil

		for i := range v.Len() {
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

	for opt := range strings.SplitSeq(strings.TrimSpace(options), ",") {
		switch opt {
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
	out := []Term{}

	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		name, flag := parseTag(f)
		v := v.Field(i)

		if flag.omitempty && v.IsZero() {
			continue
		}

		tuple := Tuple{}
		if name != "-" {
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
			val = Atom(v.String())

		case flag.binary:
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
