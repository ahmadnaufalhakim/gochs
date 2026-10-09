package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ahmadnaufalhakim/gochs/internal/app"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
	author  = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Version: %s\nCommit: %s\nDate: %s\nAuthor: %s\n", version, commit, date, author)
		os.Exit(0)
	}

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
