package app

import (
	"fmt"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
)

func Run() {
	fmt.Println("Hello from gochs!")

	var r chess.Renderer
	r.Theme = chess.DEFAULT
	r.Perspective = chess.White

	var b chess.Board
	b.Reset()

	r.Print(b)
}
