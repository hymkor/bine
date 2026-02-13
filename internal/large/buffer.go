package large

import (
	"errors"
	"io"
	"os"
)

type chunk = []byte

type Buffer struct {
	*Storage
	Fetch    func() ([]byte, error)
	TryFetch func() ([]byte, error)
}

func NewBuffer(r io.Reader) *Buffer {
	f := &bufferFetch{
		reader:    r,
		allocSize: 8,
	}
	return &Buffer{
		Storage:  newStorage(),
		Fetch:    f.fetch,
		TryFetch: f.fetch,
	}
}

func (b *Buffer) fetchAndStore() error {
	data, err := b.Fetch()
	b.Store(data, err)
	return err
}

func (b *Buffer) tryFetchAndStore() error {
	data, err := b.TryFetch()
	b.Store(data, err)
	return err
}

func (b *Buffer) ReadAll() error {
	for {
		err := b.fetchAndStore()
		if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func (b *Buffer) WriteTo(w io.Writer) (int64, error) {
	if err := b.ReadAll(); err != nil {
		return 0, err
	}
	n := int64(0)
	for p := b.lines.Front(); p != nil; p = p.Next() {
		m, err := w.Write(p.Value.(chunk))
		n += int64(m)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
