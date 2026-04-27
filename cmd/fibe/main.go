package main

import (
	"os"
)

func main() {
	if err := RootCmd().Execute(); err != nil {
		outputError(err)
		os.Exit(1)
	}
}
