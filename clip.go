package bine

type Clip struct {
	data []byte
}

func NewClip() *Clip {
	return &Clip{data: make([]byte, 0, 100)}
}

func (c *Clip) Push(n byte) {
	c.data = append(c.data, n)
}

func (c *Clip) Pop() byte {
	var newByte byte
	if len(c.data) > 0 {
		tail := len(c.data) - 1
		newByte = c.data[tail]
		c.data = c.data[:tail]
	}
	return newByte
}

func (c *Clip) Len() int {
	return len(c.data)
}
