package bine

import (
	"fmt"
	"io"
	"strings"
)

type editModeType interface {
	Handle(string, *Application) error
	String() string
	PrintByte(value byte, on, off string, scheme *Scheme, w io.Writer)
	Reset() editModeType
	Next() (editModeType, bool)
	Prev() (editModeType, bool)
}

type viewMode struct{}

func (viewMode) Handle(ch string, app *Application) error {
	if hander, ok := jumpTable[ch]; ok {
		return hander(app)
	}
	return nil
}

func (viewMode) String() string {
	return "Command:"
}

func (d viewMode) PrintByte(value byte, on, off string, scheme *Scheme, w io.Writer) {
	fmt.Fprintf(w, "%s%02X%s", scheme.Cursor[0], value, scheme.Cursor[1])
}

func (viewMode) Reset() editModeType {
	return viewMode{}
}

func (viewMode) Next() (_ editModeType, moveNextByte bool) {
	return viewMode{}, true
}

func (viewMode) Prev() (_ editModeType, movePrevByte bool) {
	return viewMode{}, true
}

type directMode struct {
	Lower bool
}

func (directMode) Handle(ch string, app *Application) error {
	if index := strings.Index("0123456789abcdef", ch); index >= 0 {
		return keyFuncReplaceInline(app, byte(index))
	}
	return viewMode{}.Handle(ch, app)
}

func (directMode) String() string {
	return "Direct:"
}

func (d directMode) PrintByte(value byte, on, off string, scheme *Scheme, w io.Writer) {
	upper := (value >> 4) & 15
	lower := value & 15
	if d.Lower {
		fmt.Fprintf(w, "%s%1X%s%s%1X%s",
			on,
			upper,
			off,
			scheme.Cursor[0],
			lower,
			scheme.Cursor[1])
	} else {
		fmt.Fprintf(w, "%s%1X%s%s%1X%s",
			scheme.Cursor[0],
			upper,
			scheme.Cursor[1],
			on,
			lower,
			off)
	}
}

func (directMode) Reset() editModeType {
	return directMode{}
}

func (d directMode) Next() (_ editModeType, moveNextByte bool) {
	if d.Lower {
		return directMode{Lower: false}, true
	}
	return directMode{Lower: true}, false
}

func (d directMode) Prev() (_ editModeType, movePrevByte bool) {
	if d.Lower {
		return directMode{Lower: false}, false
	}
	return directMode{Lower: true}, true
}
