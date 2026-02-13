package bine

import (
	"io"
	"strings"
	"testing"

	"github.com/nyaosorg/go-ttyadapter/auto"

	. "github.com/hymkor/binview/internal/large"
)

func _insert(exp string) func(*Application) error {
	return func(app *Application) error {
		return app.InsertExp(exp)
	}
}

func _append(exp string) func(*Application) error {
	return func(app *Application) error {
		return app.AppendExp(exp)
	}
}

func try(
	t *testing.T,
	source string,
	expect string,
	funcs ...func(app *Application) error) {

	ALLOC_SIZE = 4
	app, err := NewApplication(
		&auto.Pilot{Text: []string{}},
		strings.NewReader(source),
		io.Discard,
		"dummy")
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	app.buffer.ReadAll()

	for _, f := range funcs {
		if err := f(app); err != nil {
			app.Close()
			t.Fatal(err.Error())
			return
		}
	}

	var output strings.Builder
	app.buffer.WriteTo(&output)
	app.Close()
	if outputStr := output.String(); outputStr != expect {
		t.Fatalf("expect '%s' but '%s'", expect, outputStr)
	}
}

func TestNoModify(t *testing.T) {
	const sample = "1234567890"
	try(t, sample, sample)
}

func TestKeyFuncRemoveByte(t *testing.T) {
	try(t, "1234567890", "234567890",
		keyFuncRemoveByte)
}

func TestKeyFuncInsertByte(t *testing.T) {
	try(t, "1234567890", "\0001234567890",
		_insert("0x00"))
}

func TestForwardAndRemove(t *testing.T) {
	try(t, "1234567890", "134567890",
		keyFuncForward,
		keyFuncRemoveByte)
}

func TestRemoveEOF(t *testing.T) {
	try(t, "1234567890", "123456789",
		keyFuncGoEndOfFile,
		keyFuncRemoveByte)
}

func TestRemoveEndOfLine(t *testing.T) {
	try(t, "0123456789ABCDEFG", "0123456789ABCDEG",
		keyFuncGoEndOfLine,
		keyFuncRemoveByte)
}

func TestEndOfLine2Backward2Delete(t *testing.T) {
	try(t, "0123456789ABCDEFG", "0123456789ABCDFG",
		keyFuncGoEndOfLine,
		keyFuncBackword,
		keyFuncRemoveByte)
}

func TestKeyFuncNext(t *testing.T) {
	try(t, "0123456789ABCDEFGHI", "0123456789ABCDEFHI",
		keyFuncNext,
		keyFuncRemoveByte)
}

func TestCutAndPasteAfter(t *testing.T) {
	try(t, "0123456789ABCDEFGHI", "1023456789ABCDEFGHI",
		keyFuncRemoveByte,
		keyFuncPasteAfter)
}

func TestCutAndPasteBefore(t *testing.T) {
	try(t, "0123456789ABCDEFGHI", "F0123456789ABCDEGHI",
		keyFuncGoEndOfLine,
		keyFuncRemoveByte,
		keyFuncGoBeginOfFile,
		keyFuncPasteBefore)
}

func TestKeyFuncPrevious(t *testing.T) {
	try(t, "0123456789ABCDEF"+"GHIJ", "01J23456789ABCDEFGHI",
		keyFuncGoEndOfFile,
		keyFuncRemoveByte,
		keyFuncPrevious,
		keyFuncPasteBefore)
}

func TestKeyFuncBeginOfLine(t *testing.T) {
	try(t, "0123456789ABCDEF"+"GHIJ", "0123456789ABCDEF"+"HIJ",
		keyFuncGoEndOfFile,
		keyFuncGoBeginOfLine,
		keyFuncRemoveByte)
}

func TestKeyFuncAddByte(t *testing.T) {
	try(t, "0123456789ABCDEF", "0\000123456789ABCDEF",
		_append("0x00"))
}

func TestEmptyData(t *testing.T) {
	_, err := NewApplication(
		&auto.Pilot{Text: []string{}},
		strings.NewReader(""),
		io.Discard,
		"dummy")
	if err == nil {
		t.Fatal("with empty data,NewApplication must return error, but it did not")
		return
	}
}

func TestEndOfLineOnShortData(t *testing.T) {
	try(t, "012345", "01234\0005",
		keyFuncGoEndOfLine,
		_insert("0x00"))
}

func TestInsertAndUndo(t *testing.T) {
	try(t, "012345", "012345",
		_insert(`"abcdef"`),
		keyFuncUndo)
}

func TestAppendAndUndo(t *testing.T) {
	try(t, "012345", "012345",
		keyFuncGoEndOfLine,
		_append(`"abcdef"`),
		keyFuncUndo)
}
