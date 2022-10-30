package gobmarshaler

import (
	"encoding/gob"
	"io"

	"github.com/go-orb/orb/util/marshaler"
)

const Name = "gob"

func init() {
	err := marshaler.Plugins.Add(Name, New)
	if err != nil {
		panic(err)
	}
}

type Marshaler struct {
	enc *gob.Encoder
	dec *gob.Decoder
}

func New() marshaler.Marshaler {
	return &Marshaler{}
}

func (g *Marshaler) String() string {
	return Name
}

func (g *Marshaler) FileExtension() string {
	return ""
}

func (g *Marshaler) Init(r io.Reader, w io.Writer) error {
	if r == nil && w == nil {
		return marshaler.ErrNoSocket
	}

	if r != nil {
		g.dec = gob.NewDecoder(r)
	}

	if w != nil {
		g.enc = gob.NewEncoder(w)
	}

	return nil
}

func (g *Marshaler) EncodeSocket(v any) error {
	if g.enc == nil {
		return marshaler.ErrNoSocket
	}

	return g.enc.Encode(v)
}

func (g *Marshaler) DecodeSocket(v any) error {
	if g.dec == nil {
		return marshaler.ErrNoSocket
	}

	return g.dec.Decode(v)
}
