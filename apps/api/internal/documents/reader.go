package documents

import "io"

// prefixReader returns a reader that first yields prefix bytes, then reads from r.
type prefixReader struct {
	prefix []byte
	offset int
	r      io.Reader
}

func newPrefixReader(prefix []byte, r io.Reader) io.Reader {
	return &prefixReader{prefix: prefix, r: r}
}

func (p *prefixReader) Read(buf []byte) (int, error) {
	if p.offset < len(p.prefix) {
		n := copy(buf, p.prefix[p.offset:])
		p.offset += n
		return n, nil
	}
	return p.r.Read(buf)
}
