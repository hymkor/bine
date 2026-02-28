package bine

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"

	"github.com/nyaosorg/go-ttyadapter"
	"github.com/nyaosorg/go-ttyadapter/tty8pe"

	"github.com/hymkor/bine/internal/argf"
	"github.com/hymkor/bine/internal/encoding"
	"github.com/hymkor/bine/internal/large"
	"github.com/hymkor/bine/internal/nonblock"
	"github.com/hymkor/go-safewrite/perm"
)

const LINE_SIZE = 16

const (
	_ANSI_CURSOR_OFF       = "\x1B[?25l"
	_ANSI_CURSOR_ON        = "\x1B[?25h"
	_ANSI_YELLOW           = "\x1B[0;33;1m"
	_ANSI_RESET            = "\x1B[0m"
	_ANSI_UNDERLINE_ON     = "\x1B[4m"
	_ANSI_UNDERLINE_OFF    = "\x1B[24m"
	_ANSI_ERASE_LINE       = "\x1B[0K"
	_ANSI_ERASE_SCRN_AFTER = "\x1B[0J"

	_CURSOR_COLOR_ON  = "\x1B[37;40;1;7m"
	_CURSOR_COLOR_OFF = "\x1B[27;22m"
	_CELL1_COLOR_ON   = "\x1B[37;40;22m"
	_CELL1_COLOR_OFF  = ""
	_CELL2_COLOR_ON   = "\x1B[37;40;1m"
	_CELL2_COLOR_OFF  = "\x1B[22m"
)

const (
	// for Line feed
	_ARROW_POINTING_DOWNWARDS_THEN_CURVING_LEFTWARDS = '\u2936'
	_DOWNWARDS_ARROW_WITH_CORNER_LEFTWARDS           = '\u21B5'
	_DOWNWARDS_ARROW_WITH_TIP_LEFTWARDS              = '\u21B2'
	_RETURN_SYMBOL                                   = '\u23CE'
	_SYMBOL_FOR_NEWLINE                              = '\u2424'
	_SYMBOL_FOR_LINE_FEED                            = '\u240A'
	_DOWNWARDS_ARROW                                 = '\u2193' // wide
	_HALFWIDTH_DOWNWARDS_ARROW                       = '\uFFEC'

	// for carriage return
	_SYMBOL_FOR_CARRIAGE_RETURN = '\u240D' // CR
	_LEFTWARDS_ARROW            = '\u2190' // wide
	_HALFWIDTH_LEFTWARDS_ARROW  = '\uFFE9' // <-

	// for tab
	_SYMBOL_FOR_HORIZONTAL_TABULATION        = '\u2409' // HT
	_RIGHTWARDS_ARROW_TO_BAR                 = '\u21E5' // ->|
	_RIGHTWARDS_TRIANGLE_HEADED_ARROW_TO_BAR = '\u2B72' // ->|
)

// See. en.wikipedia.org/wiki/Unicode_control_characters#Control_pictures

func makeHexPart(pointer *large.Pointer, cursorAddress int64, out *strings.Builder) bool {
	fmt.Fprintf(out, "%s%08X%s ", _CELL2_COLOR_ON, pointer.Address(), _CELL2_COLOR_OFF)
	var fieldSeperator string
	for i := 0; i < LINE_SIZE; i++ {
		var on, off string
		if pointer.Address() == cursorAddress {
			on = _CURSOR_COLOR_ON
			off = _CURSOR_COLOR_OFF
		} else if ((i >> 2) & 1) == 0 {
			on = _CELL1_COLOR_ON
			off = _CELL1_COLOR_OFF
		} else {
			on = _CELL2_COLOR_ON
			off = _CELL2_COLOR_OFF
		}
		fmt.Fprintf(out, "%s%s%02X%s", fieldSeperator, on, pointer.Value(), off)
		if err := pointer.Next(); err != nil {
			for ; i < LINE_SIZE-1; i++ {
				out.WriteString("   ")
			}
			return false
		}
		fieldSeperator = " "
	}
	return true
}

var dontview = map[rune]rune{
	'\u000a': _HALFWIDTH_DOWNWARDS_ARROW,
	'\u000d': _HALFWIDTH_LEFTWARDS_ARROW,
	'\t':     _RIGHTWARDS_ARROW_TO_BAR,
	'\u202e': '.', // Right-to-Left override
	'\u202d': '.', // Left-to-Right override
	'\u202c': '.', // Pop Directional Formatting
}

