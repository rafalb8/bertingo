package bert

import "bytes"

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
