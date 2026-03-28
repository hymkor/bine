package bine

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/keys"

	"github.com/nyaosorg/go-ttyadapter"

	"github.com/hymkor/bine/internal/ansi"
)

type scheme struct {
	Cursor [2]string
	Select [2]string
	Cell   [2]string
	Status string
}

var colorScheme = &scheme{
	Cursor: [2]string{"\x1B[39;49;1;7m", "\x1B[27;22m"},
	Select: [2]string{"\x1B[39;44m", "\x1B[39;49m"},
	Cell:   [2]string{"\x1B[39;49;22m", ""},
	Status: "\x1B[0;33;22m",
}

var monoScheme = &scheme{
	Cursor: [2]string{"\x1B[1;7m", "\x1B[27;22m"},
	Select: [2]string{"\x1B[22;7m", "\x1B[27m"},
	Cell:   [2]string{"\x1B[22m", ""},
	Status: "\x1B[0m",
}

func (scheme *scheme) getline(out io.Writer, prompt string, defaultStr string, history readline.IHistory) (string, error) {
	editor := readline.Editor{
		Writer:  out,
		Default: defaultStr,
		Cursor:  65535,
		PromptWriter: func(w io.Writer) (int, error) {
			fmt.Fprintf(w, "\r%s%s%s", scheme.Status, prompt, ansi.EraseLine)
			return 2, nil
		},
		LineFeedWriter: func(readline.Result, io.Writer) (int, error) { return 0, nil },
		History:        history,
	}
	defer io.WriteString(out, ansi.CursorOff)
	editor.BindKey(keys.CtrlG, readline.CmdInterrupt)
	editor.BindKey(keys.Escape+keys.CtrlG, readline.CmdInterrupt)
	text, err := editor.ReadLine(context.Background())
	if err == readline.CtrlC {
		return "", errors.New("Canceled")
	}
	return text, err
}

func (scheme *scheme) ask(tty1 ttyadapter.Tty, out io.Writer, message string) (string, error) {
	fmt.Fprintf(out, "%s\r%s%s %s", scheme.Status, message, ansi.EraseLine, ansi.CursorOn)
	ch, err := tty1.GetKey()
	io.WriteString(out, ansi.CursorOff)
	return ch, err
}

func (scheme *scheme) yesNo(tty1 ttyadapter.Tty, out io.Writer, message string) bool {
	ch, err := scheme.ask(tty1, out, message)
	if err == nil && (ch == "y" || ch == "Y") {
		fmt.Fprintf(out, " %s ", ch)
		return true
	}
	return false
}
