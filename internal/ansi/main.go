package ansi

const (
	CursorOff      = "\x1B[?25l"
	CursorOn       = "\x1B[?25h"
	Reset          = "\x1B[0m"
	UnderlineOn    = "\x1B[4m"
	UnderlineOff   = "\x1B[24m"
	EraseLine      = "\x1B[0K"
	EraseScrnAfter = "\x1B[0J"
)
