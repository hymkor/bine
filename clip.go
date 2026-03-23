package bine

type clipBoard struct {
	data [][]byte
}

func (c *clipBoard) Push(n []byte) {
	c.data = append(c.data, n)
}

func (c *clipBoard) Pop() []byte {
	var newByte []byte
	if len(c.data) > 0 {
		tail := len(c.data) - 1
		newByte = c.data[tail]
		c.data = c.data[:tail]
	}
	return newByte
}

func (c *clipBoard) Len() int {
	return len(c.data)
}
