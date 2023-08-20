package stream

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"

	"google.golang.org/protobuf/proto"
)

const MaxBodyLen = 1<<16 - 1

// A Decoder reads and decodes Proto Buffer values from an input stream.
type Decoder struct {
	r *bufio.Reader
	b []byte
}

// NewDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may
// read data from r beyond the Proto Buffer values requested.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

const minBufSize = 256

// Decode reads the next ProtoBuffer-encoded value from its
// input and stores it in the value pointed to by v.
func (dec *Decoder) Decode(m proto.Message) error {
	var nn uint16
	if err := binary.Read(dec.r, binary.BigEndian, &nn); err != nil {
		return err
	}

	n := int(nn)
	l := len(dec.b)
	for l < n {
		l <<= 1
		if l < minBufSize {
			l = minBufSize
		}
		dec.b = make([]byte, l)
	}
	_, err := io.ReadFull(dec.r, dec.b[:n])
	if err != nil {
		return err
	}

	return proto.Unmarshal(dec.b[:n], m)
}

// An Encoder writes JSON values to an output stream.
type Encoder struct {
	w *bufio.Writer
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: bufio.NewWriter(w)}
}

var ErrMessageTooLarge = errors.New("message too large")

// Encode writes the Proto Buffer encoding of v to the stream.
// preceded by big-endian uint16 len.
func (enc *Encoder) Encode(m proto.Message) error {
	b, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	n := len(b)
	if n > MaxBodyLen {
		return ErrMessageTooLarge
	}
	if err := binary.Write(enc.w, binary.BigEndian, uint16(n)); err != nil {
		return err
	}
	if wn, werr := enc.w.Write(b); n != wn {
		if werr == nil {
			werr = io.ErrShortWrite
		}
		return werr
	}
	return enc.w.Flush()
}
