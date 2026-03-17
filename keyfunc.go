package bine

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mattn/go-isatty"

	"github.com/nyaosorg/go-inline-animation"
	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/simplehistory"

	"github.com/hymkor/bine/internal/encoding"
	"github.com/hymkor/bine/internal/large"
	"github.com/hymkor/go-safewrite"
	"github.com/hymkor/go-safewrite/perm"
)

const (
	_KEY_CTRL_A = "\x01"
	_KEY_CTRL_B = "\x02"
	_KEY_CTRL_E = "\x05"
	_KEY_CTRL_F = "\x06"
	_KEY_CTRL_L = "\x0C"
	_KEY_CTRL_N = "\x0E"
	_KEY_CTRL_P = "\x10"
	_KEY_DOWN   = "\x1B[B"
	_KEY_ESC    = "\x1B"
	_KEY_LEFT   = "\x1B[D"
	_KEY_RIGHT  = "\x1B[C"
	_KEY_UP     = "\x1B[A"
	_KEY_F2     = "\x1B[OQ"
	_KEY_DEL    = "\x1B[3~"
	_KEY_ALT_A  = "\x1Ba"
	_KEY_ALT_U  = "\x1Bu"
	_KEY_ALT_L  = "\x1Bl"
	_KEY_ALT_B  = "\x1Bb"
)

// keyFuncNext moves the cursor to the the next 16-bytes block.
func keyFuncNext(this *Application) error {
	if err := this.cursor.Skip(LINE_SIZE); err != nil {
		if !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrDeadlineExceeded) {
			return err
		}
	}
	return nil
}

// keyFuncBackword move the cursor to the previous byte.
func keyFuncBackword(app *Application) error {
	var ok bool
	if app.editMode, ok = app.editMode.Prev(); ok {
		app.cursor.Prev()
	}
	return nil
}

// keyFuncPrevious moves the cursor the the previous 16-bytes block.
func keyFuncPrevious(this *Application) error {
	this.cursor.Rewind(LINE_SIZE)
	return nil
}

func keyFuncQuit(this *Application) error {
	if this.dirty {
		ch, err := ask(this.tty1, this.out, `Quit: Save changes ? ["y": save, "n": quit without saving, other: cancel]`)
		if err != nil {
			return err
		}
		if ch == "y" || ch == "Y" {
			newfname, err := writeFile(this)
			if err != nil {
				this.message = err.Error()
				return nil
			}
			this.dirty = false
			this.savePath = newfname
		} else if ch != "n" && ch != "N" {
			return nil
		}
	}
	io.WriteString(this.out, "\n")
	return io.EOF
}

// keyFuncForward moves the cursor to the next one byte.
func keyFuncForward(app *Application) error {
	var ok bool
	if app.editMode, ok = app.editMode.Next(); ok {
		app.cursor.Next()
	}
	return nil
}

// keyFuncGoBeginOfLine move the cursor the the top of the 16bytes-block.
func keyFuncGoBeginOfLine(app *Application) error {
	n := app.cursor.Address() % LINE_SIZE
	if n > 0 {
		app.cursor.Rewind(n)
	}
	app.editMode = app.editMode.Reset()
	return nil
}

// keyFuncGoEndOfLine move the cursor to the end of the current 16 byte block.
func keyFuncGoEndOfLine(app *Application) error {
	n := LINE_SIZE - app.cursor.Address()%LINE_SIZE - 1
	if n > 0 {
		app.cursor.Skip(n)
	}
	app.editMode = app.editMode.Reset()
	return nil
}

func keyFuncGoBeginOfFile(app *Application) error {
	app.cursor = large.NewPointer(app.buffer)
	app.window = large.NewPointer(app.buffer)
	app.editMode = app.editMode.Reset()
	return nil
}

// keyFuncGoEndOfFile moves the cursor to the end of the file.
func keyFuncGoEndOfFile(app *Application) error {
	app.cursor.GoEndOfFile()
	app.editMode = app.editMode.Reset()
	return nil
}

