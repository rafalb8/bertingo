package bert

import "bytes"

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
