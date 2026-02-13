package bine

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/hymkor/binview/internal/encoding"
	"github.com/hymkor/binview/internal/large"
)

var (
	rxUnicodeCodePoint = regexp.MustCompile(`^\s*[uU]\+([0-9A-Fa-f]+)`)
	rxByte             = regexp.MustCompile(`^\s*0x([0-9A-Fa-f]+)`)
	rxDigit            = regexp.MustCompile(`^\s*([0-9]+)`)
	rxString           = regexp.MustCompile(`^\s*[uU]?"([^"]+)"`)
)

func evalExpression(exp string, enc encoding.Encoding) ([]byte, error) {
	bytes := make([]byte, 0)
	for len(exp) > 0 {
		if m := rxUnicodeCodePoint.FindStringSubmatch(exp); m != nil {
			exp = exp[len(m[0]):]
			theRune, err := strconv.ParseUint(m[1], 16, 32)
			if err != nil {
				return nil, err
			}
			if bin, err := enc.EncodeFromString(string(rune(theRune))); err == nil {
				bytes = append(bytes, bin...)
			}
		} else if m := rxByte.FindStringSubmatch(exp); m != nil {
			exp = exp[len(m[0]):]
			theByte, err := strconv.ParseUint(m[1], 16, 16)
			if err != nil {
				return nil, err
			}
			bytes = append(bytes, byte(theByte))
		} else if m := rxDigit.FindStringSubmatch(exp); m != nil {
			exp = exp[len(m[0]):]
			value, err := strconv.ParseUint(m[1], 10, 16)
			if err != nil {
				return nil, err
			}
			bytes = append(bytes, byte(value))
		} else if m := rxString.FindStringSubmatch(exp); m != nil {
			exp = exp[len(m[0]):]
			if bin, err := enc.EncodeFromString(m[1]); err == nil {
				bytes = append(bytes, bin...)
			}
		} else {
			return bytes, fmt.Errorf("`%s` are ignored", exp)
		}
	}
	return bytes, nil
}

func insertExp(exp string, enc encoding.Encoding, ptr *large.Pointer) (int, error) {
	bytes, err := evalExpression(exp, enc)
	if err != nil {
		return 0, err
	}
	space := ptr.InsertSpace(len(bytes))
	copy(space, bytes)
	return len(bytes), nil
}

func (app *Application) InsertExp(exp string) error {
	undoAddress := app.cursor.Address()
	orgDirty := app.dirty
	size, err := insertExp(exp, app.encoding, app.cursor)
	if err == nil {
		undo := func(app *Application) {
			p := large.NewPointerAt(undoAddress, app.buffer)
			p.RemoveSpace(size)
			app.dirty = orgDirty
		}
		app.undoFuncs = append(app.undoFuncs, undo)
		app.dirty = true
	}
	return err
}

func appendExp(exp string, enc encoding.Encoding, ptr *large.Pointer) (int, error) {
	bytes, err := evalExpression(exp, enc)
	if err != nil {
		return 0, err
	}
	space := ptr.AppendSpace(len(bytes))
	copy(space, bytes)
	return len(bytes), nil
}

func (app *Application) AppendExp(exp string) error {
	size, err := appendExp(exp, app.encoding, app.cursor)
	undoAddress := app.cursor.Address() + 1
	orgDirty := app.dirty
	if err == nil {
		undo := func(app *Application) {
			p := large.NewPointerAt(undoAddress, app.buffer)
			p.RemoveSpace(size)
			app.dirty = orgDirty
		}
		app.undoFuncs = append(app.undoFuncs, undo)
		app.dirty = true
	}
	return err
}