// keyFuncPasteAfter inserts the top byte of clipboard after the cursor.
func keyFuncPasteAfter(this *Application) error {
	if this.clipBoard.Len() <= 0 {
		return nil
	}
	newBytes := this.clipBoard.Pop()
	orgAddress := this.cursor.Address() + 1
	orgDirty := this.dirty
	undo := func(app *Application) (rv int64) {
		p := large.NewPointerAt(orgAddress, app.buffer)
		rv = p.Address()
		p.RemoveSpace(len(newBytes))
		this.dirty = orgDirty
		return
	}
	space := this.cursor.AppendSpace(len(newBytes))
	copy(space, newBytes)
	this.undoFuncs = append(this.undoFuncs, undo)
	this.dirty = true
	return nil
}

// keyFuncPasteBefore inserts the top of the clipboard at the cursor.
func keyFuncPasteBefore(this *Application) error {
	if this.clipBoard.Len() <= 0 {
		return nil
	}
	newBytes := this.clipBoard.Pop()
	orgAddress := this.cursor.Address()
	orgDirty := this.dirty
	undo := func(app *Application) (rv int64) {
		p := large.NewPointerAt(orgAddress, app.buffer)
		rv = p.Address()
		p.RemoveSpace(len(newBytes))
		this.dirty = orgDirty
		return
	}
	space := this.cursor.InsertSpace(len(newBytes))
	copy(space, newBytes)
	this.undoFuncs = append(this.undoFuncs, undo)
	this.dirty = true
	return nil
}

func fromTo(a, b int64) (from, to int64) {
	if a < b {
		return a, b + 1
	} else {
		return b, a + 1
	}
}

func dupFromPointer(start, until int64, buffer *large.Buffer) (b []byte) {
	b = make([]byte, 0, until-start)
	p := buffer.NewPointerAt(start)
	for p.Address() < until {
		b = append(b, p.Value())
		if p.Next() != nil {
			return
		}
	}
	return
}

// keyFuncRemoveByte removes the byte where cursor exists.
func keyFuncRemoveByte(this *Application) error {
	orgValue := this.cursor.Value()
	address := this.cursor.Address()
	orgDirty := this.dirty
	undo := func(app *Application) int64 {
		p := large.NewPointerAt(address, app.buffer)
		p.Insert(orgValue)
		app.dirty = orgDirty
		return p.Address()
	}
	this.undoFuncs = append(this.undoFuncs, undo)
	this.dirty = true
	this.clipBoard.Push([]byte{this.cursor.Value()})
	if this.cursor.Remove() == large.RemoveAll {
		return io.EOF
	}
	return nil
}

func keyFuncYank(app *Application) error {
	if app.mark < 0 {
		app.clipBoard.Push([]byte{app.cursor.Value()})
		return nil
	}
	from, to := fromTo(app.mark, app.cursor.Address())
	if to-from > 0x80000000 {
		return errors.New("too long area")
	}
	app.mark = -1
	app.clipBoard.Push(dupFromPointer(from, to, app.buffer))
	return nil
}

func keyFuncDelete(app *Application) error {
	var orgValue []byte
	var from, to int64

	if app.mark < 0 {
		from = app.cursor.Address()
		to = from
		orgValue = []byte{app.cursor.Value()}
		app.cursor.Remove()
	} else {
		from, to = fromTo(app.mark, app.cursor.Address())
		if to-from > 0x80000000 {
			return errors.New("too long area")
		}
		app.mark = -1
		orgValue = dupFromPointer(from, to, app.buffer)
		if from <= 0 {
			app.cursor = app.buffer.NewPointer()
		} else {
			app.cursor = app.buffer.NewPointerAt(from)
		}
		app.cursor.RemoveSpace(int(to - from))
	}

	orgDirty := app.dirty
	undo := func(app *Application) int64 {
		p := app.buffer.NewPointerAt(from)
		space := p.InsertSpace(int(to - from))
		copy(space, orgValue)
		app.dirty = orgDirty
		return p.Address()
	}
	app.undoFuncs = append(app.undoFuncs, undo)
	app.dirty = true
	app.clipBoard.Push(orgValue)

	if app.buffer.Len() <= 0 {
		return io.EOF
	}
	app.cursor = app.buffer.NewPointerAt(from)
	return nil
}

