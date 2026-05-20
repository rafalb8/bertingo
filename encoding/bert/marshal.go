package bert

import "bytes"

// Marshal converts a Go value into a slice of bytes formatted as a Binary Erlang Term.
// If you pass true for the optional bert2 parameter, it will add a BERT2 length prefix.
func Marshal(v any, bert2 ...bool) ([]byte, error) {
	term, err := ToTerm(v)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	enc := NewEncoder(buf)

	if len(bert2) > 0 && bert2[0] {
		enc.BERT2 = true
	}

	err = enc.Encode(term)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
