package stream

import (
	"bufio"
	"encoding/binary"
	"io"

	"google.golang.org/protobuf/proto"
)

const MaxRequestBodyLen = 1<<16 - 1

// A Decoder reads and decodes Proto Buffer values from an input stream.
type Decoder struct {
	r *bufio.Reader
}

// NewDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may
// read data from r beyond the Proto Buffer values requested.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReaderSize(r, MaxRequestBodyLen)}
}

// Decode reads the next ProtoBuffer-encoded value from its
// input and stores it in the value pointed to by v.
func (dec *Decoder) Decode(v proto.Message) error {
	var n uint16
	if err := binary.Read(dec.r, binary.BigEndian, &n); err != nil {
		return err
	}
	b, err := dec.r.Peek(int(n))
	if err != nil {
		return err
	}
	defer func() { _, _ = dec.r.Discard(int(n)) }()

	if err := proto.Unmarshal(b, v); err != nil {
		return err
	}
	return nil
}

// An Encoder writes JSON values to an output stream.
type Encoder struct {
	w *bufio.Writer
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: bufio.NewWriterSize(w, MaxRequestBodyLen)}
}

// Encode writes the Proto Buffer encoding of v to the stream.
// preceded by big-endian uint16 len.
func (enc *Encoder) Encode(v proto.Message) error {
	b, err := proto.Marshal(v)
	if err != nil {
		return err
	}

	var n = uint16(len(b))
	if err := binary.Write(enc.w, binary.BigEndian, n); err != nil {
		return err
	}
	if n, err := enc.w.Write(b); err != nil || len(b) != n {
		if err == nil {
			err = io.ErrShortWrite
		}
		return err
	}
	return enc.w.Flush()
}