func makeAsciiPart(enc encoding.Encoding, pointer *large.Pointer, cursorAddress int64, out *strings.Builder) bool {
	for i := 0; i < LINE_SIZE; {
		var c rune
		startAddress := pointer.Address()
		b := pointer.Value()

		var runeBuffer [utf8.UTFMax]byte
		savePointer := pointer.Clone()

		length := enc.Count(b, pointer.Address())
		runeBuffer[0] = b
		readCount := 1
		for j := 1; j < length && pointer.Next() == nil; j++ {
			runeBuffer[j] = pointer.Value()
			readCount++
		}
		c = enc.Decode(runeBuffer[:readCount])

		if c == utf8.RuneError {
			c = '.'
			length = 1
			pointer = savePointer
		}

		if _c, ok := dontview[c]; ok {
			c = _c
		} else if unicode.IsControl(c) {
			c = '.'
		}

		if startAddress <= cursorAddress && cursorAddress <= pointer.Address() {
			out.WriteString(_CURSOR_COLOR_ON)
			out.WriteRune(c)
			out.WriteString(_CURSOR_COLOR_OFF)
		} else {
			out.WriteString(_CELL1_COLOR_ON)
			out.WriteRune(c)
			out.WriteString(_CELL1_COLOR_OFF)
		}
		if length == 3 {
			out.WriteByte(' ')
		} else if length == 4 {
			out.WriteString("  ")
		}
		i += length
		if pointer.Next() != nil {
			return false
		}
	}
	return true
}

func makeLineImage(enc encoding.Encoding, pointer *large.Pointer, cursorAddress int64) (string, bool) {
	var out strings.Builder
	off := ""
	if p := pointer.Address(); p <= cursorAddress && cursorAddress < p+LINE_SIZE {
		out.WriteString(_ANSI_UNDERLINE_ON)
		off = _ANSI_UNDERLINE_OFF
	}

	asciiPointer := *pointer
	hasNextLine := makeHexPart(pointer, cursorAddress, &out)
	out.WriteByte(' ')
	makeAsciiPart(enc, &asciiPointer, cursorAddress, &out)

	out.WriteString(_ANSI_ERASE_LINE)
	out.WriteString(off)
	return out.String(), hasNextLine
}

func (app *Application) View() (int, error) {
	h := app.screenHeight - 1
	out := app.out
	count := 0

	cursor := app.window.Clone()
	cursorAddress := app.cursor.Address()
	for {
		line, cont := makeLineImage(app.encoding, cursor, cursorAddress)

		if f := app.cache[count]; f != line {
			io.WriteString(out, line)
			app.cache[count] = line
		}
		if !cont || count+1 >= h {
			return count, nil
		}
		count++
		io.WriteString(out, "\r\n") // "\r" is for Linux and go-tty
	}
}

type Application struct {
	tty1         ttyadapter.Tty
	in           io.Reader
	out          io.Writer
	screenWidth  int
	screenHeight int
	cursor       *large.Pointer
	window       *large.Pointer
	buffer       *large.Buffer
	clipBoard    *Clip
	dirty        bool
	savePath     string
	message      string
	cache        map[int]string
	encoding     encoding.Encoding
	undoFuncs    []func(app *Application)
}

func (app *Application) dataHeight() int {
	return app.screenHeight - 1
}

func detectEncoding(p *large.Pointer) encoding.Encoding {
	p = p.Clone()
	byte1 := p.Value()
	if p.Next() == nil {
		byte2 := p.Value()
		if byte1 == 0xFF && byte2 == 0xFE {
			return encoding.UTF16LE()
		}
		if byte1 == 0xFE && byte2 == 0xFF {
			return encoding.UTF16BE()
		}
	}
	return encoding.UTF8Encoding{}
}

func NewApplication(tty ttyadapter.Tty, in io.Reader, out io.Writer, defaultName string) (*Application, error) {
	this := &Application{
		savePath:  defaultName,
		in:        in,
		out:       out,
		buffer:    large.NewBuffer(in),
		clipBoard: NewClip(),
	}
	this.window = large.NewPointer(this.buffer)
	if this.window == nil {
		return nil, io.EOF
	}
	this.cursor = large.NewPointer(this.buffer)
	if this.cursor == nil {
		return nil, io.EOF
	}
	this.encoding = detectEncoding(this.cursor)

	this.tty1 = tty
	err := this.tty1.Open(nil)
	if err != nil {
		return nil, err
	}
	io.WriteString(out, _ANSI_CURSOR_OFF)
	return this, nil
}

func (app *Application) Close() error {
	io.WriteString(app.out, _ANSI_CURSOR_ON)
	io.WriteString(app.out, _ANSI_RESET)

	if app.tty1 != nil {
		app.tty1.Close()
	}
	return nil
}

var unicodeName = map[rune]string{
	'\uFEFF': "ByteOrderMark",
	'\uFFFE': "Reverted ByteOrderMark",
	'\u200D': "ZeroWidthJoin",
	'\u202E': "RightToLeftOverride",
	'\u202D': "LeftToRightOverride",
}

