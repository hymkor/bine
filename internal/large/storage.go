package large

import (
	"container/list"
	"errors"
	"io"
	"os"
)

type Storage struct {
	lines   *list.List
	allsize int64
}

func newStorage() *Storage {
	return &Storage{
		lines:   list.New(),
		allsize: 0,
	}
}

func (b *Storage) Len() int64 {
	return b.allsize
}

func (b *Storage) Store(data []byte, err error) bool {
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrDeadlineExceeded) {
		return false
	}
	if len(data) > 0 {
		b.lines.PushBack(chunk(data))
		b.allsize += int64(len(data))
	}
	return !errors.Is(err, os.ErrDeadlineExceeded)
}
