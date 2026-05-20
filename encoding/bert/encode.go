package bert

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 4<<10)
		return &b
	},
}

type Encoder struct {
	w io.Writer

	BERT2 bool
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) Encode(term Term) error {
	p := bufferPool.Get().(*[]byte)
	buf := (*p)[:0]

	buf = append(buf, byte(Version))
	buf = term.Append(buf)

	if e.BERT2 {
		var prefix [binary.MaxVarintLen64]byte
		n := binary.PutUvarint(prefix[:], uint64(len(buf)))
		if _, err := e.w.Write(prefix[:n]); err != nil {
			bufferPool.Put(p)
			return err
		}
	}

	_, err := e.w.Write(buf)
	if cap(buf) <= 64<<10 {
		*p = buf
		bufferPool.Put(p)
	}
	return err
}

func Tree(b Term, prefix ...byte) string {
	pfx := string(prefix)
	sb := strings.Builder{}
	switch b := b.(type) {
	case Tuple:
		sb.WriteString(pfx + "Tuple: {\n")
		for _, v := range b {
			sb.WriteString(Tree(v, []byte(pfx+"  ")...))
		}
		sb.WriteString(pfx + "}\n")

	case List:
		sb.WriteString(pfx + "List: {\n")
		for _, v := range b {
			sb.WriteString(Tree(v, []byte(pfx+"  ")...))
		}
		sb.WriteString(pfx + "}\n")

	case Binary:
		sb.WriteString(pfx + "Binary: <<" + b.String() + ">>\n")

	case Nil:
		sb.WriteString(pfx + "Nil\n")

	default:
		typ := strings.TrimLeft(fmt.Sprintf("%T: ", b), "bert.")
		sb.WriteString(pfx + typ + b.String() + "\n")
	}
	return sb.String()
}