func (app *Application) printDefaultStatusBar() {
	io.WriteString(app.out, _ANSI_YELLOW)
	if app.dirty {
		io.WriteString(app.out, "*")
	} else {
		io.WriteString(app.out, " ")
	}
	fmt.Fprintf(app.out, "[%s]", app.encoding.ModeString())

	fmt.Fprintf(app.out, "%4[1]d='\\x%02[1]X'", app.cursor.Value())

	theRune, thePosInRune, theLenOfRune := app.encoding.RuneOver(app.cursor.Clone())
	if theRune != utf8.RuneError {
		fmt.Fprintf(app.out, "(%d/%d:U+%04X",
			thePosInRune+1,
			theLenOfRune,
			theRune)
		if name, ok := unicodeName[theRune]; ok {
			fmt.Fprintf(app.out, ":%s", name)
		}
		app.out.Write([]byte{')'})
	} else {
		fmt.Fprintf(app.out, "(bin:'\\x%02X')", app.cursor.Value())
	}

	fmt.Fprintf(app.out,
		" @ %[1]d=0x%[1]X/%[2]d=0x%[2]X",
		app.cursor.Address(),
		app.buffer.Len())

	io.WriteString(app.out, _ANSI_ERASE_SCRN_AFTER)
	io.WriteString(app.out, _ANSI_RESET)
}

func (app *Application) shiftWindowToSeeCursorLine() {
	if app.cursor.Address() < app.window.Address() {
		app.window = app.cursor.Clone()
		if n := app.window.Address() % LINE_SIZE; n > 0 {
			app.window.Rewind(n)
		}
	} else if app.cursor.Address() >= app.window.Address()+LINE_SIZE*int64(app.dataHeight()) {
		app.window = app.cursor.Clone()
		app.window.Rewind(
			app.window.Address()%LINE_SIZE +
				int64(LINE_SIZE*(app.dataHeight()-1)))
	}
}

func Run(args []string) error {
	defer perm.RestoreAll()

	disable := colorable.EnableColorsStdout(nil)
	if disable != nil {
		defer disable()
	}
	out := colorable.NewColorableStdout()

	in, err := argf.New(args)
	if err != nil {
		return err
	}
	defer in.Close()

	savePath := ""
	if len(args) > 0 {
		savePath, err = filepath.Abs(args[0])
		if err != nil {
			return err
		}
	}

	app, err := NewApplication(&tty8pe.Tty{}, in, out, savePath)
	if err != nil {
		return err
	}
	defer app.Close()

	// nonblock runs data reading in the background while waiting for key input.
	// If Fetch is called directly, it may run concurrently and also bypass
	// buffered data already queued in keyWorker, which can break data order.
	//
	// Therefore, all reads are centralized in keyWorker, and this goroutine
	// accesses data only via keyWorker.Fetch / TryFetch.
	keyWorker := nonblock.New(app.tty1.GetKey, app.buffer.Fetch)
	defer keyWorker.Close()
	app.buffer.Fetch = keyWorker.Fetch
	app.buffer.TryFetch = func() ([]byte, error) {
		return keyWorker.TryFetch(time.Second / 100)
	}

	var lastWidth, lastHeight int
	autoRepaint := true
	for {
		app.screenWidth, app.screenHeight, err = app.tty1.Size()
		if err != nil {
			return err
		}
		if lastWidth != app.screenWidth || lastHeight != app.screenHeight {
			app.cache = map[int]string{}
			lastWidth = app.screenWidth
			lastHeight = app.screenHeight
			io.WriteString(app.out, _ANSI_CURSOR_OFF)
		}
		lf, err := app.View()
		if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
			return err
		}
		if app.buffer.Len() <= 0 {
			return nil
		}
		io.WriteString(app.out, "\r\n") // \r is for Linux & go-tty
		lf++
		if app.message != "" {
			io.WriteString(app.out, _ANSI_YELLOW)
			io.WriteString(app.out, runewidth.Truncate(app.message, app.screenWidth-1, ""))
			io.WriteString(app.out, _ANSI_ERASE_SCRN_AFTER)
			io.WriteString(app.out, _ANSI_RESET)
		} else {
			app.printDefaultStatusBar()
		}

		const interval = 10
		displayUpdateTime := time.Now().Add(time.Second / interval)

		ch, err := keyWorker.GetOr(func(data []byte, err error) (cont bool) {
			cont = app.buffer.Store(data, err)
			if app.message != "" {
				return
			}
			if err == io.EOF || time.Now().After(displayUpdateTime) {
				app.out.Write([]byte{'\r'})
				if autoRepaint {
					if lf > 0 {
						fmt.Fprintf(app.out, "\x1B[%dA", lf)
					}
					lf, _ = app.View()
					io.WriteString(app.out, "\r\n") // \r is for Linux & go-tty
					lf++
					if app.buffer.Len() >= int64(app.screenHeight*LINE_SIZE) {
						autoRepaint = false
					}
				}
				app.printDefaultStatusBar()
				displayUpdateTime = time.Now().Add(time.Second / interval)
			}
			return
		})
		if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
			return err
		}
		app.message = ""
		if hander, ok := jumpTable[ch]; ok {
			if err := hander(app); err != nil {
				return err
			}
		}
		if app.buffer.Len() <= 0 {
			return nil
		}

		app.shiftWindowToSeeCursorLine()

		if lf > 0 {
			fmt.Fprintf(app.out, "\r\x1B[%dA", lf)
		} else {
			io.WriteString(app.out, "\r")
		}
	}
}
