package app

import "github.com/ahmadnaufalhakim/gochs/internal/gui"

func Run() error {
	for {
		result, err := gui.Run()
		if err != nil {
			return err
		}
		if result == gui.Exit {
			return nil
		}

		runLocalGame()
	}
}
