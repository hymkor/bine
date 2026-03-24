package bine

import (
	"context"
	"io"
	"os"
	"os/signal"

	"github.com/nyaosorg/go-inline-animation"

	"github.com/hymkor/bine/internal/large"
)

type search struct {
	data    []byte
	reverse bool
}

func (s search) Exec(cursor *large.Pointer, out io.Writer) (*large.Pointer, string) {
	if len(s.data) <= 0 {
		return cursor, ""
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	p := cursor.Clone()
	out.Write([]byte{' '})
	end := animation.Dots.Progress(out)
	defer end()
	for {
		if err := ctx.Err(); err != nil {
			return cursor, "Search interrupted"
		}
		if err := s.Walk(p); err != nil {
			if err == io.EOF {
				return cursor, "not found"
			}
			return cursor, err.Error()
		}
		if p.Value() == s.data[0] {
			q := p.Clone()
			i := 1
			for {
				if i >= len(s.data) {
					return p, ""
				}
				if err := q.Next(); err != nil {
					break
				}
				if q.Value() != s.data[i] {
					break
				}
				i++
			}
		}
	}
}

func (s search) Walk(p *large.Pointer) error {
	if s.reverse {
		return p.Prev()
	}
	return p.Next()
}

func (s search) Reverse() search {
	return search{data: s.data, reverse: !s.reverse}
}

func keyFuncSearchForward(app *Application) error {
	expStr := app.searchWord
	if expStr == "" {
		expStr = "0x"
	}
	var err error
	expStr, err = app.scheme.getline(app.out, "search forward>", expStr, nil)
	if err != nil {
		app.message = err.Error()
		return nil
	}
	exp, err := evalExpression(expStr, app.encoding)
	if err != nil {
		app.message = err.Error()
		return nil
	}
	app.searchWord = expStr
	app.search = search{data: exp, reverse: false}
	app.cursor, app.message = app.search.Exec(app.cursor, app.out)
	return nil
}

func keyFuncSearchBackward(app *Application) error {
	expStr := app.searchWord
	if expStr == "" {
		expStr = "0x"
	}
	var err error
	expStr, err = app.scheme.getline(app.out, "search backward>", expStr, nil)
	if err != nil {
		app.message = err.Error()
		return nil
	}
	exp, err := evalExpression(expStr, app.encoding)
	if err != nil {
		return err
	}
	app.search = search{data: exp, reverse: true}
	app.cursor, app.message = app.search.Exec(app.cursor, app.out)
	return nil
}

func keyFuncSearchForwardNext(app *Application) error {
	app.cursor, app.message = app.search.Exec(app.cursor, app.out)
	return nil
}

func keyFuncSearchBackwardNext(app *Application) error {
	app.cursor, app.message = app.search.Reverse().Exec(app.cursor, app.out)
	return nil
}
