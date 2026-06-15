package main

import (
	"fmt"
	"io"
)

func outputHint(w io.Writer, hint string) {
	if effectiveOutput() == "table" && hint != "" {
		fmt.Fprintln(w, hint)
	}
}
