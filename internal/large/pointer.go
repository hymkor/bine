package large

import (
	"container/list"
	"io"
)

type Pointer struct {
	buffer  *Buffer
	address int64
	element *list.Element
	offset  int
}

func (p *Pointer) Clone() *Pointer {
	clone := *p
	return &clone
}

func (p *Pointer) Address() int64 { return p.address }

func NewPointer(b *Buffer) *Pointer {
	element := b.lines.Front()
	if element == nil {
		if err := b.fetchAndStore(); err != nil && err != io.EOF {
			return nil
		}
		element = b.lines.Front()
		if element == nil {
			return nil
		}
	}
	return &Pointer{
		buffer:  b,
		address: 0,
		element: element,
		offset:  0,
	}
}

func NewPointerAt(at int64, b *Buffer) *Pointer {
	p := NewPointer(b)
	if p != nil {
		p.Skip(at)
	}
	return p
}

func (p *Pointer) Value() byte {
	return p.element.Value.(chunk)[p.offset]
}

func (p *Pointer) SetValue(value byte) {
	p.element.Value.(chunk)[p.offset] = value
}

func (p *Pointer) Prev() error {
	return p.Rewind(1)
}

func (p *Pointer) Next() error {
	return p.Skip(1)
}

func (p *Pointer) Rewind(n int64) error {
	for {
		if n <= int64(p.offset) {
			p.offset -= int(n)
			p.address -= n
			return nil
		}
		prevElement := p.element.Prev()
		if prevElement == nil {
			return io.EOF
		}
		p.address -= int64(p.offset)
		n -= int64(p.offset)
		p.element = prevElement
		p.offset = len(p.element.Value.(chunk))
	}
}

func (p *Pointer) Skip(n int64) error {
	for {
		if int64(p.offset)+n < int64(len(p.element.Value.(chunk))) {
			p.offset += int(n)
			p.address += n
			return nil
		}
		nextElement := p.element.Next()
		if nextElement == nil {
			if err := p.buffer.tryFetchAndStore(); err != nil {
				// move cursor the end of the current block
				moveBytes := len(p.element.Value.(chunk)) - p.offset - 1
				p.offset += moveBytes
				p.address += int64(moveBytes)
				return err
			}
			nextElement = p.buffer.lines.Back()
		}
		moveBytes := len(p.element.Value.(chunk)) - p.offset
		n -= int64(moveBytes)
		p.element = nextElement
		p.offset = 0
		p.address += int64(moveBytes)
	}
}

func (p *Pointer) GoEndOfFile() {
	p.element = p.buffer.lines.Back()
	p.address = p.buffer.Len() - 1
	p.offset = len(p.element.Value.(chunk)) - 1
}

func (p *Pointer) Insert(value byte) {
	p.buffer.allsize++
	block := p.element.Value.(chunk)
	block = append(block, 0)
	copy(block[p.offset+1:], block[p.offset:])
	block[p.offset] = value
	p.element.Value = chunk(block)
}

func (p *Pointer) Append(value byte) {
	p.buffer.allsize++
	block := p.element.Value.(chunk)
	if len(block) == p.offset+1 {
		block = append(block, value)
	} else {
		block = append(block, 0)
		copy(block[p.offset+2:], block[p.offset+1:])
		block[p.offset+1] = value
	}
	p.element.Value = chunk(block)
}

func (p *Pointer) makeSpace(size int) chunk {
	block := p.element.Value.(chunk)
	if len(block) > size {
		block = append(block, block[len(block)-size:]...)
	} else {
		for i := 0; i < size; i++ {
			block = append(block, 0)
		}
	}
	p.element.Value = block
	p.buffer.allsize += int64(size)
	return block
}

func (p *Pointer) InsertSpace(size int) []byte {
	block := p.makeSpace(size)
	copy(block[p.offset+size:], block[p.offset:])
	return block[p.offset : p.offset+size]
}

func (p Pointer) AppendSpace(size int) []byte {
	block := p.makeSpace(size)
	copy(block[p.offset+size+1:], block[p.offset+1:])
	return block[p.offset+1 : p.offset+size+1]
}

const (
	RemoveSuccess = iota
	RemoveAll
	RemoveRefresh
)

func (p *Pointer) Remove() int {
	p.buffer.allsize--
	block := p.element.Value.(chunk)
	if len(block) <= 1 {
		defer p.buffer.lines.Remove(p.element)
		if next := p.element.Next(); next != nil {
			p.element = next
			p.offset = 0
			return RemoveSuccess
		} else if prev := p.element.Prev(); prev != nil {
			p.element = prev
			p.address--
			p.offset = len(p.element.Value.(chunk)) - 1
			return RemoveRefresh
		} else {
			return RemoveAll
		}
	}
	copy(block[p.offset:], block[p.offset+1:])
	block = block[:len(block)-1]
	p.element.Value = chunk(block)
	if p.offset >= len(block) {
		p.offset = len(block) - 1
		p.address--
	}
	return RemoveSuccess
}

func (p *Pointer) RemoveSpace(space int) {
	block := p.element.Value.(chunk)

	if space <= 0 {
		return
	} else if p.offset == 0 && space > len(block) {
		tmp := p.element.Next()
		if tmp != nil {
			p.buffer.lines.Remove(p.element)
			p.element = tmp
			p.buffer.allsize -= int64(len(block))
			p.RemoveSpace(space - len(block))
		}
		return
	} else if left := len(block) - p.offset; space > left {
		p.element.Value = chunk(block[:p.offset])
		tmp := p.element.Next()
		p.buffer.allsize -= int64(left)
		if tmp != nil {
			p.element = tmp
			p.offset = 0
			p.RemoveSpace(space - left)
		} else {
			p.offset--
		}
		return
	}
	copy(block[p.offset:], block[p.offset+space:])
	p.element.Value = chunk(block[:len(block)-space])
	p.buffer.allsize -= int64(space)
}