func getlineOr(out io.Writer, prompt string, defaultString string, history readline.IHistory) (string, error) {
	return getline(out, prompt, defaultString, history)
}

var fnameHistory = simplehistory.New()

func writeFile(app *Application) (string, error) {
	buffer := app.buffer
	tty1 := app.tty1
	out := app.out
	fname := app.savePath

	var err error
	fname, err = getlineOr(out, "write to>", fname, fnameHistory)
	if err != nil {
		return "", err
	}
	if fname == "-" {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			return "", errors.New("stdout is a terminal. Refusing to write binary data")
		}
		_, err := buffer.WriteTo(os.Stdout)
		return "-", err
	}
	prompt := func(info *safewrite.Info) bool {
		if info.Status != safewrite.NONE {
			return true
		}
		if info.ReadOnly() {
			return yesNo(tty1, out, "Overwrite READONLY file \""+info.Name+"\" [y/n] ?")
		}
		return yesNo(tty1, out, "Overwrite as \""+info.Name+"\" [y/n] ?")
	}

	fd, err := safewrite.Open(fname, prompt)
	if err != nil {
		return "", err
	}
	end := animation.Dots.Progress(out)
	defer end()

	_, err1 := buffer.WriteTo(fd)
	err2 := fd.Close()
	if err1 != nil {
		return "", err1
	}
	if err2 != nil {
		var e *safewrite.BackupError
		if errors.As(err2, &e) {
			return "",
				fmt.Errorf("Failed to backup %q to %q (tmp: %q)",
					filepath.Base(e.Target),
					filepath.Base(e.Backup),
					filepath.Base(e.Tmp))
		}
		var re *safewrite.ReplaceError
		if errors.As(err2, &re) {
			return "",
				fmt.Errorf("Failed to replace %q to %q",
					filepath.Base(re.Tmp),
					filepath.Base(re.Target))
		}
		return "", err2
	}
	fnameHistory.Add(fname)
	perm.Track(fd)
	return fname, nil
}

func keyFuncWriteFile(this *Application) error {
	newfname, err := writeFile(this)
	if err != nil {
		this.message = err.Error()
	} else {
		this.dirty = false
		this.savePath = newfname
	}
	return nil
}

var byteHistory = simplehistory.New()

func keyFuncReplaceByte(this *Application) error {
	bytes, err := getlineOr(this.out, "replace>",
		fmt.Sprintf("0x%02X", this.cursor.Value()),
		byteHistory)
	if err != nil {
		this.message = err.Error()
		return nil
	}
	if n, err := strconv.ParseUint(bytes, 0, 8); err == nil {
		address := this.cursor.Address()
		orgValue := this.cursor.Value()
		orgDirty := this.dirty
		undo := func(app *Application) int64 {
			p := large.NewPointerAt(address, app.buffer)
			p.SetValue(orgValue)
			app.dirty = orgDirty
			return p.Address()
		}
		this.undoFuncs = append(this.undoFuncs, undo)
		this.cursor.SetValue(byte(n))
		this.dirty = true
		byteHistory.Add(bytes)
	} else {
		this.message = err.Error()
	}
	return nil
}

func keyFuncRepaint(this *Application) error {
	this.cache = map[int]string{}
	return nil
}

func gotoAddress(app *Application, address int64) error {
	prevousAddress := app.cursor.Address()
	if address > prevousAddress {
		app.cursor.Skip(address - prevousAddress)
	} else if address < prevousAddress {
		app.cursor.Rewind(prevousAddress - address)
	}
	return nil
}

var addressHistory = simplehistory.New()

func keyFuncGoTo(app *Application) error {
	addressStr, err := getlineOr(app.out, "Goto Offset>", "0x", addressHistory)
	if err != nil {
		app.message = err.Error()
		return nil
	}
	address, err := strconv.ParseInt(addressStr, 0, 64)
	if err != nil {
		app.message = err.Error()
		return nil
	}
	addressHistory.Add(addressStr)
	return gotoAddress(app, address)
}

func keyFuncDbcsMode(app *Application) error {
	app.encoding = encoding.DBCSEncoding{}
	return nil
}

