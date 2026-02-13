package large

import (
	"errors"
	"io"
	"os"
)

var ALLOC_SIZE = 4096

type bufferFetch struct {
	reader    io.Reader
	allocSize int
}

func (b *bufferFetch) fetch() ([]byte, error) {
	if b.reader == nil {
		return nil, io.EOF
	}
	if b.allocSize*2 <= ALLOC_SIZE {
		b.allocSize *= 2
	}
	buffer := make([]byte, b.allocSize)
	n, err := b.reader.Read(buffer)
	if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
		b.reader = nil
	}
	return buffer[:n], err
}
