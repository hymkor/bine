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
	"github.com/mattn/go-isatty"
	"github.com/mattn/go-runewidth"

	"github.com/nyaosorg/go-ttyadapter"
	"github.com/nyaosorg/go-ttyadapter/tty8pe"

	"github.com/hymkor/go-safewrite/perm"

	"github.com/hymkor/bine/internal/ansi"
	"github.com/hymkor/bine/internal/argf"
	"github.com/hymkor/bine/internal/encoding"
	"github.com/hymkor/bine/internal/large"
	"github.com/hymkor/bine/internal/nonblock"
)

const lineSize = 16

func (app *Application) makeHexOne(pointer *large.Pointer, out *strings.Builder) {
	cursorAddress := app.cursor.Address()
	m := app.editMode

	value := pointer.Value()
	var on, off string
	i := pointer.Address() % lineSize
	if ((i >> 2) & 1) == 0 {
		on = app.Scheme.Cell1[0]
		off = app.Scheme.Cell1[1]
	} else {
		on = app.Scheme.Cell2[0]
		off = app.Scheme.Cell2[1]
	}
	if pointer.Address() == cursorAddress {
		if _, ok := app.mark.(marking); ok {
			m.PrintByte(value, app.Scheme.Select[0], app.Scheme.Select[1], app.Scheme, out)
		} else {
			m.PrintByte(value, on, off, app.Scheme, out)
		}
	} else if app.mark.Contains(pointer.Address(), cursorAddress) {
		fmt.Fprintf(out, "%s%02X%s", app.Scheme.Select[0], value, app.Scheme.Select[1])
	} else {
		fmt.Fprintf(out, "%s%02X%s", on, value, off)
	}
}

// See. en.wikipedia.org/wiki/Unicode_control_characters#Control_pictures

func (app *Application) makeHexPart(pointer *large.Pointer, out *strings.Builder) bool {
	fmt.Fprintf(out, "%s%08X%s ", app.Scheme.Cell2[0], pointer.Address(), app.Scheme.Cell2[1])
	for i := 0; i < lineSize; i++ {
		app.makeHexOne(pointer, out)
		out.WriteByte(' ')
		if err := pointer.Next(); err != nil {
			for ; i < lineSize-1; i++ {
				out.WriteString("   ")
			}
			return false
		}
	}
	return true
}

var dontview = map[rune]rune{
	'\u000a': '\uFFEC', // for Line feed
	'\u000d': '\uFFE9', // <- halfwidth leftwards arrow; for carriage return
	'\t':     '\u21E5', // ->| rightwards arrow to bar; for tab
	'\u202e': '.',      // Right-to-Left override
	'\u202d': '.',      // Left-to-Right override
	'\u202c': '.',      // Pop Directional Formatting
}

func (app *Application) makeAsciiPart(pointer *large.Pointer, out *strings.Builder) bool {
	enc := app.encoding
	cursorAddress := app.cursor.Address()
	for i := 0; i < lineSize; {
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
			out.WriteString(app.Scheme.Cursor[0])
			out.WriteRune(c)
			out.WriteString(app.Scheme.Cursor[1])
		} else if app.mark.Contains(startAddress, cursorAddress) {
			out.WriteString(app.Scheme.Select[0])
			out.WriteRune(c)
			out.WriteString(app.Scheme.Select[1])
		} else {
			out.WriteString(app.Scheme.Cell1[0])
			out.WriteRune(c)
			out.WriteString(app.Scheme.Cell1[1])
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

func (app *Application) makeLineImage(pointer *large.Pointer) (string, bool) {
	cursorAddress := app.cursor.Address()
	var out strings.Builder
	off := ""
	if p := pointer.Address(); p <= cursorAddress && cursorAddress < p+lineSize {
		out.WriteString(ansi.UnderlineOn)
		off = ansi.UnderlineOff
	}

	asciiPointer := pointer.Clone()
	hasNextLine := app.makeHexPart(pointer, &out)
	app.makeAsciiPart(asciiPointer, &out)

	out.WriteString(ansi.EraseLine)
	out.WriteString(off)
	return out.String(), hasNextLine
}

func (app *Application) View() (int, error) {
	h := app.screenHeight - 1
	out := app.out
	count := 0

	cursor := app.window.Clone()
	for {
		line, cont := app.makeLineImage(cursor)

		if f := app.cache[count]; f != line {
			io.WriteString(out, line)
			app.cache[count] = line
		}
		if !cont {
			for i := count + 1; i < h; i++ {
				app.cache[i] = ""
			}
			return count, nil
		}
		if count+1 >= h {
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
	clipBoard    clipBoard
	dirty        bool
	savePath     string
	message      string
	cache        map[int]string
	Scheme       *Scheme
	encoding     encoding.Encoding
	undoFuncs    []func(app *Application) int64
	editMode     editModeType
	mark         markMode
	searchWord   string
	searchRevert bool
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
		savePath: defaultName,
		in:       in,
		out:      out,
		buffer:   large.NewBuffer(in),
		editMode: viewMode{},
		Scheme:   colorScheme,
		mark:     noMarking{},
	}
	if noColor := os.Getenv("NO_COLOR"); len(noColor) > 0 {
		this.Scheme = monoScheme
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
	io.WriteString(out, ansi.CursorOff)
	return this, nil
}

func (app *Application) Close() error {
	io.WriteString(app.out, ansi.CursorOn)
	io.WriteString(app.out, ansi.Reset)

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
	io.WriteString(app.out, app.Scheme.Status)
	if app.dirty {
		io.WriteString(app.out, "*")
	} else {
		io.WriteString(app.out, " ")
	}
	io.WriteString(app.out, app.editMode.String())
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

	io.WriteString(app.out, ansi.EraseScrnAfter)
	io.WriteString(app.out, ansi.Reset)
}

func (app *Application) shiftWindowToSeeCursorLine() {
	if app.cursor.Address() < app.window.Address() {
		app.window = app.cursor.Clone()
		if n := app.window.Address() % lineSize; n > 0 {
			app.window.Rewind(n)
		}
	} else if app.cursor.Address() >= app.window.Address()+lineSize*int64(app.dataHeight()) {
		app.window = app.cursor.Clone()
		app.window.Rewind(
			app.window.Address()%lineSize +
				int64(lineSize*(app.dataHeight()-1)))
	}
}

func Run(args []string) error {
	defer perm.RestoreAll()

	var out io.Writer
	if isatty.IsTerminal(os.Stdout.Fd()) {
		disable := colorable.EnableColorsStdout(nil)
		if disable != nil {
			defer disable()
		}
		out = colorable.NewColorableStdout()
	} else {
		out = colorable.NewColorableStderr()
	}
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
			io.WriteString(app.out, ansi.CursorOff)
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
			io.WriteString(app.out, app.Scheme.Status)
			io.WriteString(app.out, runewidth.Truncate(app.message, app.screenWidth-1, ""))
			io.WriteString(app.out, ansi.EraseScrnAfter)
			io.WriteString(app.out, ansi.Reset)
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
					if app.buffer.Len() >= int64(app.screenHeight*lineSize) {
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

		if err := app.editMode.Handle(ch, app); err != nil {
			return err
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