func keyFuncUtf8Mode(app *Application) error {
	app.encoding = encoding.UTF8Encoding{}
	return nil
}

func keyFuncUtf16LeMode(app *Application) error {
	app.encoding = encoding.UTF16LE()
	return nil
}

func keyFuncUtf16BeMode(app *Application) error {
	app.encoding = encoding.UTF16BE()
	return nil
}

var expHistory = simplehistory.New()

func readExpression(app *Application, prompt string) (string, error) {
	exp, err := getlineOr(app.out, prompt, "0x00", expHistory)
	if err != nil {
		return "", err
	}
	expHistory.Add(exp)
	return exp, err
}

func keyFuncInsertExp(app *Application) error {
	exp, err := readExpression(app, "insert>")
	if err != nil {
		app.message = err.Error()
		return nil
	}
	err = app.InsertExp(exp)
	if err != nil {
		app.message = err.Error()
	}
	return nil
}

func keyFuncInsertZero(app *Application) error {
	app.InsertZero()
	app.editMode = app.editMode.Reset()
	return nil
}

func keyFuncAppendExp(app *Application) error {
	exp, err := readExpression(app, "append>")
	if err != nil {
		app.message = err.Error()
		return nil
	}
	err = app.AppendExp(exp)
	if err != nil {
		app.message = err.Error()
	}
	return keyFuncForward(app)
}

func keyFuncAppendZero(app *Application) error {
	app.AppendZero()
	app.cursor.Next()
	app.editMode = app.editMode.Reset()
	return nil
}

func keyFuncUndo(app *Application) error {
	if len(app.undoFuncs) <= 0 {
		return nil
	}
	addressSave := app.cursor.Address()

	undoFunc1 := app.undoFuncs[len(app.undoFuncs)-1]
	app.undoFuncs = app.undoFuncs[:len(app.undoFuncs)-1]
	undoneAddress := undoFunc1(app)

	app.cursor = large.NewPointer(app.buffer)
	if undoneAddress >= 0 {
		app.cursor.Skip(undoneAddress)
	} else {
		app.cursor.Skip(addressSave)
	}
	return nil
}

func keyFuncReplaceInline(app *Application, n byte) error {
	address := app.cursor.Address()
	orgValue := app.cursor.Value()
	orgDirty := app.dirty

	if app.editMode.(directMode).Lower {
		app.cursor.SetValue((orgValue &^ 0x0F) | (n & 0xF))
	} else {
		app.cursor.SetValue((orgValue &^ 0xF0) | (n << 4))
	}
	var ok bool
	if app.editMode, ok = app.editMode.Next(); ok {
		app.cursor.Next()
	}
	undo := func(ap *Application) int64 {
		p := large.NewPointerAt(address, ap.buffer)
		p.SetValue(orgValue)
		ap.dirty = orgDirty
		return p.Address()
	}
	app.undoFuncs = append(app.undoFuncs, undo)
	app.dirty = true
	return nil
}

func keyFuncChangeMode(app *Application) error {
	if _, ok := app.editMode.(directMode); ok {
		app.editMode = viewMode{}
	} else {
		app.editMode = directMode{}
	}
	return nil
}

func keyFuncMarking(app *Application) error {
	if app.mark > 0 {
		app.mark = -1
	} else {
		app.mark = app.cursor.Address()
	}
	return nil
}

func searchBytes(app *Application, exp []byte, walk func(*large.Pointer) error) error {
	p := app.cursor.Clone()
	for {
		if err := walk(p); err != nil {
			if err == io.EOF {
				app.message = "not found"
			} else {
				app.message = err.Error()
			}
			return nil
		}
		if p.Value() == exp[0] {
			q := p.Clone()
			i := 1
			for {
				if i >= len(exp) {
					app.cursor = p
					return nil
				}
				if err := q.Next(); err != nil {
					break
				}
				if q.Value() != exp[i] {
					break
				}
				i++
			}
		}
	}
}

func walkForward(p *large.Pointer) error { return p.Next() }

func walkBackward(p *large.Pointer) error { return p.Prev() }

