package bine

import (
	"github.com/nyaosorg/go-windows-dbg"
)

func debug(args ...any) {
	if false {
		dbg.Println(args...)
	}
}
