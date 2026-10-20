package main

import (
	"fmt"
	"os"

	"github.com/ahmadnaufalhakim/gochs/internal/app"
)

func main() {
	if err := app.Run3D(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