func keyFuncSearchForward(app *Application) error {
	expStr := app.searchWord
	if expStr == "" {
		expStr = "0x"
	}
	var err error
	expStr, err = getlineOr(app.out, "search forward>", expStr, nil)
	if err != nil {
		app.message = err.Error()
		return nil
	}
	exp, err := evalExpression(expStr, app.encoding)
	if err != nil {
		return err
	}
	if len(exp) <= 0 {
		return nil
	}
	app.searchWord = expStr
	app.searchRevert = false
	return searchBytes(app, exp, func(p *large.Pointer) error { return p.Next() })
}

func keyFuncSearchForwardNext(app *Application) error {
	exp, err := evalExpression(app.searchWord, app.encoding)
	if err != nil {
		return err
	}
	f := walkForward
	if app.searchRevert {
		f = walkBackward
	}
	return searchBytes(app, exp, f)
}

func keyFuncSearchBackward(app *Application) error {
	expStr := app.searchWord
	if expStr == "" {
		expStr = "0x"
	}
	var err error
	expStr, err = getlineOr(app.out, "search backward>", expStr, nil)
	if err != nil {
		app.message = err.Error()
		return nil
	}
	exp, err := evalExpression(expStr, app.encoding)
	if err != nil {
		return err
	}
	if len(exp) <= 0 {
		return nil
	}
	app.searchWord = expStr
	app.searchRevert = true
	return searchBytes(app, exp, func(p *large.Pointer) error { return p.Prev() })
}

func keyFuncSearchBackwardNext(app *Application) error {
	exp, err := evalExpression(app.searchWord, app.encoding)
	if err != nil {
		return err
	}
	f := walkBackward
	if app.searchRevert {
		f = walkForward
	}
	return searchBytes(app, exp, f)
}

var jumpTable = map[string]func(this *Application) error{
	"u":         keyFuncUndo,
	"i":         keyFuncInsertExp,
	"I":         keyFuncInsertZero,
	"a":         keyFuncAppendExp,
	"A":         keyFuncAppendZero,
	_KEY_ALT_A:  keyFuncDbcsMode,
	_KEY_ALT_U:  keyFuncUtf8Mode,
	_KEY_ALT_L:  keyFuncUtf16LeMode,
	_KEY_ALT_B:  keyFuncUtf16BeMode,
	"&":         keyFuncGoTo,
	"q":         keyFuncQuit,
	"j":         keyFuncNext,
	_KEY_DOWN:   keyFuncNext,
	_KEY_CTRL_N: keyFuncNext,
	"h":         keyFuncBackword,
	"\b":        keyFuncBackword,
	_KEY_LEFT:   keyFuncBackword,
	_KEY_CTRL_B: keyFuncBackword,
	"k":         keyFuncPrevious,
	_KEY_UP:     keyFuncPrevious,
	_KEY_CTRL_P: keyFuncPrevious,
	"l":         keyFuncForward,
	" ":         keyFuncForward,
	_KEY_RIGHT:  keyFuncForward,
	_KEY_CTRL_F: keyFuncForward,
	"0":         keyFuncGoBeginOfLine,
	"^":         keyFuncGoBeginOfLine,
	_KEY_CTRL_A: keyFuncGoBeginOfLine,
	"$":         keyFuncGoEndOfLine,
	_KEY_CTRL_E: keyFuncGoEndOfLine,
	"<":         keyFuncGoBeginOfFile,
	">":         keyFuncGoEndOfFile,
	"G":         keyFuncGoEndOfFile,
	"p":         keyFuncPasteAfter,
	"P":         keyFuncPasteBefore,
	"x":         keyFuncRemoveByte,
	"d":         keyFuncDelete,
	_KEY_DEL:    keyFuncRemoveByte,
	"w":         keyFuncWriteFile,
	"r":         keyFuncReplaceByte,
	_KEY_CTRL_L: keyFuncRepaint,
	"R":         keyFuncChangeMode,
	"v":         keyFuncMarking,
	"y":         keyFuncYank,
	"/":         keyFuncSearchForward,
	"n":         keyFuncSearchForwardNext,
	"?":         keyFuncSearchBackward,
	"N":         keyFuncSearchBackwardNext,
}
