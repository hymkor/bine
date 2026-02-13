package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/hymkor/binview"
)

var version string = "snapshot"

func main() {
	fmt.Fprintf(os.Stderr, "Bine %s-%s-%s\n", version, runtime.GOOS, runtime.GOARCH)
	if err := bine.Run(os.Args[1:]); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
