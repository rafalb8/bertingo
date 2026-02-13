package bert

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

type Encoder struct {
	buf   *bufio.Writer
	BERT2 bool
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{buf: bufio.NewWriter(w)}
}

func (e *Encoder) Encode(b Term) error {
	chunk := make([]byte, 1, 4<<10)
	chunk[0] = byte(Version)
	chunk = b.Append(chunk)

	if e.BERT2 {
		_, err := e.buf.Write(binary.AppendUvarint(e.buf.AvailableBuffer(), uint64(len(chunk))))
		if err != nil {
			return err
		}
	}

	_, err := e.buf.Write(chunk)
	if err != nil {
		return err
	}
	return e.buf.Flush()
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
