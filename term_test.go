package bert

import (
	"fmt"
	"strings"
	"testing"
)

func TestAppend(t *testing.T) {
	aLotOfTerms := make([]Term, 300)
	for i := range aLotOfTerms {
		aLotOfTerms[i] = Integer(i)
	}

	tc := map[string]struct {
		Term     Term
		Prefix   []byte
		Expected []byte
	}{
		"SmallInteger": {
			Term:     SmallInteger(0x1D),
			Expected: []byte{byte(SmallIntegerExt), 0x1D},
		},

		"Integer": {
			Term:     Integer(0xFAFA),
			Expected: []byte{byte(IntegerExt), 0, 0, 0xFA, 0xFA},
		},

		"Float": {
			Term:     Float(0.5),
			Expected: fmt.Appendf([]byte{byte(FloatExt)}, "%.20e", 0.5),
		},

		"SmallTuple": {
			Term:     Tuple{Atom("Int"), SmallInteger(0xFF)},
			Expected: []byte{byte(SmallTupleExt), 2, byte(AtomExt), 0, 3, 'I', 'n', 't', byte(SmallIntegerExt), 0xFF},
		},

		"BigTuple": {
			Term:   Tuple(aLotOfTerms),
			Prefix: []byte{byte(LargeTupleExt), 0, 0, 0x01, 0x2C},
		},

		"Map": {
			Term:   Map(aLotOfTerms),
			Prefix: []byte{byte(MapExt), 0, 0, 0, 0x96},
		},

		"Nil": {
			Term:   Nil{},
			Prefix: []byte{byte(NilExt)},
		},

		"String": {
			Term:     String("a"),
			Expected: []byte{byte(StringExt), 0, 1, 'a'},
		},

		"List": {
			Term:     List{Atom("a"), SmallInteger(16)},
			Expected: []byte{byte(ListExt), 0, 0, 0, 2, byte(AtomExt), 0, 1, 'a', byte(SmallIntegerExt), 16, byte(NilExt)},
		},

		"Binary": {
			Term:     Binary("a"),
			Expected: []byte{byte(BinaryExt), 0, 0, 0, 1, 'a'},
		},

		"NewFloat": {
			Term:     NewFloat(3.14),
			Expected: []byte{byte(NewFloatExt), 64, 9, 30, 184, 81, 235, 133, 31},
		},

		"Atom": {
			Term:     Atom("atom"),
			Expected: []byte{byte(AtomExt), 0, 4, 'a', 't', 'o', 'm'},
		},
	}

	for name, tt := range tc {
		t.Run(name, func(t *testing.T) {
			result := make([]byte, 0, len(tt.Expected))
			result = tt.Term.Append(result)
			if len(tt.Expected) > 0 && string(result) != string(tt.Expected) {
				t.Errorf("unexpected result:\ngot:\t%+v\nexpected:\t%+v", result, tt.Expected)
			}

			if len(tt.Prefix) > 0 && !strings.HasPrefix(string(result), string(tt.Prefix)) {
				t.Errorf("prefix not found:\ngot:\t%+v\nprefix:\t%+v", result, tt.Prefix)
			}
		})
	}
}
